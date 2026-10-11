package tree

import (
	"context"
	"fmt"
	"sort"
	"sync"
	"testing"

	"github.com/astralp2p/astral-go/astral"
)

// memNode is an in-memory node safe for concurrent use. Create answers ErrAlreadyExists for an
// existing name. A memNode with a gate holds the first gate.n calls of Sub until all of them
// have listed the subnodes.
type memNode struct {
	mu   sync.Mutex
	subs map[string]Node
	gate *gate
}

// gate releases its waiters once n of them arrive. Arrivals past n pass through.
type gate struct {
	n    int
	open chan struct{}
}

func newMemNode() *memNode { return &memNode{subs: map[string]Node{}} }

func (n *memNode) Get(*astral.Context, bool) (<-chan astral.Object, error) {
	return nil, ErrUnsupported
}

func (n *memNode) Set(*astral.Context, astral.Object) error { return ErrUnsupported }

func (n *memNode) Delete(*astral.Context) error { return ErrUnsupported }

// Sub lists the subnodes, then holds the caller at the gate.
func (n *memNode) Sub(*astral.Context) (map[string]Node, error) {
	n.mu.Lock()
	subs := make(map[string]Node, len(n.subs))
	for name, sub := range n.subs {
		subs[name] = sub
	}
	var wait chan struct{}
	if n.gate != nil && n.gate.n > 0 {
		n.gate.n--
		wait = n.gate.open
		if n.gate.n == 0 {
			close(wait)
		}
	}
	n.mu.Unlock()

	if wait != nil {
		<-wait
	}
	return subs, nil
}

func (n *memNode) Create(_ *astral.Context, name string) (Node, error) {
	n.mu.Lock()
	defer n.mu.Unlock()
	if _, ok := n.subs[name]; ok {
		return nil, ErrAlreadyExists
	}
	sub := newMemNode()
	n.subs[name] = sub
	return sub, nil
}

// gatedQuery runs one Query per path in parallel. Every caller lists the root before any caller
// creates under it, so all callers but one lose the race for the first segment.
func gatedQuery(ctx *astral.Context, root *memNode, paths []string) []error {
	root.gate = &gate{n: len(paths), open: make(chan struct{})}

	errs := make([]error, len(paths))
	var wg sync.WaitGroup
	for i, path := range paths {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, errs[i] = Query(ctx, root, path, true)
		}()
	}
	wg.Wait()
	return errs
}

// Parallel binders sharing a prefix all land on their full path, and none binds under the root.
func TestQuery_ParallelSharedPrefix(t *testing.T) {
	ctx := astral.NewContext(context.Background())
	root := newMemNode()

	const binders = 8
	paths := make([]string, binders)
	for i := range paths {
		paths[i] = fmt.Sprintf("/mod/m%d/settings", i)
	}

	for i, err := range gatedQuery(ctx, root, paths) {
		if err != nil {
			t.Fatalf("Query %s: %v", paths[i], err)
		}
	}

	top, _ := root.Sub(ctx)
	if len(top) != 1 || top["mod"] == nil {
		t.Errorf("root holds %v, want [mod]", sortedNames(top))
	}
	for _, path := range paths {
		if _, err := Query(ctx, root, path, false); err != nil {
			t.Errorf("%s: %v", path, err)
		}
	}
}

// Query without create answers an error for a missing segment and creates nothing.
func TestQuery_MissingWithoutCreate(t *testing.T) {
	ctx := astral.NewContext(context.Background())
	root := newMemNode()

	if _, err := Query(ctx, root, "/mod/tcp", false); err == nil {
		t.Error("Query of a missing path without create succeeded")
	}
	if len(root.subs) != 0 {
		t.Errorf("Query without create created %v", sortedNames(root.subs))
	}
}

func sortedNames(m map[string]Node) []string {
	names := make([]string, 0, len(m))
	for name := range m {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}
