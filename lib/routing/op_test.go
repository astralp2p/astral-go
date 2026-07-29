package routing

import (
	"errors"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/astralp2p/astral-go/astral"
	"github.com/astralp2p/astral-go/lib/query"
)

// Op wraps a handler function and dispatches a query to it. NewOp is the
// signature gate: a method that does not match is rejected here, which is what
// lets AddStruct scan an arbitrary struct and mount only the real ops.

type opArgs struct {
	Name  string
	Count int
}

// helpers -------------------------------------------------------------------

// routeTo dispatches a query string through the router and returns the result.
func routeTo(t *testing.T, router astral.Router, queryString string) (io.WriteCloser, error) {
	t.Helper()

	id := astral.GenerateIdentity()
	q := astral.Launch(query.New(id, id, queryString, nil))

	return router.RouteQuery(astral.NewContext(nil), q, nopWriteCloser{})
}

type nopWriteCloser struct{}

func (nopWriteCloser) Write(p []byte) (int, error) { return len(p), nil }
func (nopWriteCloser) Close() error                { return nil }

// NewOp signature gate ------------------------------------------------------

func TestNewOp_AcceptsValidSignatures(t *testing.T) {
	cases := []struct {
		name string
		fn   any
	}{
		{"two arguments", func(*astral.Context, *IncomingQuery) error { return nil }},
		{"three arguments, struct", func(*astral.Context, *IncomingQuery, opArgs) error { return nil }},
		{"three arguments, pointer to struct", func(*astral.Context, *IncomingQuery, *opArgs) error { return nil }},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			op, err := NewOp(c.fn)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if op == nil {
				t.Fatal("want an op, got nil")
			}
		})
	}
}

func TestNewOp_RejectsInvalidSignatures(t *testing.T) {
	cases := []struct {
		name string
		fn   any
	}{
		{"not a function", 42},
		{"no arguments", func() error { return nil }},
		{"one argument", func(*astral.Context) error { return nil }},
		{"four arguments", func(*astral.Context, *IncomingQuery, opArgs, int) error { return nil }},
		{"wrong context type", func(string, *IncomingQuery) error { return nil }},
		// any non-pointer fails the kind check before the type check; a plain
		// struct stands in for IncomingQuery, which cannot be copied
		{"non-pointer second argument", func(*astral.Context, opArgs) error { return nil }},
		{"wrong second argument type", func(*astral.Context, *opArgs) error { return nil }},
		{"no return value", func(*astral.Context, *IncomingQuery) {}},
		{"non-error return value", func(*astral.Context, *IncomingQuery) string { return "" }},
		{"two return values", func(*astral.Context, *IncomingQuery) (string, error) { return "", nil }},
		{"non-struct third argument", func(*astral.Context, *IncomingQuery, int) error { return nil }},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			op, err := NewOp(c.fn)
			if err == nil {
				t.Fatalf("want an error for %s, got nil", c.name)
			}
			if op != nil {
				t.Fatal("want no op on error, got one")
			}
		})
	}
}

// A signature failure other than "not a function" reports ErrInvalidSignature,
// so a caller can tell the two apart.
func TestNewOp_ReportsErrInvalidSignature(t *testing.T) {
	_, err := NewOp(func(*astral.Context) error { return nil })

	if !errors.Is(err, ErrInvalidSignature) {
		t.Fatalf("want ErrInvalidSignature, got %v", err)
	}
}

// dispatch ------------------------------------------------------------------

// An accepting op hands the caller a writer.
func TestOp_RouteQueryAcceptsTheQuery(t *testing.T) {
	op, err := NewOp(func(_ *astral.Context, q *IncomingQuery) error {
		conn := q.AcceptRaw()
		defer conn.Close()
		return nil
	})
	if err != nil {
		t.Fatalf("NewOp: %v", err)
	}

	w, err := routeTo(t, op, "op")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if w == nil {
		t.Fatal("want a writer, got nil")
	}
}

// A rejecting op reports ErrRejected to the caller.
func TestOp_RouteQueryRejectsTheQuery(t *testing.T) {
	op, err := NewOp(func(_ *astral.Context, q *IncomingQuery) error {
		return q.Reject()
	})
	if err != nil {
		t.Fatalf("NewOp: %v", err)
	}

	w, err := routeTo(t, op, "op")

	if !errors.Is(err, &astral.ErrRejected{}) {
		t.Fatalf("want ErrRejected, got %v", err)
	}
	if w != nil {
		t.Fatal("want no writer on rejection, got one")
	}
}

// An op that returns without resolving the query is rejected on its behalf, so
// the caller is never left waiting on a handler that forgot to answer.
func TestOp_RouteQueryAutoRejectsAnUnresolvedQuery(t *testing.T) {
	op, err := NewOp(func(*astral.Context, *IncomingQuery) error { return nil })
	if err != nil {
		t.Fatalf("NewOp: %v", err)
	}

	w, err := routeTo(t, op, "op")

	if !errors.Is(err, &astral.ErrRejected{}) {
		t.Fatalf("want ErrRejected, got %v", err)
	}
	if w != nil {
		t.Fatal("want no writer, got one")
	}
}

// An op that fails before resolving is likewise auto-rejected.
func TestOp_RouteQueryAutoRejectsOnHandlerError(t *testing.T) {
	op, err := NewOp(func(*astral.Context, *IncomingQuery) error {
		return errors.New("handler failed")
	})
	if err != nil {
		t.Fatalf("NewOp: %v", err)
	}

	_, err = routeTo(t, op, "op")

	if !errors.Is(err, &astral.ErrRejected{}) {
		t.Fatalf("want ErrRejected, got %v", err)
	}
}

// LogFunc observes every finished call, including the handler's error.
func TestOp_LogFuncReceivesTheReport(t *testing.T) {
	handlerErr := errors.New("handler failed")
	reports := make(chan *Report, 1)

	op, err := NewOp(func(_ *astral.Context, q *IncomingQuery) error {
		q.Reject()
		return handlerErr
	})
	if err != nil {
		t.Fatalf("NewOp: %v", err)
	}
	op.LogFunc = func(r *Report) { reports <- r }

	routeTo(t, op, "op?a=1")

	select {
	case report := <-reports:
		if !errors.Is(report.Err, handlerErr) {
			t.Fatalf("Err: want %v, got %v", handlerErr, report.Err)
		}
		if report.Query == nil || report.Query.QueryString != "op?a=1" {
			t.Fatalf("Query: want the dispatched query, got %v", report.Query)
		}
		if report.Time < 0 {
			t.Fatalf("Time: want a non-negative duration, got %v", report.Time)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("LogFunc was not called")
	}
}

// The origin recorded on the in-flight query reaches the handler.
func TestOp_RouteQueryPassesTheOriginThrough(t *testing.T) {
	origins := make(chan string, 1)

	op, err := NewOp(func(_ *astral.Context, q *IncomingQuery) error {
		origins <- q.Origin()
		return q.Reject()
	})
	if err != nil {
		t.Fatalf("NewOp: %v", err)
	}

	id := astral.GenerateIdentity()
	q := astral.Launch(query.New(id, id, "op", nil))
	q.Extra.Set("origin", astral.OriginNetwork)

	op.RouteQuery(astral.NewContext(nil), q, nopWriteCloser{})

	select {
	case got := <-origins:
		if got != astral.OriginNetwork {
			t.Fatalf("want %q, got %q", astral.OriginNetwork, got)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("the handler did not run")
	}
}

// argument binding ----------------------------------------------------------

// Query parameters are bound to the third argument, by value or by pointer.
func TestOp_BindsArguments(t *testing.T) {
	t.Run("struct", func(t *testing.T) {
		got := make(chan opArgs, 1)

		op, err := NewOp(func(_ *astral.Context, q *IncomingQuery, args opArgs) error {
			got <- args
			return q.Reject()
		})
		if err != nil {
			t.Fatalf("NewOp: %v", err)
		}

		routeTo(t, op, "op?name=bob&count=7")

		args := <-got
		if args.Name != "bob" || args.Count != 7 {
			t.Fatalf("want {bob 7}, got %+v", args)
		}
	})

	t.Run("pointer to struct", func(t *testing.T) {
		got := make(chan *opArgs, 1)

		op, err := NewOp(func(_ *astral.Context, q *IncomingQuery, args *opArgs) error {
			got <- args
			return q.Reject()
		})
		if err != nil {
			t.Fatalf("NewOp: %v", err)
		}

		routeTo(t, op, "op?name=bob&count=7")

		args := <-got
		if args == nil {
			t.Fatal("want the arguments allocated, got nil")
		}
		if args.Name != "bob" || args.Count != 7 {
			t.Fatalf("want {bob 7}, got %+v", *args)
		}
	})
}

// An unknown parameter is ignored rather than failing the call.
func TestOp_IgnoresUnknownArguments(t *testing.T) {
	got := make(chan opArgs, 1)

	op, err := NewOp(func(_ *astral.Context, q *IncomingQuery, args opArgs) error {
		got <- args
		return q.Reject()
	})
	if err != nil {
		t.Fatalf("NewOp: %v", err)
	}

	routeTo(t, op, "op?name=bob&unknown=x")

	if args := <-got; args.Name != "bob" {
		t.Fatalf("want the known argument bound, got %+v", args)
	}
}

// A missing required argument fails the call before the handler body runs.
func TestOp_RequiredArgumentIsEnforced(t *testing.T) {
	type requiredArgs struct {
		Name string `query:"required"`
	}

	var called bool
	op, err := NewOp(func(_ *astral.Context, q *IncomingQuery, args requiredArgs) error {
		called = true
		return q.Reject()
	})
	if err != nil {
		t.Fatalf("NewOp: %v", err)
	}

	errs := make(chan error, 1)
	op.LogFunc = func(r *Report) { errs <- r.Err }

	routeTo(t, op, "op")

	select {
	case err := <-errs:
		if err == nil {
			t.Fatal("want an error for the missing required argument, got nil")
		}
		if !strings.Contains(err.Error(), "name") {
			t.Fatalf("want the field name in %q", err.Error())
		}
	case <-time.After(5 * time.Second):
		t.Fatal("LogFunc was not called")
	}

	if called {
		t.Fatal("want the handler body skipped, got it called")
	}
}

// A required argument that is supplied lets the call through.
func TestOp_RequiredArgumentSatisfied(t *testing.T) {
	type requiredArgs struct {
		Name string `query:"required"`
	}

	got := make(chan string, 1)
	op, err := NewOp(func(_ *astral.Context, q *IncomingQuery, args requiredArgs) error {
		got <- args.Name
		return q.Reject()
	})
	if err != nil {
		t.Fatalf("NewOp: %v", err)
	}

	routeTo(t, op, "op?name=bob")

	if name := <-got; name != "bob" {
		t.Fatalf("want %q, got %q", "bob", name)
	}
}

// A malformed argument value fails the call rather than binding a zero value.
func TestOp_MalformedArgumentFailsTheCall(t *testing.T) {
	var called bool
	op, err := NewOp(func(_ *astral.Context, q *IncomingQuery, args opArgs) error {
		called = true
		return q.Reject()
	})
	if err != nil {
		t.Fatalf("NewOp: %v", err)
	}

	errs := make(chan error, 1)
	op.LogFunc = func(r *Report) { errs <- r.Err }

	routeTo(t, op, "op?count=not-a-number")

	select {
	case err := <-errs:
		if err == nil {
			t.Fatal("want a conversion error, got nil")
		}
	case <-time.After(5 * time.Second):
		t.Fatal("LogFunc was not called")
	}

	if called {
		t.Fatal("want the handler body skipped, got it called")
	}
}

// ArgumentSpecs ---------------------------------------------------------------

func TestOp_ArgumentSpecs(t *testing.T) {
	type specArgs struct {
		Name   string `query:"required"`
		Count  int
		Hidden string `query:"skip"`
	}

	op, err := NewOp(func(*astral.Context, *IncomingQuery, specArgs) error { return nil })
	if err != nil {
		t.Fatalf("NewOp: %v", err)
	}

	specs := op.ArgumentSpecs()

	byName := map[string]query.FieldSpec{}
	for _, spec := range specs {
		byName[spec.Name] = spec
	}

	if len(byName) != 2 {
		t.Fatalf("want 2 specs, got %d (%v)", len(byName), specs)
	}
	if !byName["name"].Required {
		t.Fatal("name.Required: want true, got false")
	}
	if byName["count"].Required {
		t.Fatal("count.Required: want false, got true")
	}
	if _, found := byName["hidden"]; found {
		t.Fatal("a skip-tagged argument must not appear in the specs")
	}
}

// An op's published spec carries each float argument at its own width. A float64
// argument previously advertised "float32", so a caller reading .spec could size or
// validate its input to 32-bit precision while the field parsed and held 64.
func TestOp_ArgumentSpecs_FloatWidths(t *testing.T) {
	type floatArgs struct {
		Ratio  float32
		Amount float64
	}

	op, err := NewOp(func(*astral.Context, *IncomingQuery, floatArgs) error { return nil })
	if err != nil {
		t.Fatalf("NewOp: %v", err)
	}

	byName := map[string]query.FieldSpec{}
	for _, spec := range op.ArgumentSpecs() {
		byName[spec.Name] = spec
	}

	if got := byName["ratio"].Type; got != "float32" {
		t.Fatalf("ratio.Type: want %q, got %q", "float32", got)
	}
	if got := byName["amount"].Type; got != "float64" {
		t.Fatalf("amount.Type: want %q, got %q", "float64", got)
	}
}

// An op without a third argument advertises no parameters.
func TestOp_ArgumentSpecsWithoutArguments(t *testing.T) {
	op, err := NewOp(func(*astral.Context, *IncomingQuery) error { return nil })
	if err != nil {
		t.Fatalf("NewOp: %v", err)
	}

	if specs := op.ArgumentSpecs(); len(specs) != 0 {
		t.Fatalf("want no specs, got %v", specs)
	}
}
