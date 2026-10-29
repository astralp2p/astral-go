package routing

import (
	"errors"
	"fmt"
	"io"
	"reflect"
	"runtime/debug"
	"slices"
	"strings"
	"time"

	"github.com/astralp2p/astral-go/astral"
	"github.com/astralp2p/astral-go/astral/channel"
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
// deadline expires, which rejects the query with astral.CodeTimeout. An op that
// returns without answering, or panics, is an internal error: the query is
// rejected with astral.CodeInternalError, and a panic is reported as *ErrPanic
// on the Report.
func (op *Op) RouteQuery(ctx *astral.Context, q *astral.InFlightQuery, remoteWriter io.WriteCloser) (io.WriteCloser, error) {
	var origin string
	if o, found := q.Extra.Get("origin"); found {
		origin = o.(string)
	}
	var inQuery = NewIncomingQuery(q.Query, remoteWriter, origin)

	go func() {
		// why an internal error: an op that returns without accepting or rejecting
		// failed to answer, which is the node's fault and not a refusal of the caller.
		// note: a noop when the op answered.
		defer inQuery.RejectWithCode(astral.CodeInternalError)

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
		if err == nil {
			err = checkRequired(arg, op.ArgumentSpecs(), params)
		}

		// why: an argument error is the caller's mistake, so it reaches the caller
		// as an error_message naming the argument, not as a bare reject code.
		// note: only an op that takes in or out speaks the channel protocol.
		if err != nil && op.speaksChannel() {
			ch := q.Accept(channel.WithFormats(params["in"], params["out"]))
			defer ch.Close()
			ch.Send(astral.Err(err))
			return err
		}

		// why: a raw-stream op (objects.read, gateway.node_route, shell.shell)
		// answers bytes its caller reads as payload, so its argument error is a
		// rejection; the query itself is what is wrong.
		if err != nil {
			q.RejectWithCode(astral.CodeInvalidQuery)
			return err
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

// speaksChannel reports whether the op takes an in or out format argument.
func (op *Op) speaksChannel() bool {
	for _, spec := range op.ArgumentSpecs() {
		if spec.Name == "in" || spec.Name == "out" {
			return true
		}
	}
	return false
}

// ArgumentSpecs returns info about the op arguments
func (op *Op) ArgumentSpecs() (args []query.FieldSpec) {
	if op.argType == nil {
		return
	}

	return query.EditValue(reflect.New(op.argType)).Spec()
}

// maxUnknownNamed bounds how many unknown keys an answer names.
//
// why: the keys are the caller's own, and an error_message is a String16, so an
// unbounded list could outgrow the one answer the caller gets.
const maxUnknownNamed = 8

// checkRequired names every required argument params lacks and, beside them,
// every key arg binds to no field. The channel format keys in and out are never
// named unknown.
//
// why: SetMany skips an unknown key, so a caller still sending a renamed
// argument learns the old name only from the missing argument it causes.
func checkRequired(arg *query.Editor, specs []query.FieldSpec, params map[string]string) error {
	var missing []string
	for _, spec := range specs {
		if _, found := params[spec.Name]; spec.Required && !found {
			missing = append(missing, spec.Name)
		}
	}
	if len(missing) == 0 {
		return nil
	}
	err := fmt.Errorf("%w: %s", query.ErrArgumentRequired, strings.Join(missing, ", "))

	var unknown []string
	for key := range params {
		if _, ferr := arg.Field(key); errors.Is(ferr, query.ErrFieldNotFound) && key != "in" && key != "out" {
			unknown = append(unknown, key)
		}
	}
	if len(unknown) == 0 {
		return err
	}
	slices.Sort(unknown)
	if len(unknown) > maxUnknownNamed {
		unknown = append(unknown[:maxUnknownNamed], "…")
	}
	return fmt.Errorf("%w; %w: %s", err, query.ErrUnknownArgument, strings.Join(unknown, ", "))
}
