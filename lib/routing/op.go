package routing

import (
	"errors"
	"fmt"
	"io"
	"reflect"
	"runtime/debug"
	"time"

	"github.com/astralp2p/astral-go/astral"
	"github.com/astralp2p/astral-go/lib/query"
)

/*
Op wraps a function of this signature:

 	type Op func(ctx *astral.Context, q *Query) error

With an optional third argument that can be a struct with fields representing operation arguments:

	type ArgType struct {
		ID      *astral.Identity
        Message string
	}

	type Op func(ctx *astral.Context, q *Query, args ArgType) error

Then allows this function to be invoked via the Call() function:

	op.Call(ctx, query, map[string]string{
		"id": "033e9ce4e8f0e68f7db49ffb6b9eecc10605f3f3fcb3c630545887749ab515b9c7",
		"message": "Hello, world!",
    }) error
*/

type Op struct {
	LogFunc func(report *Report)
	opFunc  reflect.Value // the function to be called
	argType reflect.Type  // optional type of the 3rd func argument
}

var ErrInvalidSignature = errors.New("invalid function signature")

// NewOp wraps a function as an Op. fn has to have one of the following signatures:
// - func(*astral.Context, *shell.Query) error
// - func(*astral.Context, *shell.Query, ArgType) error
// ArgType can be a struct or a pointer to a struct with all fields exported.
func NewOp(fn any) (*Op, error) {
	var v = reflect.ValueOf(fn)
	var t = v.Type()
	var op = &Op{opFunc: v}

	// check function signature
	switch {
	case v.Kind() != reflect.Func:
		return nil, errors.New("fn must be a function")
	case t.NumIn() != 2 && t.NumIn() != 3:
		return nil, ErrInvalidSignature
	case t.In(0) != reflect.TypeOf(&astral.Context{}):
		return nil, ErrInvalidSignature
	case t.In(1).Kind() != reflect.Ptr:
		return nil, ErrInvalidSignature
	case t.In(1).Elem() != typeOfIncomingQuery:
		return nil, ErrInvalidSignature
	case t.NumOut() != 1 || !t.Out(0).Implements(reflect.TypeOf((*error)(nil)).Elem()):
		return nil, ErrInvalidSignature
	case t.NumIn() == 3:
		at := v.Type().In(2)
		if at.Kind() == reflect.Pointer {
			at = at.Elem()
		}
		if at.Kind() != reflect.Struct {
			return nil, ErrInvalidSignature
		}
		op.argType = at
	}

	return op, nil
}

// RouteQuery dispatches the query to the op in a new goroutine with a detached
// context; the caller blocks until the op calls Accept/Reject or the 5-second
// deadline expires. A panicking op is contained: it is reported as *ErrPanic on
// the Report and the query is rejected with astral.CodeInternalError.
func (op *Op) RouteQuery(ctx *astral.Context, q *astral.InFlightQuery, remoteWriter io.WriteCloser) (io.WriteCloser, error) {
	var origin string
	if o, found := q.Extra.Get("origin"); found {
		origin = o.(string)
	}
	var inQuery = NewIncomingQuery(q.Query, remoteWriter, origin)

	go func() {
		// reject the query at the end in case the op did not respond to it, will noop if it did.
		defer inQuery.Reject()

		// detach from the parent context
		ctx := ctx.Detach()

		var report = Report{Query: q.Query}
		var start = time.Now()

		// call the op
		report.Err = op.invoke(ctx, inQuery)
		report.Time = time.Since(start)

		// why: a panicking op is an internal error, not a plain rejection. This runs
		// before the deferred Reject above, so it wins the resolve; an op that already
		// accepted or rejected keeps its own outcome.
		//
		// The two calls cover the two shapes a panic can take, and exactly one of them
		// does anything. Panicking before resolving: RejectWithCode wins the CAS and
		// abandon finds no connection. Panicking after accepting: the CAS fails and the
		// caller rightly keeps its connection, but the op is gone and nothing else would
		// ever close it, so abandon does.
		var errPanic *ErrPanic
		if errors.As(report.Err, &errPanic) {
			inQuery.RejectWithCode(astral.CodeInternalError)
			inQuery.abandon(report.Err)
		}

		if op.LogFunc != nil {
			op.LogFunc(&report)
		}
	}()

	return inQuery.await(ctx)
}

// invoke calls the op function, binding query arguments first. A panic in the op
// is recovered and returned as *ErrPanic.
//
// why: the goroutine in RouteQuery is the root of its own stack, so no caller can
// recover for it. Containing the panic here rather than in that goroutine keeps the
// recovery independent of defer ordering.
func (op *Op) invoke(ctx *astral.Context, q *IncomingQuery) (err error) {
	defer func() {
		if r := recover(); r != nil {
			err = &ErrPanic{Value: r, Stack: debug.Stack()}
		}
	}()

	var fnArgs = []reflect.Value{
		reflect.ValueOf(ctx),
		reflect.ValueOf(q),
	}

	if op.argType != nil {
		argVal := reflect.New(op.argType)

		_, params := query.Parse(q.QueryString())

		arg := query.EditValue(argVal)

		err := arg.SetMany(params)
		if err != nil {
			return err
		}

		for _, spec := range op.ArgumentSpecs() {
			if !spec.Required {
				continue
			}

			if _, found := params[spec.Name]; !found {
				return fmt.Errorf("field required: %s", spec.Name)
			}
		}

		if op.opFunc.Type().In(2).Kind() == reflect.Pointer {
			fnArgs = append(fnArgs, argVal)
		} else {
			fnArgs = append(fnArgs, argVal.Elem())
		}
	}

	var ret = op.opFunc.Call(fnArgs)[0]

	if ret.IsNil() {
		return nil
	}

	return ret.Interface().(error)
}

// ArgumentSpecs returns info about the op arguments
func (op *Op) ArgumentSpecs() (args []query.FieldSpec) {
	if op.argType == nil {
		return
	}

	return query.EditValue(reflect.New(op.argType)).Spec()
}
