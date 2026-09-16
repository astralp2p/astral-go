package tree

import (
	"context"
	"testing"
	"time"

	"github.com/astralp2p/astral-go/astral"
)

// stubNode is a Node whose Set always succeeds and whose change notification is
// delivered by the test, standing in for the asynchronous hop chain a real node's
// update crosses before it reaches a bound Value.
type stubNode struct {
	updates chan astral.Object
	sets    []astral.Object
}

func (n *stubNode) Get(ctx *astral.Context, follow bool) (<-chan astral.Object, error) {
	return n.updates, nil
}

func (n *stubNode) Set(ctx *astral.Context, object astral.Object) error {
	n.sets = append(n.sets, object)
	return nil
}

func (n *stubNode) Delete(ctx *astral.Context) error { return nil }

func (n *stubNode) Sub(ctx *astral.Context) (map[string]Node, error) { return nil, nil }

func (n *stubNode) Create(ctx *astral.Context, name string) (Node, error) { return nil, nil }

// bindStub returns a Value bound to a stubNode already holding initial.
func bindStub(t *testing.T, initial string) (*Value[*astral.String8], *stubNode, *astral.Context) {
	t.Helper()

	ctx := astral.NewContext(context.Background())
	node := &stubNode{updates: make(chan astral.Object, 8)}

	first := astral.String8(initial)
	node.updates <- &first

	var v Value[*astral.String8]
	if err := v.Bind(ctx, node); err != nil {
		t.Fatalf("bind: %v", err)
	}
	if got := v.Get(); got == nil || string(*got) != initial {
		t.Fatalf("bind did not seed the cache: got %v", got)
	}

	return &v, node, ctx
}

// TestValue_SetRefreshesCache covers the write-then-read race: Set writes through to
// the node and the node's notification arrives later, so the cache must be refreshed
// by Set itself or the caller reads its own pre-Set value.
func TestValue_SetRefreshesCache(t *testing.T) {
	v, node, ctx := bindStub(t, "before")

	next := astral.String8("after")
	if err := v.Set(ctx, &next); err != nil {
		t.Fatalf("set: %v", err)
	}

	if len(node.sets) != 1 {
		t.Fatalf("node.Set called %d times, want 1", len(node.sets))
	}
	got := v.Get()
	if got == nil || string(*got) != "after" {
		t.Errorf("Get after Set = %v, want \"after\"", got)
	}
}

// TestValue_SetFollowEmitsOnce covers the other half: refreshing the cache must not
// add a queue push of its own, or every follower sees the write twice.
func TestValue_SetFollowEmitsOnce(t *testing.T) {
	v, node, ctx := bindStub(t, "before")

	follow := v.Follow(ctx)
	if got := <-follow; got == nil || string(*got) != "before" {
		t.Fatalf("Follow seed = %v, want \"before\"", got)
	}

	next := astral.String8("after")
	if err := v.Set(ctx, &next); err != nil {
		t.Fatalf("set: %v", err)
	}

	// the node acknowledges the write the way a real node does, asynchronously
	node.updates <- &next

	select {
	case got := <-follow:
		if got == nil || string(*got) != "after" {
			t.Fatalf("Follow update = %v, want \"after\"", got)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("Follow lost the update")
	}

	select {
	case got := <-follow:
		t.Errorf("Follow doubled the update: extra %v", got)
	case <-time.After(200 * time.Millisecond):
	}
}
