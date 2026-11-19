package inference

import (
	"errors"
	"sync/atomic"
	"testing"
	"time"

	"github.com/astralp2p/astral-go/api/inference"
	"github.com/astralp2p/astral-go/api/services"
	servicescli "github.com/astralp2p/astral-go/api/services/client"
	"github.com/astralp2p/astral-go/astral"
	"github.com/astralp2p/astral-go/lib/routing"
)

type discoverArgs struct {
	Services string
	Follow   bool
	Reach    string
	For      string
}

// node answers services.discover with stream, then, when hold is set, keeps
// the stream open until the caller leaves.
func node(t *testing.T, got chan<- discoverArgs, hold bool, stream ...astral.Object) *Client {
	t.Helper()
	c, _ := clientFor(t, map[string]any{services.MethodDiscover: func(_ *astral.Context, q *routing.IncomingQuery, args discoverArgs) error {
		got <- args
		ch := q.Accept()
		defer ch.Close()
		for _, o := range stream {
			if err := ch.Send(o); err != nil {
				return err
			}
		}
		for hold {
			if _, err := ch.Receive(); err != nil {
				return nil
			}
		}
		return nil
	}})
	return c
}

func offering(t *testing.T, provider *astral.Identity, info ...astral.Object) *services.Update {
	t.Helper()
	bundle := astral.NewBundle()
	if err := bundle.Append(info...); err != nil {
		t.Fatal(err)
	}
	return &services.Update{Available: true, Name: inference.ServiceName, ProviderID: provider, Info: bundle}
}

func TestDiscoverReadsTheOffering(t *testing.T) {
	ai0, other := astral.GenerateIdentity(), astral.GenerateIdentity()
	got := make(chan discoverArgs, 1)
	c := node(t, got, false,
		offering(t, ai0,
			&inference.Models{Models: []*inference.Model{{ID: "m", Tools: true}}},
			&inference.Provider{Name: "ai0", Description: "the lab box"},
			&services.OperationsList{Operations: []astral.String8{inference.OpModels, inference.OpComplete}},
		),
		&services.Update{Name: inference.ServiceName, ProviderID: other},
		&services.Removed{Offerings: []*services.OfferingKey{{ProviderID: other, Name: inference.ServiceName}, {ProviderID: ai0, Name: "player"}}},
		&services.Removed{Offerings: []*services.OfferingKey{{ProviderID: ai0, Name: "player"}}},
		&services.Incomplete{Services: []astral.String8{inference.ServiceName}},
		&astral.EOS{},
	)

	items, err := c.Discover(astral.NewContext(nil), services.ReachSwarm, false)
	if err != nil {
		t.Fatal(err)
	}
	var list []Discovery
	for d := range items {
		list = append(list, d)
	}

	if a := <-got; a.Services != inference.ServiceName || a.Reach != string(services.ReachSwarm) || a.Follow {
		t.Fatalf("the node got %+v", a)
	}
	if len(list) != 4 {
		t.Fatalf("%d items; want an offering, a withdrawal, a removal and the initial outcome: %+v", len(list), list)
	}
	o := list[0].Offering
	switch {
	case o == nil, !o.ProviderID.IsEqual(ai0), !o.Available:
		t.Fatalf("offering %+v", o)
	case o.Provider == nil || o.Provider.Name != "ai0":
		t.Fatalf("provider %+v", o.Provider)
	case o.Models == nil || len(o.Models.Models) != 1 || o.Models.Models[0].ID != "m":
		t.Fatalf("models %+v", o.Models)
	case len(o.Operations) != 2 || o.Operations[1] != inference.OpComplete:
		t.Fatalf("operations %v", o.Operations)
	}
	if w := list[1].Offering; w == nil || w.Available || w.Models != nil || w.Provider != nil {
		t.Fatalf("withdrawal %+v", w)
	}
	if r := list[2].Removed; len(r) != 1 || !r[0].IsEqual(other) {
		t.Fatalf("removed %v; want the inference provider alone", r)
	}
	if i := list[3].Initial; i == nil || i.Complete || len(i.Incomplete) != 1 {
		t.Fatalf("initial %+v", i)
	}
}

// A follower that ends ctx sees the channel close at once, though the node
// sends nothing more.
func TestDiscoverFollowEndsWithCtx(t *testing.T) {
	c := node(t, make(chan discoverArgs, 1), true, &astral.EOS{})
	ctx, cancel := astral.NewContext(nil).WithCancel()
	defer cancel()

	items, err := c.Discover(ctx, services.ReachLocal, true)
	if err != nil {
		t.Fatal(err)
	}
	if d := <-items; d.Initial == nil || !d.Initial.Complete {
		t.Fatalf("first item %+v; want a complete initial outcome", d)
	}
	cancel()

	select {
	case d, open := <-items:
		if open {
			t.Fatalf("an item after the end of ctx: %+v", d)
		}
	case <-time.After(time.Second):
		t.Fatal("the channel is still open 1 s after ctx ended")
	}
}

// A follow the node ends is reopened: the first stream's offering, its
// initial outcome, then the reopened follow's difference (the offering it no
// longer shows) and its own initial outcome. The channel stays open.
func TestDiscoverFollowReopens(t *testing.T) {
	ai0 := astral.GenerateIdentity()
	var calls atomic.Int32
	c, _ := clientFor(t, map[string]any{services.MethodDiscover: func(_ *astral.Context, q *routing.IncomingQuery, _ discoverArgs) error {
		ch := q.Accept()
		defer ch.Close()
		if calls.Add(1) == 1 {
			if err := ch.Send(offering(t, ai0)); err != nil {
				return err
			}
			return ch.Send(&astral.EOS{})
		}
		if err := ch.Send(&astral.EOS{}); err != nil {
			return err
		}
		for {
			if _, err := ch.Receive(); err != nil {
				return nil
			}
		}
	}})
	ctx, cancel := astral.NewContext(nil).WithCancel()
	defer cancel()

	items, err := c.Discover(ctx, services.ReachLocal, true)
	if err != nil {
		t.Fatal(err)
	}
	var list []Discovery
	for len(list) < 4 {
		select {
		case d, open := <-items:
			if !open {
				t.Fatalf("the channel closed after %+v", list)
			}
			list = append(list, d)
		case <-time.After(5 * time.Second):
			t.Fatalf("items %+v; want four within 5 s", list)
		}
	}
	switch {
	case list[0].Offering == nil || !list[0].Offering.ProviderID.IsEqual(ai0):
		t.Fatalf("first item %+v; want the offering", list[0])
	case list[1].Initial == nil:
		t.Fatalf("second item %+v; want the first initial outcome", list[1])
	case len(list[2].Removed) != 1 || !list[2].Removed[0].IsEqual(ai0):
		t.Fatalf("third item %+v; want the offering the reopened follow lost", list[2])
	case list[3].Initial == nil || !list[3].Initial.Complete:
		t.Fatalf("fourth item %+v; want the reopened follow's initial outcome", list[3])
	}
	if n := calls.Load(); n != 2 {
		t.Fatalf("%d follows opened; want 2", n)
	}
}

// A refusal reaches the caller first as ErrNotGranted carrying the refused
// names, and the offerings the caller may still see follow it.
func TestDiscoverReportsTheRefusal(t *testing.T) {
	ai0 := astral.GenerateIdentity()
	c := node(t, make(chan discoverArgs, 1), false,
		&services.NotGranted{Services: []astral.String8{inference.ServiceName}},
		offering(t, ai0),
		&astral.EOS{},
	)

	items, err := c.Discover(astral.NewContext(nil), services.ReachSwarm, false)
	if err != nil {
		t.Fatal(err)
	}
	var list []Discovery
	for d := range items {
		list = append(list, d)
	}

	if len(list) != 3 {
		t.Fatalf("%d items; want the refusal, the offering and the initial outcome: %+v", len(list), list)
	}
	var refused *NotGrantedError
	switch err := list[0].Err; {
	case !errors.Is(err, ErrNotGranted):
		t.Fatalf("first item %+v; want ErrNotGranted", list[0])
	case !errors.As(err, &refused) || len(refused.Services) != 1 || refused.Services[0] != inference.ServiceName:
		t.Fatalf("refusal %v; want the inference service named", err)
	}
	if o := list[1].Offering; o == nil || !o.ProviderID.IsEqual(ai0) {
		t.Fatalf("second item %+v; want the visible offering", list[1])
	}
	if list[2].Initial == nil {
		t.Fatalf("last item %+v; want the initial outcome", list[2])
	}
}

// A refusal naming only other services is not an inference item.
func TestReadDiscoveryIgnoresOtherRefusals(t *testing.T) {
	if d, ok := readDiscovery(servicescli.Event{NotGranted: []string{"chat"}}); ok {
		t.Fatalf("item %+v for a refusal of another service", d)
	}
}

// An event with no field set is not an item.
func TestReadDiscoverySkipsAnEmptyEvent(t *testing.T) {
	if d, ok := readDiscovery(servicescli.Event{}); ok {
		t.Fatalf("item %+v for an empty event", d)
	}
}
