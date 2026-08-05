package tree

import (
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/astralp2p/astral-go/api/tree"
	"github.com/astralp2p/astral-go/astral"
	"github.com/astralp2p/astral-go/lib/astrald"
	"github.com/astralp2p/astral-go/lib/query"
	"github.com/astralp2p/astral-go/lib/routing"
)

// Create rides tree.set in batch mode. The server opens the path before it starts
// reading, so a failure there is written while the client is still expected to be
// writing — which is why Create must read its response, and why it cannot do so by
// sending the terminator inline.

// localRouter routes queries in-process, so the real client talks to the real server
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

// clientFor mounts ops behind a tree client.
func clientFor(t *testing.T, node tree.Node) *Client {
	t.Helper()

	ops := routing.NewOpRouter()
	op, err := routing.NewOp(NewNodeOps(node).Set)
	if err != nil {
		t.Fatalf("NewOp: %v", err)
	}
	if err := ops.AddOp(tree.MethodSet, op); err != nil {
		t.Fatalf("AddOp: %v", err)
	}

	id := astral.GenerateIdentity()
	return New(id, astrald.New(&localRouter{inner: ops, id: id}))
}

// failingNode refuses to create children.
type failingNode struct{ fakeNode }

var errRefused = errors.New("permission denied")

func (n *failingNode) Create(*astral.Context, string) (tree.Node, error) {
	return nil, errRefused
}

func (n *failingNode) Sub(*astral.Context) (map[string]tree.Node, error) {
	return map[string]tree.Node{}, nil
}

// mustFinish bounds the call: the naive fix — sending the terminator inline and then
// reading — deadlocks here, and a deadlock must fail this test rather than hang the
// package until the go test timeout.
func mustFinish(t *testing.T, d time.Duration, fn func()) {
	t.Helper()

	done := make(chan struct{})
	go func() { defer close(done); fn() }()

	select {
	case <-done:
	case <-time.After(d):
		t.Fatalf("call did not finish within %v", d)
	}
}

func TestNodeCreate_ReportsAServerSideFailure(t *testing.T) {
	root := &failingNode{}
	client := clientFor(t, root)

	var node tree.Node
	var err error

	mustFinish(t, 10*time.Second, func() {
		node, err = client.Root().Create(astral.NewContext(nil), "child")
	})

	if err == nil {
		t.Fatal("want an error for a child the server refused to create, got nil")
	}
	if !strings.Contains(err.Error(), errRefused.Error()) {
		t.Errorf("want the server's reason, got %v", err)
	}
	if node != nil {
		t.Errorf("want no node, got %v", node)
	}
}

func TestNodeCreate_SucceedsAndCreatesTheChild(t *testing.T) {
	root := &fakeNode{}
	client := clientFor(t, root)

	var node tree.Node
	var err error

	mustFinish(t, 10*time.Second, func() {
		node, err = client.Root().Create(astral.NewContext(nil), "child")
	})

	if err != nil {
		t.Fatalf("want no error, got %v", err)
	}
	if node == nil {
		t.Fatal("want a node, got nil")
	}
	if _, ok := root.subs["child"]; !ok {
		t.Errorf("want the child created server-side, got %v", root.subs)
	}
}

// Create must not store a value: it opens the node and nothing more.
func TestNodeCreate_DoesNotSetAValue(t *testing.T) {
	root := &fakeNode{}
	client := clientFor(t, root)

	mustFinish(t, 10*time.Second, func() {
		client.Root().Create(astral.NewContext(nil), "child")
	})

	child := root.subs["child"]
	if child == nil {
		t.Fatal("want the child created")
	}
	if child.sets != 0 {
		t.Errorf("want no value stored, got %d Set calls", child.sets)
	}
}
