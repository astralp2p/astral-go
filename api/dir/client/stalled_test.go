package dir

import (
	"context"
	"errors"
	"io"
	"testing"
	"time"

	"github.com/astralp2p/astral-go/api/dir"
	"github.com/astralp2p/astral-go/astral"
	"github.com/astralp2p/astral-go/lib/astrald"
	"github.com/astralp2p/astral-go/lib/query"
	"github.com/astralp2p/astral-go/lib/routing"
)

// opsRouter routes queries in-process to the ops of inner.
type opsRouter struct{ inner astral.Router }

func (r *opsRouter) RouteQuery(ctx *astral.Context, q *astral.InFlightQuery) (astral.Conn, error) {
	return query.RouteInFlight(ctx, r.inner, q)
}

func (*opsRouter) GuestID() *astral.Identity { return nil }
func (*opsRouter) HostID() *astral.Identity  { return nil }

// stalledClient returns a client whose target accepts every dir op, reads all
// the caller sends, and never answers, as a node does that hangs past the
// accept.
func stalledClient(t *testing.T) *Client {
	t.Helper()
	ops := routing.NewOpRouter()
	for _, name := range []string{
		dir.MethodAliasMap, dir.MethodApplyFilters, dir.MethodFilters,
		dir.MethodGetAlias, dir.MethodResolve, dir.MethodSetAlias,
	} {
		op, err := routing.NewOp(func(_ *astral.Context, q *routing.IncomingQuery) error {
			conn := q.AcceptRaw()
			_, _ = io.Copy(io.Discard, conn)
			return conn.Close()
		})
		if err != nil {
			t.Fatalf("NewOp %s: %v", name, err)
		}
		if err := ops.AddOp(name, op); err != nil {
			t.Fatalf("AddOp %s: %v", name, err)
		}
	}
	return New(nil, astrald.New(&opsRouter{inner: ops}))
}

// Every call stops waiting on a target that accepted and never answers when
// its ctx ends, and says that ctx ended rather than handing back an empty
// answer.
func TestEveryCallEndsWithCtx(t *testing.T) {
	c := stalledClient(t)
	c.EnableCache = true
	id := astral.GenerateIdentity()

	calls := map[string]func(*astral.Context) error{
		"AliasMap":        func(ctx *astral.Context) error { _, err := c.AliasMap(ctx); return err },
		"ApplyFilters":    func(ctx *astral.Context) error { _, err := c.ApplyFilters(ctx, id, "swarm"); return err },
		"Filters":         func(ctx *astral.Context) error { _, err := c.Filters(ctx); return err },
		"GetAlias":        func(ctx *astral.Context) error { _, err := c.GetAlias(ctx, id); return err },
		"ResolveIdentity": func(ctx *astral.Context) error { _, err := c.ResolveIdentity(ctx, "alias"); return err },
		"SetAlias":        func(ctx *astral.Context) error { return c.SetAlias(ctx, id, "alias") },
	}

	for name, call := range calls {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			ctx, cancel := astral.NewContext(nil).WithTimeout(100 * time.Millisecond)
			defer cancel()

			done := make(chan error, 1)
			go func() { done <- call(ctx) }()

			select {
			case err := <-done:
				if !errors.Is(err, context.DeadlineExceeded) {
					t.Fatalf("err = %v; want context.DeadlineExceeded", err)
				}
			case <-time.After(5 * time.Second):
				t.Fatal("still waiting on the node 5 s after ctx ended")
			}
		})
	}
}

// An alias lookup that ctx cut short leaves nothing in the cache.
func TestGetAliasCachesNothingWhenCtxEnds(t *testing.T) {
	c := stalledClient(t)
	c.EnableCache = true
	id := astral.GenerateIdentity()

	ctx, cancel := astral.NewContext(nil).WithTimeout(100 * time.Millisecond)
	defer cancel()
	_, _ = c.GetAlias(ctx, id)

	if alias, ok := c.aliasCache.Get(id.String()); ok {
		t.Fatalf("the cache holds %q for a lookup that never got an answer", alias)
	}
}
