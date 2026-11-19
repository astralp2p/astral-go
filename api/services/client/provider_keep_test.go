package services

import (
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/astralp2p/astral-go/api/services"
	"github.com/astralp2p/astral-go/astral"
	"github.com/astralp2p/astral-go/lib/routing"
)

// Tests in this file hold the keeper: a Provider whose binding the node ends
// advertises again by itself.
// Spec: .ai/system/protocols/services/ops/services.advertise.md#liveness

// advertiseNode stands in for a node's services.advertise: it acks every
// binding and hands the test a switch that ends it from the node side.
type advertiseNode struct {
	calls  atomic.Int32
	opened chan chan struct{}
}

func newAdvertiseNode() *advertiseNode {
	return &advertiseNode{opened: make(chan chan struct{}, 16)}
}

func (n *advertiseNode) advertise(_ *astral.Context, q *routing.IncomingQuery, _ struct{ Services string }) error {
	n.calls.Add(1)
	ch := q.Accept()
	defer ch.Close()
	if err := ch.Send(&astral.Ack{}); err != nil {
		return err
	}
	ended := make(chan struct{})
	go func() {
		defer close(ended)
		for {
			if _, err := ch.Receive(); err != nil {
				return
			}
		}
	}()
	end := make(chan struct{})
	n.opened <- end
	select {
	case <-end:
	case <-ended:
	}
	return nil
}

func (n *advertiseNode) client(t *testing.T) *Client {
	return clientFor(t, map[string]any{services.MethodAdvertise: n.advertise})
}

// next returns the switch of the next binding the node acks.
func (n *advertiseNode) next(t *testing.T, within time.Duration) chan struct{} {
	t.Helper()
	select {
	case end := <-n.opened:
		return end
	case <-time.After(within):
		t.Fatalf("no advertise within %v", within)
		return nil
	}
}

func newTestProvider() *Provider {
	return NewProvider(map[string]OfferingFunc{"player": func(*astral.Context, *astral.Identity) (*services.Update, error) {
		return &services.Update{Available: true}, nil
	}})
}

// keepers counts the running keep goroutines.
func keepers() int {
	buf := make([]byte, 1<<20)
	n := runtime.Stack(buf, true)
	return strings.Count(string(buf[:n]), ".(*Provider).keep(")
}

// within polls cond for up to d.
func within(d time.Duration, cond func() bool) bool {
	deadline := time.Now().Add(d)
	for !cond() {
		if time.Now().After(deadline) {
			return false
		}
		time.Sleep(5 * time.Millisecond)
	}
	return true
}

// The node closes the advertise channel; within 1 s the Provider has a new
// binding open.
func TestProviderReadvertisesWhenTheNodeEndsTheBinding(t *testing.T) {
	n := newAdvertiseNode()
	c := n.client(t)
	p := newTestProvider()
	ctx, cancel := astral.NewContext(nil).WithCancel()
	defer cancel()
	defer p.Close()

	if err := p.Serve(ctx, c); err != nil {
		t.Fatalf("Serve: %v", err)
	}
	first := n.next(t, time.Second)
	old := p.current()
	if old == nil {
		t.Fatal("no binding after Serve")
	}

	start := time.Now()
	close(first)
	n.next(t, time.Second)
	reopened := within(time.Second, func() bool {
		b := p.current()
		return b != nil && b != old
	})
	if !reopened {
		t.Fatal("the Provider has no new binding within 1 s of the node ending the old one")
	}
	if d := time.Since(start); d > time.Second {
		t.Fatalf("re-advertised after %v; want within 1 s", d)
	}
	if got := n.calls.Load(); got != 2 {
		t.Fatalf("advertised %d times; want 2", got)
	}
}

// The registrar hook runs Serve on every registration; three runs leave one
// keeper, and one node-side end costs one advertise.
func TestServeStartsOneKeeper(t *testing.T) {
	if !within(time.Second, func() bool { return keepers() == 0 }) {
		t.Fatalf("%d keepers left by earlier tests", keepers())
	}
	n := newAdvertiseNode()
	c := n.client(t)
	p := newTestProvider()
	ctx, cancel := astral.NewContext(nil).WithCancel()
	defer cancel()
	defer p.Close()

	hook := func(ctx *astral.Context) error { return p.Serve(ctx, c) }
	if err := hook(ctx); err != nil {
		t.Fatalf("hook: %v", err)
	}
	end := n.next(t, time.Second)
	var wg sync.WaitGroup
	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := hook(ctx); err != nil {
				t.Errorf("hook: %v", err)
			}
		}()
	}
	wg.Wait()

	if !within(time.Second, func() bool { return keepers() == 1 }) {
		t.Fatalf("%d keepers run after three hook runs; want 1", keepers())
	}
	close(end)
	n.next(t, time.Second)
	time.Sleep(200 * time.Millisecond)
	if got := n.calls.Load(); got != 2 {
		t.Fatalf("advertised %d times; want 2: one per hook-opened binding, one per node-side end", got)
	}
	if got := keepers(); got != 1 {
		t.Fatalf("%d keepers after a re-advertise; want 1", got)
	}
}

// When the app context ends, the keeper exits and advertises nothing more.
func TestKeeperStopsWithItsContext(t *testing.T) {
	if !within(time.Second, func() bool { return keepers() == 0 }) {
		t.Fatalf("%d keepers left by earlier tests", keepers())
	}
	n := newAdvertiseNode()
	c := n.client(t)
	p := newTestProvider()
	ctx, cancel := astral.NewContext(nil).WithCancel()

	if err := p.Serve(ctx, c); err != nil {
		t.Fatalf("Serve: %v", err)
	}
	end := n.next(t, time.Second)
	cancel()

	if !within(time.Second, func() bool { return keepers() == 0 }) {
		t.Fatal("the keeper outlived its context")
	}
	close(end)
	time.Sleep(200 * time.Millisecond)
	if got := n.calls.Load(); got != 1 {
		t.Fatalf("advertised %d times after the context ended; want only the first", got)
	}
}

// Close stops the keeper: the binding it ends is not advertised again.
func TestCloseStopsTheKeeper(t *testing.T) {
	n := newAdvertiseNode()
	c := n.client(t)
	p := newTestProvider()
	ctx, cancel := astral.NewContext(nil).WithCancel()
	defer cancel()

	if err := p.Serve(ctx, c); err != nil {
		t.Fatalf("Serve: %v", err)
	}
	n.next(t, time.Second)
	if err := p.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}
	time.Sleep(200 * time.Millisecond)
	if got := n.calls.Load(); got != 1 {
		t.Fatalf("advertised %d times after Close; want only the first", got)
	}
	if p.current() != nil {
		t.Fatal("a binding is open after Close")
	}
}
