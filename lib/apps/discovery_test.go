package apps

import (
	"sync/atomic"
	"testing"
	"time"

	"github.com/astralp2p/astral-go/api/services"
	servicesclient "github.com/astralp2p/astral-go/api/services/client"
	"github.com/astralp2p/astral-go/astral"
	"github.com/astralp2p/astral-go/lib/astrald"
	"github.com/astralp2p/astral-go/lib/query"
	"github.com/astralp2p/astral-go/lib/routing"
)

type opRouter struct {
	inner astral.Router
	id    *astral.Identity
}

func (r *opRouter) RouteQuery(ctx *astral.Context, q *astral.InFlightQuery) (astral.Conn, error) {
	return query.RouteInFlight(ctx, r.inner, q)
}

func (r *opRouter) GuestID() *astral.Identity { return r.id }
func (r *opRouter) HostID() *astral.Identity  { return r.id }

// followingNode answers every services.discover with one available offering
// of provider and the boundary; the first follow then ends, every later one
// stays open. It counts the follows.
func followingNode(t *testing.T, provider *astral.Identity) (*servicesclient.Client, *atomic.Int32) {
	t.Helper()
	var calls atomic.Int32
	router := routing.NewOpRouter()
	op, err := routing.NewOp(func(_ *astral.Context, q *routing.IncomingQuery, _ struct{ Services string }) error {
		n := calls.Add(1)
		ch := q.Accept()
		defer ch.Close()
		if err := ch.Send(&services.Update{Available: true, Name: "player", ProviderID: provider}); err != nil {
			return err
		}
		if err := ch.Send(&astral.EOS{}); err != nil {
			return err
		}
		for n > 1 {
			if _, err := ch.Receive(); err != nil {
				return nil
			}
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := router.AddOp(services.MethodDiscover, op); err != nil {
		t.Fatal(err)
	}
	id := astral.GenerateIdentity()
	return servicesclient.New(id, astrald.New(&opRouter{inner: router, id: id})), &calls
}

// A discovery reopens its follow after the stream ends, and onChange sees the
// set; stop closes the Watcher and onChange sees it stopped.
func TestDiscoveryReopensAndReports(t *testing.T) {
	provider := astral.GenerateIdentity()
	client, calls := followingNode(t, provider)
	seen := make(chan *servicesclient.Watcher, 64)
	d := discovery{names: []string{"player"}, reach: services.ReachLocal, onChange: func(w *servicesclient.Watcher) { seen <- w }}

	stop, err := startDiscoveries(astral.NewContext(nil), []discovery{d}, client)
	if err != nil {
		t.Fatal(err)
	}
	var w *servicesclient.Watcher
	select {
	case w = <-seen:
	case <-time.After(10 * time.Second):
		t.Fatal("onChange never ran")
	}
	deadline := time.Now().Add(10 * time.Second)
	for calls.Load() < 2 || w.Stale() || len(w.Offerings()) != 1 {
		if time.Now().After(deadline) {
			t.Fatalf("after 10s: %d follows, stale %v, %d offerings; want the reopened follow's set", calls.Load(), w.Stale(), len(w.Offerings()))
		}
		time.Sleep(10 * time.Millisecond)
	}

	stop()
	select {
	case <-w.Done():
	default:
		t.Fatal("the Watcher runs after stop")
	}
	if w.Err() == nil {
		t.Fatal("a stopped Watcher has no Err")
	}
}

func TestWithDiscoveryChecksItsArguments(t *testing.T) {
	cfg := &serveConfig{}
	if WithDiscovery(nil, services.ReachLocal, func(*servicesclient.Watcher) {})(cfg) == nil {
		t.Fatal("no names accepted")
	}
	if WithDiscovery([]string{"player"}, services.ReachLocal, nil)(cfg) == nil {
		t.Fatal("a nil callback accepted")
	}
	if err := WithDiscovery([]string{"player"}, services.ReachSwarm, func(*servicesclient.Watcher) {})(cfg); err != nil || len(cfg.discoveries) != 1 {
		t.Fatalf("WithDiscovery: %v, %d discoveries", err, len(cfg.discoveries))
	}
}
