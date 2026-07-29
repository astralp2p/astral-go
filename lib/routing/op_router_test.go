package routing

import (
	"errors"
	"testing"

	"github.com/astralp2p/astral-go/astral"
)

// OpRouter is the flat dispatch table: it matches the query's op name against
// registered handlers. AddStruct is how an app mounts a whole struct, and its
// filtering rule — skip anything that is not a valid op — is what lets an app
// struct carry ordinary methods alongside its ops.

// opHost carries a mix of valid ops and methods that must not be mounted.
type opHost struct{}

func (opHost) Ping(_ *astral.Context, q *IncomingQuery) error       { return q.Reject() }
func (opHost) ReadObject(_ *astral.Context, q *IncomingQuery) error { return q.Reject() }
func (opHost) WithArgs(_ *astral.Context, q *IncomingQuery, _ opArgs) error {
	return q.Reject()
}

// not ops
func (opHost) Helper() string       { return "" }
func (opHost) Compute(a int) int    { return a }
func (opHost) unexported() string   { return "" }
func (opHost) Mismatched(int) error { return nil }

func newTestOp(t *testing.T) *Op {
	t.Helper()

	op, err := NewOp(func(_ *astral.Context, q *IncomingQuery) error { return q.Reject() })
	if err != nil {
		t.Fatalf("NewOp: %v", err)
	}
	return op
}

func TestOpRouter_DispatchesOnTheOpName(t *testing.T) {
	router := NewOpRouter()
	if err := router.AddOp("ping", newTestOp(t)); err != nil {
		t.Fatalf("AddOp: %v", err)
	}

	// the op rejects, which proves it was reached
	_, err := routeTo(t, router, "ping")
	if !errors.Is(err, &astral.ErrRejected{}) {
		t.Fatalf("want ErrRejected from the mounted op, got %v", err)
	}
}

// Arguments are not part of the match: only the path segment before "?" is.
func TestOpRouter_MatchesThePathIgnoringArguments(t *testing.T) {
	router := NewOpRouter()
	router.AddOp("ping", newTestOp(t))

	_, err := routeTo(t, router, "ping?a=1&b=2")
	if !errors.Is(err, &astral.ErrRejected{}) {
		t.Fatalf("want the op reached, got %v", err)
	}
}

func TestOpRouter_UnknownOpIsNotFound(t *testing.T) {
	router := NewOpRouter()
	router.AddOp("ping", newTestOp(t))

	_, err := routeTo(t, router, "missing")
	if !errors.Is(err, &astral.ErrRouteNotFound{}) {
		t.Fatalf("want ErrRouteNotFound, got %v", err)
	}
}

func TestOpRouter_AddOpRejectsDuplicates(t *testing.T) {
	router := NewOpRouter()

	if err := router.AddOp("ping", newTestOp(t)); err != nil {
		t.Fatalf("first AddOp: unexpected error: %v", err)
	}
	if err := router.AddOp("ping", newTestOp(t)); err == nil {
		t.Fatal("want an error on a duplicate name, got nil")
	}
}

// OpRouter is flat: it cannot host a scoped route, so the scoped adder refuses
// any non-empty scope rather than flattening it.
func TestOpRouter_AddScopedOp(t *testing.T) {
	router := NewOpRouter()

	if err := router.AddScopedOp("", "ping", newTestOp(t)); err != nil {
		t.Fatalf("empty scope: unexpected error: %v", err)
	}
	if !router.HasRoute("ping") {
		t.Fatal("want the op mounted at the root")
	}

	if err := router.AddScopedOp("scope", "ping", newTestOp(t)); err == nil {
		t.Fatal("want an error for a non-empty scope, got nil")
	}
}

func TestOpRouter_HasRoute(t *testing.T) {
	router := NewOpRouter()
	router.AddOp("ping", newTestOp(t))

	if !router.HasRoute("ping") {
		t.Fatal("HasRoute(ping): want true, got false")
	}
	if router.HasRoute("missing") {
		t.Fatal("HasRoute(missing): want false, got true")
	}
}

func TestOpRouter_GetOp(t *testing.T) {
	router := NewOpRouter()
	op := newTestOp(t)
	router.AddOp("ping", op)

	got, err := router.GetOp("ping")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != op {
		t.Fatal("want the registered op, got another")
	}

	if _, err := router.GetOp("missing"); err == nil {
		t.Fatal("want an error for an unknown op, got nil")
	}
}

func TestOpRouter_RemoveOp(t *testing.T) {
	router := NewOpRouter()
	router.AddOp("ping", newTestOp(t))

	if err := router.RemoveOp("ping"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if router.HasRoute("ping") {
		t.Fatal("want the op removed")
	}

	if err := router.RemoveOp("ping"); err == nil {
		t.Fatal("want an error removing an absent op, got nil")
	}
}

// AddStruct mounts every method with a valid op signature, snake_cased, and
// silently skips the rest.
func TestOpRouter_AddStruct(t *testing.T) {
	router := NewOpRouter(&opHost{})

	for _, name := range []string{"ping", "read_object", "with_args"} {
		if !router.HasRoute(name) {
			t.Fatalf("want %q mounted", name)
		}
	}

	for _, name := range []string{"helper", "compute", "unexported", "mismatched"} {
		if router.HasRoute(name) {
			t.Fatalf("want %q skipped, got it mounted", name)
		}
	}
}

func TestOpRouter_AddStructRequiresAPointerToStruct(t *testing.T) {
	router := NewOpRouter()

	cases := []struct {
		name string
		arg  any
	}{
		{"struct by value", opHost{}},
		{"pointer to a non-struct", new(int)},
		{"plain value", 42},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if err := router.AddStruct(c.arg); err == nil {
				t.Fatalf("want an error for %T, got nil", c.arg)
			}
		})
	}
}

// AddStructPrefix mounts only the methods carrying the prefix, and strips it
// from the mounted name.
func TestOpRouter_AddStructPrefix(t *testing.T) {
	router := NewOpRouter()

	if err := router.AddStructPrefix(&opHost{}, "Read"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !router.HasRoute("object") {
		t.Fatal("want ReadObject mounted as \"object\"")
	}
	if router.HasRoute("read_object") {
		t.Fatal("want the prefix stripped from the mounted name")
	}
	if router.HasRoute("ping") {
		t.Fatal("want methods without the prefix skipped")
	}
}

func TestOpRouter_Spec(t *testing.T) {
	router := NewOpRouter(&opHost{})

	specs := router.Spec()

	byName := map[string]OpSpec{}
	for _, spec := range specs {
		byName[spec.Name] = spec
	}

	if len(byName) != 3 {
		t.Fatalf("want 3 specs, got %d (%v)", len(byName), specs)
	}

	withArgs, found := byName["with_args"]
	if !found {
		t.Fatalf("want with_args in the specs, got %v", byName)
	}
	if len(withArgs.Parameters) != 2 {
		t.Fatalf("want 2 parameters, got %v", withArgs.Parameters)
	}

	ping, found := byName["ping"]
	if !found {
		t.Fatalf("want ping in the specs, got %v", byName)
	}
	if len(ping.Parameters) != 0 {
		t.Fatalf("want no parameters for ping, got %v", ping.Parameters)
	}
}

func TestOpRouter_SpecOfAnEmptyRouter(t *testing.T) {
	if specs := NewOpRouter().Spec(); len(specs) != 0 {
		t.Fatalf("want no specs, got %v", specs)
	}
}
