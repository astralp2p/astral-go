package services

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/astralp2p/astral-go/api/services"
	"github.com/astralp2p/astral-go/astral"
	"github.com/astralp2p/astral-go/lib/routing"
)

// follow is one services.discover the scripted node answers: it sends send,
// waiting on gate (when set) before the last object, and then ends the
// stream, or with hold keeps it open until the caller leaves.
type follow struct {
	send []astral.Object
	gate chan struct{}
	hold bool
}

// scriptedNode answers the n-th services.discover with follows[n]; once the
// script runs out it holds every further follow open. It counts the calls.
func scriptedNode(t *testing.T, follows ...follow) (*Client, *atomic.Int32) {
	t.Helper()
	var calls atomic.Int32
	c := clientFor(t, map[string]any{services.MethodDiscover: func(_ *astral.Context, q *routing.IncomingQuery, _ discoverArgs) error {
		n := int(calls.Add(1)) - 1
		f := follow{hold: true}
		if n < len(follows) {
			f = follows[n]
		}
		ch := q.Accept()
		defer ch.Close()
		for i, o := range f.send {
			if f.gate != nil && i == len(f.send)-1 {
				<-f.gate
			}
			if err := ch.Send(o); err != nil {
				return err
			}
		}
		for f.hold {
			if _, err := ch.Receive(); err != nil {
				return nil
			}
		}
		return nil
	}})
	return c, &calls
}

// delays records the backoff waits without waiting.
type delays struct {
	mu   sync.Mutex
	list []time.Duration
}

func (d *delays) wait(_ *astral.Context, delay time.Duration) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.list = append(d.list, delay)
	return nil
}

func (d *delays) get() []time.Duration {
	d.mu.Lock()
	defer d.mu.Unlock()
	return append([]time.Duration(nil), d.list...)
}

// startReconnecting starts a reconnecting watcher on c whose Changed channel
// counts up to 64 signals, and whose backoff waits are recorded in d.
func startReconnecting(t *testing.T, c *Client, d *delays) *Watcher {
	t.Helper()
	ctx, cancel := astral.NewContext(nil).WithCancel()
	w := newWatcher(cancel)
	w.changed = make(chan struct{}, 64)
	cfg := newWatchConfig([]WatchOption{WithReconnect(time.Second, 30*time.Second), withWait(d.wait)})
	open := func(ctx *astral.Context) (<-chan Event, error) {
		return c.DiscoverIn(ctx, services.ReachSwarm, []string{"player"}, true)
	}
	go w.runReconnecting(ctx, open, cfg.backoff)
	t.Cleanup(func() { w.Close() })
	return w
}

func drain(ch <-chan struct{}) int {
	n := 0
	for {
		select {
		case <-ch:
			n++
		default:
			return n
		}
	}
}

func eventually(t *testing.T, what string, cond func() bool) {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for !cond() {
		if time.Now().After(deadline) {
			t.Fatalf("within 5s: %s", what)
		}
		time.Sleep(5 * time.Millisecond)
	}
}

func waitSignal(t *testing.T, ch <-chan struct{}, what string) {
	t.Helper()
	select {
	case <-ch:
	case <-time.After(5 * time.Second):
		t.Fatalf("no %s within 5s", what)
	}
}

func offer(provider *astral.Identity) *services.Update {
	return &services.Update{Available: true, Name: "player", ProviderID: provider}
}

func providers(list []*services.Update) []string {
	var ids []string
	for _, u := range list {
		ids = append(ids, u.ProviderID.String())
	}
	return ids
}

func sameProviders(got []*services.Update, want ...*astral.Identity) bool {
	if len(got) != len(want) {
		return false
	}
	have := map[string]bool{}
	for _, u := range got {
		have[u.ProviderID.String()] = true
	}
	for _, id := range want {
		if !have[id.String()] {
			return false
		}
	}
	return true
}

// A follow that ends is reopened; the old set stays, marked stale, while the
// new follow is staged, and its complete boundary applies the difference in
// one change: B, C, EOS, close; then B, EOS leaves [B].
func TestReconnectAppliesTheDifferenceAtTheBoundary(t *testing.T) {
	b, c := astral.GenerateIdentity(), astral.GenerateIdentity()
	gate := make(chan struct{})
	open := true
	// why: a failed check must not leave the node waiting on the gate
	defer func() {
		if open {
			close(gate)
		}
	}()
	node, calls := scriptedNode(t,
		follow{send: []astral.Object{offer(b), offer(c), &astral.EOS{}}},
		follow{send: []astral.Object{offer(b), &astral.EOS{}}, gate: gate, hold: true},
	)
	w := startReconnecting(t, node, &delays{})

	if o, ok := <-w.Initial(); !ok || !o.Complete {
		t.Fatalf("initial outcome %+v, %v", o, ok)
	}
	waitSignal(t, w.Synced(), "first Synced")
	eventually(t, "the first follow ended and the set turned stale", w.Stale)
	old := w.Offerings()
	if !sameProviders(old, b, c) {
		t.Fatalf("stale offerings %v; want the old B and C", providers(old))
	}
	eventually(t, "the second follow opened", func() bool { return calls.Load() == 2 })
	drain(w.changed)

	// the node sent the new B and holds the EOS: nothing shows yet
	time.Sleep(100 * time.Millisecond)
	if n := drain(w.changed); n != 0 {
		t.Fatalf("Changed fired %d times while the reopened follow was staged", n)
	}
	if got := w.Offerings(); len(got) != 2 || got[0] != old[0] || got[1] != old[1] {
		t.Fatalf("offerings changed before the boundary: %v", providers(got))
	}

	close(gate)
	open = false
	waitSignal(t, w.Synced(), "Synced at the reopened follow's boundary")
	got := w.Offerings()
	if !sameProviders(got, b) || got[0] == old[0] || got[0] == old[1] {
		t.Fatalf("offerings %v after the boundary; want the reopened follow's B alone", providers(got))
	}
	if w.Stale() {
		t.Fatal("Stale after a complete boundary")
	}
	if n := drain(w.changed); n != 1 {
		t.Fatalf("Changed fired %d times for the difference; want 1", n)
	}
	if w.Err() != nil {
		t.Fatalf("Err %v while reconnecting", w.Err())
	}
}

// The node refuses the very first follow before its boundary: the watcher
// stops with that error and never opens a second follow.
func TestReconnectStopsOnAFirstRefusal(t *testing.T) {
	refused := astral.NewError("follow requires services")
	node, calls := scriptedNode(t, follow{send: []astral.Object{astral.Err(refused)}})
	d := &delays{}
	w := startReconnecting(t, node, d)

	waitSignal(t, w.Done(), "Done after a first refusal")
	var em *astral.ErrorMessage
	if err := w.Err(); !errors.As(err, &em) || err.Error() != refused.Error() {
		t.Fatalf("Err %v; want the node's error message", err)
	}
	initialClosed(t, w, time.Second)
	time.Sleep(50 * time.Millisecond)
	if n := calls.Load(); n != 1 {
		t.Fatalf("%d follows opened; want 1", n)
	}
	if len(d.get()) != 0 {
		t.Fatalf("waited %v after a first refusal", d.get())
	}
}

// A refusal of a reopened follow is retried with backoff, and Err stays nil.
func TestReconnectRetriesALaterRefusal(t *testing.T) {
	node, calls := scriptedNode(t,
		follow{send: []astral.Object{&astral.EOS{}}},
		follow{send: []astral.Object{astral.NewError("services not ready")}},
	)
	d := &delays{}
	w := startReconnecting(t, node, d)

	eventually(t, "a third follow opened", func() bool { return calls.Load() == 3 })
	if err := w.Err(); err != nil {
		t.Fatalf("Err %v after a later refusal; want nil", err)
	}
	select {
	case <-w.Done():
		t.Fatal("the watcher stopped on a later refusal")
	default:
	}
	if got := d.get(); len(got) != 2 || got[0] != time.Second || got[1] != 2*time.Second {
		t.Fatalf("delays %v; want 1s after the reached follow, then 2s after the refusal", got)
	}
}

// The backoff returns to its minimum only after a follow that reached its
// boundary.
func TestReconnectBackoff(t *testing.T) {
	cases := []struct {
		name   string
		follow follow
		want   []time.Duration
	}{
		{"drops right after eos", follow{send: []astral.Object{&astral.EOS{}}}, []time.Duration{time.Second, time.Second, time.Second}},
		{"drops before eos", follow{}, []time.Duration{time.Second, 2 * time.Second, 4 * time.Second}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			node, calls := scriptedNode(t, tc.follow, tc.follow, tc.follow)
			d := &delays{}
			w := startReconnecting(t, node, d)
			eventually(t, "a fourth follow opened", func() bool { return calls.Load() == 4 })
			got := d.get()
			if len(got) != len(tc.want) {
				t.Fatalf("delays %v; want %v", got, tc.want)
			}
			for i := range got {
				if got[i] != tc.want[i] {
					t.Fatalf("delays %v; want %v", got, tc.want)
				}
			}
			if w.Err() != nil {
				t.Fatalf("Err %v", w.Err())
			}
		})
	}
}

// The backoff doubles up to its maximum.
func TestReconnectBackoffCap(t *testing.T) {
	var waits []time.Duration
	b := backoff{min: time.Second, max: 3 * time.Second, wait: func(_ *astral.Context, d time.Duration) error {
		waits = append(waits, d)
		if len(waits) == 4 {
			return errors.New("stop")
		}
		return nil
	}}
	err := b.run(astral.NewContext(nil), func() (bool, error) { return false, ErrStreamEnded }, func(error) {})
	if err == nil || err.Error() != "stop" {
		t.Fatalf("run returned %v", err)
	}
	want := []time.Duration{time.Second, 2 * time.Second, 3 * time.Second, 3 * time.Second}
	for i := range want {
		if waits[i] != want[i] {
			t.Fatalf("waits %v; want %v", waits, want)
		}
	}
}

// An incomplete boundary keeps the offerings the reopened follow did not
// show, and the set stays stale until the follow shows each of them again.
func TestReconnectIncompleteBoundaryKeepsUnseenKeys(t *testing.T) {
	b, c := astral.GenerateIdentity(), astral.GenerateIdentity()
	gate := make(chan struct{})
	open := true
	defer func() {
		if open {
			close(gate)
		}
	}()
	incomplete := &services.Incomplete{Services: []astral.String8{"player"}}
	node, _ := scriptedNode(t,
		follow{send: []astral.Object{offer(b), offer(c), &astral.EOS{}}},
		follow{send: []astral.Object{offer(b), incomplete, &astral.EOS{}, offer(c)}, gate: gate, hold: true},
	)
	w := startReconnecting(t, node, &delays{})

	waitSignal(t, w.Synced(), "first Synced")
	waitSignal(t, w.Synced(), "second Synced")
	if !sameProviders(w.Offerings(), b, c) {
		t.Fatalf("offerings %v; want B and the unseen C", providers(w.Offerings()))
	}
	if !w.Stale() {
		t.Fatal("not Stale after an incomplete boundary")
	}

	close(gate)
	open = false
	eventually(t, "the late update of C confirmed the set", func() bool { return !w.Stale() })
	if !sameProviders(w.Offerings(), b, c) {
		t.Fatalf("offerings %v; want B and C", providers(w.Offerings()))
	}
}

// Initial reports the watcher's first boundary once; a reopened follow's
// boundary is only Synced. A first follow that ends before its boundary is
// retried, and Initial waits for the boundary that comes.
func TestReconnectInitialFiresOnce(t *testing.T) {
	incomplete := &services.Incomplete{Services: []astral.String8{"player"}}
	cases := []struct {
		name    string
		follows []follow
		want    InitialOutcome
	}{
		{"first follow reached", []follow{
			{send: []astral.Object{&astral.EOS{}}},
			{send: []astral.Object{incomplete, &astral.EOS{}}, hold: true},
		}, InitialOutcome{Complete: true}},
		{"first follow cut", []follow{
			{},
			{send: []astral.Object{incomplete, &astral.EOS{}}},
			{send: []astral.Object{&astral.EOS{}}, hold: true},
		}, InitialOutcome{Incomplete: []string{"player"}}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			node, calls := scriptedNode(t, tc.follows...)
			w := startReconnecting(t, node, &delays{})
			eventually(t, "the last follow opened", func() bool { return int(calls.Load()) == len(tc.follows) })
			waitSignal(t, w.Synced(), "Synced")

			o, ok := <-w.Initial()
			if !ok || o.Complete != tc.want.Complete || len(o.Incomplete) != len(tc.want.Incomplete) {
				t.Fatalf("initial outcome %+v, %v; want %+v", o, ok, tc.want)
			}
			if o, ok := <-w.Initial(); ok {
				t.Fatalf("a second initial outcome %+v", o)
			}
		})
	}
}

// Close ends a reconnecting watcher, also while it waits to reopen, and no
// further follow opens.
func TestReconnectEndsOnClose(t *testing.T) {
	node, calls := scriptedNode(t, follow{send: []astral.Object{&astral.EOS{}}})
	ctx, cancel := astral.NewContext(nil).WithCancel()
	w := newWatcher(cancel)
	open := func(ctx *astral.Context) (<-chan Event, error) {
		return node.DiscoverIn(ctx, services.ReachSwarm, []string{"player"}, true)
	}
	go w.runReconnecting(ctx, open, backoff{min: time.Hour, max: time.Hour, wait: sleep})

	eventually(t, "the first follow ended", w.Stale)
	closed := make(chan struct{})
	go func() { w.Close(); close(closed) }()
	waitSignal(t, closed, "Close returning")
	if !errors.Is(w.Err(), context.Canceled) {
		t.Fatalf("Err %v; want context.Canceled", w.Err())
	}
	if n := calls.Load(); n != 1 {
		t.Fatalf("%d follows opened; want 1", n)
	}
}

// WatchIn with WithReconnect checks the names and returns without a node.
func TestWatchInWithReconnectChecksNames(t *testing.T) {
	node, _ := scriptedNode(t, follow{send: []astral.Object{&astral.EOS{}}, hold: true})
	if _, err := node.WatchIn(astral.NewContext(nil), services.ReachLocal, nil, WithReconnect(0, 0)); !errors.Is(err, ErrFollowWithoutNames) {
		t.Fatalf("WatchIn without names: %v", err)
	}
	w, err := node.WatchIn(astral.NewContext(nil), services.ReachLocal, []string{"player"}, WithReconnect(0, 0))
	if err != nil {
		t.Fatal(err)
	}
	waitSignal(t, w.Synced(), "Synced")
	w.Close()
}

// FollowReconnecting reads as one follow: the reopened follow's difference
// arrives at its boundary, before its Initial.
func TestFollowReconnectingSendsTheDifference(t *testing.T) {
	b, c := astral.GenerateIdentity(), astral.GenerateIdentity()
	node, _ := scriptedNode(t,
		follow{send: []astral.Object{offer(b), offer(c), &astral.EOS{}}},
		follow{send: []astral.Object{offer(b), &astral.EOS{}}, hold: true},
	)
	ctx, cancel := astral.NewContext(nil).WithCancel()
	defer cancel()
	events, err := node.FollowReconnecting(ctx, services.ReachSwarm, []string{"player"}, withWait(func(*astral.Context, time.Duration) error { return nil }))
	if err != nil {
		t.Fatal(err)
	}
	var kinds []string
	var removed []*services.OfferingKey
	for len(kinds) < 6 {
		select {
		case ev := <-events:
			switch {
			case ev.Update != nil:
				kinds = append(kinds, "update")
			case ev.Removed != nil:
				kinds = append(kinds, "removed")
				removed = ev.Removed
			case ev.Initial != nil:
				kinds = append(kinds, "initial")
			default:
				t.Fatalf("unexpected event %+v", ev)
			}
		case <-time.After(5 * time.Second):
			t.Fatalf("events %v; want six", kinds)
		}
	}
	want := []string{"update", "update", "initial", "update", "removed", "initial"}
	for i := range want {
		if kinds[i] != want[i] {
			t.Fatalf("events %v; want %v", kinds, want)
		}
	}
	if len(removed) != 1 || !removed[0].ProviderID.IsEqual(c) {
		t.Fatalf("removed %v; want C", removed)
	}

	cancel()
	ok, _ := drained(events, time.Second)
	if !ok {
		t.Fatal("the stream did not close after its context ended")
	}
}

// FollowReconnecting ends with the node's refusal of the very first follow.
func TestFollowReconnectingStopsOnAFirstRefusal(t *testing.T) {
	node, calls := scriptedNode(t, follow{send: []astral.Object{astral.NewError("follow requires services")}})
	events, err := node.FollowReconnecting(astral.NewContext(nil), services.ReachLocal, []string{"player"})
	if err != nil {
		t.Fatal(err)
	}
	ok, errs := drained(events, 5*time.Second)
	if !ok || len(errs) != 1 || errs[0].Error() != "follow requires services" {
		t.Fatalf("closed %v with %v; want the refusal alone", ok, errs)
	}
	if n := calls.Load(); n != 1 {
		t.Fatalf("%d follows opened; want 1", n)
	}
}
