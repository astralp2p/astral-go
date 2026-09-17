package user

import (
	"testing"
	"time"

	"github.com/astralp2p/astral-go/api/user"
	"github.com/astralp2p/astral-go/astral"
	"github.com/astralp2p/astral-go/lib/astrald"
	"github.com/astralp2p/astral-go/lib/query"
	"github.com/astralp2p/astral-go/lib/routing"
)

// astrald names the target of user.sync_with `identity` and declares it required.
// The op binder skips a key it does not declare, so a client that names the
// argument anything else builds a query the node rejects before the handler runs,
// and nothing on the client side reports why. The op below mirrors astrald's
// declaration, so routing the real client through it pins the wire key: rename the
// argument and the required field goes unbound.

type syncWithArgs struct {
	Identity *astral.Identity `query:"required"`
}

// syncWithNode records what the op bound and acks, standing in for the node.
type syncWithNode struct {
	bound chan *astral.Identity
}

func (n *syncWithNode) SyncWith(_ *astral.Context, q *routing.IncomingQuery, args syncWithArgs) error {
	n.bound <- args.Identity

	ch := q.Accept()
	defer ch.Close()

	return ch.Send(&astral.Ack{})
}

// localRouter routes queries in-process, so the real client talks to the real op
// over a pipe with no node involved.
type localRouter struct {
	inner astral.Router
	id    *astral.Identity
}

func (r *localRouter) RouteQuery(ctx *astral.Context, q *astral.InFlightQuery) (astral.Conn, error) {
	return query.RouteInFlight(ctx, r.inner, q)
}

func (r *localRouter) GuestID() *astral.Identity { return r.id }
func (r *localRouter) HostID() *astral.Identity  { return r.id }

func clientForSyncWith(t *testing.T, node *syncWithNode) *Client {
	t.Helper()

	ops := routing.NewOpRouter()
	op, err := routing.NewOp(node.SyncWith)
	if err != nil {
		t.Fatalf("NewOp: %v", err)
	}
	if err = ops.AddOp(user.OpSyncWith, op); err != nil {
		t.Fatalf("AddOp: %v", err)
	}

	id := astral.GenerateIdentity()
	return New(id, astrald.New(&localRouter{inner: ops, id: id}))
}

// The argument the client sends binds to the op's required identity field, and the
// op's ack reaches the caller as a nil error.
func TestSyncWithBindsTheRequiredIdentity(t *testing.T) {
	node := &syncWithNode{bound: make(chan *astral.Identity, 1)}
	client := clientForSyncWith(t, node)

	target := astral.GenerateIdentity()

	if err := client.SyncWith(astral.NewContext(nil), target); err != nil {
		t.Fatalf("SyncWith: %v", err)
	}

	select {
	case got := <-node.bound:
		if got == nil {
			t.Fatal("want the identity bound, got nil: the client named the argument something the op does not declare")
		}
		if !got.IsEqual(target) {
			t.Fatalf("want %s bound, got %s", target, got)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("the op was never called")
	}
}
