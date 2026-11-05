package nearby

import (
	"errors"
	"testing"
	"time"

	"github.com/astralp2p/astral-go/api/auth"
	"github.com/astralp2p/astral-go/astral"
	"github.com/astralp2p/astral-go/lib/astrald"
	"github.com/astralp2p/astral-go/lib/query"
	"github.com/astralp2p/astral-go/lib/routing"
)

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

// scanNode records each call and answers reply, standing in for the node.
type scanNode struct {
	called chan struct{}
	reply  astral.Object
}

func (n *scanNode) Scan(_ *astral.Context, q *routing.IncomingQuery) error {
	n.called <- struct{}{}

	ch := q.Accept()
	defer ch.Close()

	return ch.Send(n.reply)
}

// clientForScan routes the op under the literal name astrald serves, so the
// client's constant must match it.
func clientForScan(t *testing.T, node *scanNode) *Client {
	t.Helper()

	ops := routing.NewOpRouter()
	op, err := routing.NewOp(node.Scan)
	if err != nil {
		t.Fatalf("NewOp: %v", err)
	}
	if err = ops.AddOp("nearby.scan", op); err != nil {
		t.Fatalf("AddOp: %v", err)
	}

	id := astral.GenerateIdentity()
	return New(id, astrald.New(&localRouter{inner: ops, id: id}))
}

func TestScanReturnsTheAck(t *testing.T) {
	node := &scanNode{called: make(chan struct{}, 1), reply: &astral.Ack{}}

	if err := clientForScan(t, node).Scan(astral.NewContext(nil)); err != nil {
		t.Fatalf("Scan: %v", err)
	}

	select {
	case <-node.called:
	case <-time.After(5 * time.Second):
		t.Fatal("the op was never called")
	}
}

// A refusal the op sends reaches the caller as an error carrying its message.
func TestScanPassesTheRefusal(t *testing.T) {
	node := &scanNode{called: make(chan struct{}, 1), reply: auth.ErrNotPermitted}

	err := clientForScan(t, node).Scan(astral.NewContext(nil))
	if err == nil {
		t.Fatal("want the refusal, got nil")
	}
	if err.Error() != auth.ErrNotPermitted.Error() {
		t.Fatalf("want %q, got %q", auth.ErrNotPermitted, err)
	}
	var msg *astral.ErrorMessage
	if !errors.As(err, &msg) {
		t.Fatalf("want an *astral.ErrorMessage, got %T", err)
	}
}
