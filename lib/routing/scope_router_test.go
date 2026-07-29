package routing

import (
	"errors"
	"io"
	"testing"

	"github.com/astralp2p/astral-go/astral"
	"github.com/astralp2p/astral-go/lib/query"
)

// ScopeRouter splits an op name on its first dot and forwards to the matching
// scope with that prefix removed, so a scoped router sees the bare op name.
// Anything unmatched falls through to the root.

// recordingRouter captures the query it was handed.
type recordingRouter struct {
	seen *astral.InFlightQuery
	err  error
}

func (r *recordingRouter) RouteQuery(_ *astral.Context, q *astral.InFlightQuery, _ io.WriteCloser) (io.WriteCloser, error) {
	r.seen = q
	if r.err != nil {
		return nil, r.err
	}
	return nopWriteCloser{}, nil
}

func TestScopeRouter_ForwardsToAMatchedScopeWithThePrefixStripped(t *testing.T) {
	scope := &recordingRouter{}
	router := NewScopeRouter(&recordingRouter{})
	router.Add("objects", scope)

	if _, err := routeTo(t, router, "objects.read"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if scope.seen == nil {
		t.Fatal("want the scope reached, got nothing")
	}
	if scope.seen.QueryString != "read" {
		t.Fatalf("QueryString: want %q, got %q", "read", scope.seen.QueryString)
	}
}

// The arguments survive the rewrite: only the scope prefix is removed.
func TestScopeRouter_PreservesArgumentsAcrossTheRewrite(t *testing.T) {
	scope := &recordingRouter{}
	router := NewScopeRouter(&recordingRouter{})
	router.Add("objects", scope)

	routeTo(t, router, "objects.read?id=abc&offset=2")

	want := "read?id=abc&offset=2"
	if scope.seen.QueryString != want {
		t.Fatalf("want %q, got %q", want, scope.seen.QueryString)
	}
}

// The rewritten query keeps the identity of the original: same nonce, caller,
// target, and Extra. A scoped router must see the same query, not a new one.
func TestScopeRouter_PreservesQueryIdentityAndExtra(t *testing.T) {
	scope := &recordingRouter{}
	router := NewScopeRouter(&recordingRouter{})
	router.Add("objects", scope)

	id := astral.GenerateIdentity()
	original := astral.Launch(query.New(id, id, "objects.read", nil))
	original.Extra.Set("origin", astral.OriginNetwork)

	router.RouteQuery(astral.NewContext(nil), original, nopWriteCloser{})

	if scope.seen.Nonce != original.Nonce {
		t.Fatalf("Nonce: want %v, got %v", original.Nonce, scope.seen.Nonce)
	}
	if !scope.seen.Caller.IsEqual(original.Caller) {
		t.Fatal("Caller: want the original caller")
	}
	if !scope.seen.Target.IsEqual(original.Target) {
		t.Fatal("Target: want the original target")
	}

	origin, found := scope.seen.Extra.Get("origin")
	if !found || origin != astral.OriginNetwork {
		t.Fatalf("Extra[origin]: want %q, got %v", astral.OriginNetwork, origin)
	}
}

func TestScopeRouter_FallsThroughToRoot(t *testing.T) {
	cases := []struct {
		name        string
		queryString string
		wantAtRoot  string
	}{
		{"no dot", "ping", "ping"},
		{"unknown scope", "missing.read", "missing.read"},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			root := &recordingRouter{}
			router := NewScopeRouter(root)
			router.Add("objects", &recordingRouter{})

			routeTo(t, router, c.queryString)

			if root.seen == nil {
				t.Fatal("want the root reached, got nothing")
			}
			// an unmatched query reaches the root unrewritten
			if root.seen.QueryString != c.wantAtRoot {
				t.Fatalf("want %q at the root, got %q", c.wantAtRoot, root.seen.QueryString)
			}
		})
	}
}

// A nil root becomes a hard-rejecting NilRouter rather than a nil dereference.
func TestScopeRouter_NilRootRejects(t *testing.T) {
	router := NewScopeRouter(nil)

	_, err := routeTo(t, router, "ping")

	if !errors.Is(err, &astral.ErrRejected{}) {
		t.Fatalf("want ErrRejected, got %v", err)
	}
}

func TestScopeRouter_Remove(t *testing.T) {
	root := &recordingRouter{}
	scope := &recordingRouter{}
	router := NewScopeRouter(root)
	router.Add("objects", scope)

	router.Remove("objects")

	routeTo(t, router, "objects.read")

	if scope.seen != nil {
		t.Fatal("want the removed scope unreached")
	}
	if root.seen == nil {
		t.Fatal("want the query to fall through to the root")
	}
}

// AddScopedOp mounts at the root for an empty scope, and creates the scope's
// OpRouter on demand otherwise.
func TestScopeRouter_AddScopedOp(t *testing.T) {
	t.Run("empty scope mounts at the root", func(t *testing.T) {
		router := NewScopeRouter(NewOpRouter())

		if err := router.AddScopedOp("", "ping", newTestOp(t)); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !router.HasRoute("ping") {
			t.Fatal("want the op mounted at the root")
		}
	})

	t.Run("named scope is created on demand", func(t *testing.T) {
		router := NewScopeRouter(NewOpRouter())

		if err := router.AddScopedOp("objects", "read", newTestOp(t)); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !router.HasRoute("objects.read") {
			t.Fatal("want the op mounted under the scope")
		}

		// a second op joins the scope that now exists
		if err := router.AddScopedOp("objects", "write", newTestOp(t)); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !router.HasRoute("objects.write") {
			t.Fatal("want the second op mounted under the same scope")
		}
	})
}

// Mounting an op requires an OpRouter on the receiving side; any other router
// is refused rather than silently ignored.
func TestScopeRouter_AddScopedOpRequiresAnOpRouter(t *testing.T) {
	t.Run("root", func(t *testing.T) {
		router := NewScopeRouter(&recordingRouter{})

		if err := router.AddScopedOp("", "ping", newTestOp(t)); err == nil {
			t.Fatal("want an error for a non-OpRouter root, got nil")
		}
	})

	t.Run("scope", func(t *testing.T) {
		router := NewScopeRouter(NewOpRouter())
		router.Add("objects", &recordingRouter{})

		if err := router.AddScopedOp("objects", "read", newTestOp(t)); err == nil {
			t.Fatal("want an error for a non-OpRouter scope, got nil")
		}
	})
}

func TestScopeRouter_HasRoute(t *testing.T) {
	root := NewOpRouter()
	root.AddOp("ping", newTestOp(t))

	scope := NewOpRouter()
	scope.AddOp("read", newTestOp(t))

	router := NewScopeRouter(root)
	router.Add("objects", scope)

	cases := []struct {
		name string
		want bool
	}{
		{"ping", true},
		{"objects.read", true},
		{"objects.missing", false},
		{"missing", false},
		{"missing.read", false},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := router.HasRoute(c.name); got != c.want {
				t.Fatalf("want %v, got %v", c.want, got)
			}
		})
	}
}

// A router that cannot answer HasRoute reports false rather than panicking.
func TestScopeRouter_HasRouteWithoutARouteChecker(t *testing.T) {
	router := NewScopeRouter(&recordingRouter{})
	router.Add("objects", &recordingRouter{})

	if router.HasRoute("ping") {
		t.Fatal("want false for a root that cannot report routes")
	}
	if router.HasRoute("objects.read") {
		t.Fatal("want false for a scope that cannot report routes")
	}
}

// Spec flattens the tree: scoped names are prefixed, and internal ops — those
// beginning with "." — are withheld from the root's contribution.
func TestScopeRouter_Spec(t *testing.T) {
	root := NewOpRouter()
	root.AddOp("ping", newTestOp(t))
	root.AddOp(".spec", newTestOp(t))

	scope := NewOpRouter()
	scope.AddOp("read", newTestOp(t))

	router := NewScopeRouter(root)
	router.Add("objects", scope)

	names := map[string]bool{}
	for _, spec := range router.Spec() {
		names[spec.Name] = true
	}

	if !names["ping"] {
		t.Fatalf("want the root op present, got %v", names)
	}
	if !names["objects.read"] {
		t.Fatalf("want the scoped op prefixed, got %v", names)
	}
	if names[".spec"] {
		t.Fatalf("want the internal op withheld, got %v", names)
	}
}

// A scope that cannot describe itself contributes nothing rather than failing
// the whole spec.
func TestScopeRouter_SpecSkipsRoutersWithoutASpec(t *testing.T) {
	root := NewOpRouter()
	root.AddOp("ping", newTestOp(t))

	router := NewScopeRouter(root)
	router.Add("opaque", &recordingRouter{})

	specs := router.Spec()

	if len(specs) != 1 || specs[0].Name != "ping" {
		t.Fatalf("want only the root op, got %v", specs)
	}
}
