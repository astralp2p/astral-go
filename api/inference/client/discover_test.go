package inference

import (
	"testing"
	"time"

	"github.com/astralp2p/astral-go/api/inference"
	"github.com/astralp2p/astral-go/api/services"
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

// A stream the node ends while following is reported, then the channel
// closes.
func TestDiscoverFollowReportsTheEnd(t *testing.T) {
	c := node(t, make(chan discoverArgs, 1), false, &astral.EOS{})

	items, err := c.Discover(astral.NewContext(nil), services.ReachLocal, true)
	if err != nil {
		t.Fatal(err)
	}
	var last Discovery
	for d := range items {
		last = d
	}
	if last.Err == nil {
		t.Fatalf("last item %+v; want the end of the stream", last)
	}
}
