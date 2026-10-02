package services

import (
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/astralp2p/astral-go/api/services"
	"github.com/astralp2p/astral-go/astral"
	"github.com/astralp2p/astral-go/astral/channel"
	"github.com/astralp2p/astral-go/lib/astrald"
	"github.com/astralp2p/astral-go/lib/query"
	"github.com/astralp2p/astral-go/lib/routing"
)

// Tests in this file route the real client through in-process ops standing in
// for the node, so the wire argument names are pinned.

type localRouter struct {
	inner astral.Router
	id    *astral.Identity
}

func (r *localRouter) RouteQuery(ctx *astral.Context, q *astral.InFlightQuery) (astral.Conn, error) {
	return query.RouteInFlight(ctx, r.inner, q)
}

func (r *localRouter) GuestID() *astral.Identity { return r.id }
func (r *localRouter) HostID() *astral.Identity  { return r.id }

func clientFor(t *testing.T, ops map[string]any) *Client {
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
	return New(id, astrald.New(&localRouter{inner: router, id: id}))
}

type discoverArgs struct {
	Services string
	Follow   bool
	Reach    string
	For      string
}

// The reach and for arguments reach the node under the names astrald declares.
func TestDiscoverSendsReachAndFor(t *testing.T) {
	got := make(chan discoverArgs, 2)
	c := clientFor(t, map[string]any{services.MethodDiscover: func(_ *astral.Context, q *routing.IncomingQuery, args discoverArgs) error {
		got <- args
		ch := q.Accept()
		defer ch.Close()
		return ch.Send(&astral.EOS{})
	}})
	ctx := astral.NewContext(nil)

	events, err := c.DiscoverIn(ctx, services.ReachSwarm, []string{"player", "wallet"}, false)
	if err != nil {
		t.Fatal(err)
	}
	for range events {
	}
	if a := <-got; a.Reach != "swarm" || a.Services != "player,wallet" || a.For != "" {
		t.Fatalf("DiscoverIn sent %+v", a)
	}

	app := astral.GenerateIdentity()
	events, err = c.DiscoverFor(ctx, app, []string{"player"}, false)
	if err != nil {
		t.Fatal(err)
	}
	for range events {
	}
	if a := <-got; a.For != app.String() || a.Reach != "" {
		t.Fatalf("DiscoverFor sent %+v", a)
	}
}

// Plan §8: a provider restarting while the node still holds its previous
// binding is refused as already advertised, and Serve retries until the node
// lets the old binding go.
func TestProviderServeWaitsOutTheOldBinding(t *testing.T) {
	var refusals atomic.Int32
	refusals.Store(2)
	opened := make(chan struct{}, 1)
	c := clientFor(t, map[string]any{services.MethodAdvertise: func(_ *astral.Context, q *routing.IncomingQuery, _ struct{ Services string }) error {
		ch := q.Accept()
		defer ch.Close()
		if refusals.Add(-1) >= 0 {
			return ch.Send(astral.Err(services.ErrAdvertised))
		}
		if err := ch.Send(&astral.Ack{}); err != nil {
			return err
		}
		opened <- struct{}{}
		// like the node: the binding lasts until the provider closes its end
		for {
			if _, err := ch.Receive(); err != nil {
				return nil
			}
		}
	}})

	p := NewProvider(map[string]OfferingFunc{"player": func(*astral.Context, *astral.Identity) (*services.Update, error) { return nil, nil }})
	ctx, cancel := astral.NewContext(nil).WithCancel()
	defer cancel()

	start := time.Now()
	if err := p.Serve(ctx, c); err != nil {
		t.Fatalf("Serve: %v", err)
	}
	select {
	case <-opened:
	case <-time.After(5 * time.Second):
		t.Fatal("the binding never opened")
	}
	if d := time.Since(start); d < retryFirst {
		t.Fatalf("Serve retried after %v; want a backoff of at least %v", d, retryFirst)
	}
	if err := p.Serve(ctx, c); err != nil {
		t.Fatalf("a second Serve with an open binding: %v", err)
	}
	if n := refusals.Load(); n != -1 {
		t.Fatalf("advertise called %d extra times after the binding opened", -1-n)
	}
	p.Close()
}

// Plan §8: answers from concurrent handlers and changes from other goroutines
// share one channel without corrupting it.
func TestConcurrentAnswersAndChanges(t *testing.T) {
	const asks, changes = 200, 50
	caller := astral.GenerateIdentity()
	objs := []astral.Object{&astral.Ack{}}
	for i := 1; i <= asks; i++ {
		objs = append(objs, &services.Ask{RequestID: astral.Nonce(i), CallerID: caller, Service: "player"})
	}
	in := encode(t, objs...)
	out := &lockedBuffer{}
	h := func(*astral.Context, *astral.Identity, string) (*services.Update, error) {
		info := astral.NewBundle()
		_ = info.Append(astral.NewString8("some info that makes each answer longer than a write"))
		return &services.Update{Available: true, Info: info}, nil
	}
	b, err := startBinding(astral.NewContext(nil), channel.New(channel.Join(in, out), channel.WithLockedWrites()), h)
	if err != nil {
		t.Fatal(err)
	}

	var wg sync.WaitGroup
	for i := 0; i < changes; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_ = b.Change(caller)
		}()
	}
	wg.Wait()

	deadline := time.Now().Add(5 * time.Second)
	for {
		var answers, changed int
		for _, o := range decode(t, out.bytes()) {
			switch o.(type) {
			case *services.Answer:
				answers++
			case *services.Change:
				changed++
			default:
				t.Fatalf("unexpected %s on the channel", o.ObjectType())
			}
		}
		if answers == asks && changed == changes {
			return
		}
		if time.Now().After(deadline) {
			t.Fatalf("decoded %d answers and %d changes; want %d and %d", answers, changed, asks, changes)
		}
		time.Sleep(10 * time.Millisecond)
	}
}
