package apps

import (
	"context"
	"errors"
	"net"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/astralp2p/astral-go/astral"
	"github.com/astralp2p/astral-go/astral/channel"
)

// fakeNode is a node the registrar binds to over a pipe. While down, every bind
// fails; dropping it closes the node end of the current bind, as a restart does.
type fakeNode struct {
	up       atomic.Bool
	failures atomic.Int32

	mu   sync.Mutex
	conn net.Conn
}

func (n *fakeNode) bind(ctx *astral.Context) (*channel.Channel, error) {
	if !n.up.Load() {
		n.failures.Add(1)
		return nil, errors.New("node is down")
	}

	app, node := net.Pipe()
	n.mu.Lock()
	n.conn = node
	n.mu.Unlock()
	return channel.New(app), nil
}

func (n *fakeNode) restart() {
	n.up.Store(false)
	n.mu.Lock()
	n.conn.Close()
	n.mu.Unlock()
}

func withTestBackoff(min, max time.Duration) AppRegistrarOption {
	return func(s *AppRegistrar) { s.minBackoff, s.maxBackoff = min, max }
}

func withTestAttach(fn func(*astral.Context, *channel.Channel, handler) error) AppRegistrarOption {
	return func(s *AppRegistrar) { s.attachHandler = fn }
}

func waitFor(t *testing.T, what string, cond func() bool) {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for !cond() {
		if time.Now().After(deadline) {
			t.Fatalf("timed out waiting for %s", what)
		}
		time.Sleep(time.Millisecond)
	}
}

func isClosed(ch <-chan struct{}) bool {
	select {
	case <-ch:
		return true
	default:
		return false
	}
}

// TestAppRegistrarRebindsAfterNodeRestart: a node that is still down at the
// first reconnect attempt must not leave the app unregistered for good. The
// registrar keeps binding until the node is back, then re-attaches the handler
// and re-runs the hooks that advertise the app's services.
func TestAppRegistrarRebindsAfterNodeRestart(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	actx := astral.NewContext(ctx)

	node := &fakeNode{}
	node.up.Store(true)

	var attached, advertised atomic.Int32
	reg := NewAppRegistrar(actx,
		WithBind(node.bind),
		withTestBackoff(time.Millisecond, 5*time.Millisecond),
		withTestAttach(func(*astral.Context, *channel.Channel, handler) error {
			attached.Add(1)
			return nil
		}),
		WithRegistrarRegistrationHooks(func(*astral.Context) error {
			advertised.Add(1)
			return nil
		}),
	)

	if err := reg.Register(actx, "test:endpoint", astral.NewNonce()); err != nil {
		t.Fatalf("register: %v", err)
	}
	if attached.Load() != 1 || advertised.Load() != 1 {
		t.Fatalf("after register: attached %v, advertised %v; want 1, 1", attached.Load(), advertised.Load())
	}

	node.restart()

	waitFor(t, "the registrar to notice the drop", func() bool { return !isClosed(reg.Ready()) })
	waitFor(t, "three failed binds while the node is down", func() bool { return node.failures.Load() >= 3 })
	node.up.Store(true)

	waitFor(t, "the handler to be re-attached", func() bool { return attached.Load() == 2 })
	waitFor(t, "the services to be re-advertised", func() bool { return advertised.Load() == 2 })
	waitFor(t, "the registrar to be ready again", func() bool { return isClosed(reg.Ready()) })
}

// TestAppRegistrarStopsRetryingWhenCancelled: retrying is bounded by the
// registrar's context, so a cancelled app does not keep dialing a dead node.
func TestAppRegistrarStopsRetryingWhenCancelled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())

	node := &fakeNode{}
	NewAppRegistrar(astral.NewContext(ctx),
		WithBind(node.bind),
		withTestBackoff(time.Millisecond, time.Millisecond),
	)

	waitFor(t, "failed binds while the node is down", func() bool { return node.failures.Load() >= 2 })
	cancel()

	time.Sleep(20 * time.Millisecond)
	settled := node.failures.Load()
	time.Sleep(50 * time.Millisecond)
	if got := node.failures.Load(); got != settled {
		t.Fatalf("binds after cancel: %v more", got-settled)
	}
}
