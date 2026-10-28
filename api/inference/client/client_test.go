package inference

import (
	"testing"

	"github.com/astralp2p/astral-go/astral"
	"github.com/astralp2p/astral-go/lib/astrald"
	"github.com/astralp2p/astral-go/lib/query"
	"github.com/astralp2p/astral-go/lib/routing"
)

// Tests in this package route the real client through in-process ops standing
// in for a provider or a node.

type localRouter struct {
	inner astral.Router
	id    *astral.Identity
}

func (r *localRouter) RouteQuery(ctx *astral.Context, q *astral.InFlightQuery) (astral.Conn, error) {
	return query.RouteInFlight(ctx, r.inner, q)
}

func (r *localRouter) GuestID() *astral.Identity { return r.id }
func (r *localRouter) HostID() *astral.Identity  { return r.id }

// clientFor returns a client whose target answers ops, and the identity that
// stands for both the target and the local node.
func clientFor(t *testing.T, ops map[string]any) (*Client, *astral.Identity) {
	t.Helper()
	router := routing.NewOpRouter()
	for name, fn := range ops {
		op, err := routing.NewOp(fn)
		if err != nil {
			t.Fatalf("NewOp %s: %v", name, err)
		}
		if err := router.AddOp(name, op); err != nil {
			t.Fatalf("AddOp %s: %v", name, err)
		}
	}
	id := astral.GenerateIdentity()
	return New(id, astrald.New(&localRouter{inner: router, id: id})), id
}
