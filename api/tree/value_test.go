package tree

import (
	"errors"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/astralp2p/astral-go/astral"
)

// TestValue_Clear covers the unbound clear path: Clear resets the cached value to
// the zero T where Set cannot — a typed-nil pointer slips past Set's any(v) == nil
// guard. Exercised against the local fallback, with no backing node.
func TestValue_Clear(t *testing.T) {
	var v Value[*astral.String8]

	s := astral.String8("active")
	if err := v.Set(nil, &s); err != nil {
		t.Fatalf("set: %v", err)
	}
	if v.Get() == nil {
		t.Fatal("value not set before clear")
	}

	if err := v.Clear(nil); err != nil {
		t.Fatalf("clear: %v", err)
	}
	if v.Get() != nil {
		t.Errorf("value not cleared: got %v", v.Get())
	}
}

// Bind's notification goroutine applies updates under the mutex, so a reader calling
// Get concurrently with node notifications races neither cached nor queue.
// Meaningful under -race.
func TestValue_Bind_NotificationsRaceGet(t *testing.T) {
	const count = 200

	node := &notifyNode{updates: make(chan astral.Object, 1)}
	seed := astral.String8("seed")
	node.updates <- &seed

	var value Value[*astral.String8]
	if err := value.Bind(nil, node); err != nil {
		t.Fatalf("bind: %v", err)
	}

	var wg sync.WaitGroup

	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := range count {
			s := astral.String8(strconv.Itoa(i))
			node.updates <- &s
		}
		close(node.updates)
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		for range count {
			_ = value.Get()
		}
	}()

	wg.Wait()

	// the goroutine drains what is left in the channel; the last notification wins
	want := strconv.Itoa(count - 1)
	for deadline := time.Now().Add(time.Second); ; {
		if got := value.Get(); got != nil && string(*got) == want {
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("want %v, got %v", want, value.Get())
		}
		time.Sleep(time.Millisecond)
	}
}

// notifyNode hands Bind a channel of notifications the test drives. Bind uses Get alone.
type notifyNode struct {
	updates chan astral.Object
}

var _ Node = &notifyNode{}

func (n *notifyNode) Get(*astral.Context, bool) (<-chan astral.Object, error) {
	return n.updates, nil
}

func (n *notifyNode) Set(*astral.Context, astral.Object) error { return errUnsupported }
func (n *notifyNode) Delete(*astral.Context) error             { return errUnsupported }

func (n *notifyNode) Sub(*astral.Context) (map[string]Node, error) { return nil, errUnsupported }

func (n *notifyNode) Create(*astral.Context, string) (Node, error) { return nil, errUnsupported }

var errUnsupported = errors.New("not supported")
