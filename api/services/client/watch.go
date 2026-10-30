package services

import (
	"context"
	"sort"
	"sync"

	"github.com/astralp2p/astral-go/api/services"
	"github.com/astralp2p/astral-go/astral"
)

// Watcher follows a discovery and keeps the current set of available
// offerings. It has no reconnect policy: after Err the app starts a new one.
// Close ends it.
type Watcher struct {
	mu        sync.Mutex
	offerings map[watchKey]*services.Update
	err       error
	settled   bool // Initial has delivered its outcome or closed without one

	cancel  context.CancelFunc
	initial chan InitialOutcome
	changed chan struct{}
	done    chan struct{}
}

type watchKey struct{ provider, name string }

// Watch follows names on the target node.
func (client *Client) Watch(ctx *astral.Context, names []string) (*Watcher, error) {
	return client.WatchIn(ctx, services.ReachLocal, names)
}

// WatchIn follows names with an explicit reach.
func (client *Client) WatchIn(ctx *astral.Context, reach services.Reach, names []string) (*Watcher, error) {
	// why: the watcher owns a context of its own, so Close ends its stream
	// without ending the caller's.
	ctx, cancel := ctx.WithCancel()
	events, err := client.DiscoverIn(ctx, reach, names, true)
	if err != nil {
		cancel()
		return nil, err
	}
	w := &Watcher{
		cancel:    cancel,
		offerings: map[watchKey]*services.Update{},
		initial:   make(chan InitialOutcome, 1),
		changed:   make(chan struct{}, 1),
		done:      make(chan struct{}),
	}
	go w.run(ctx, events)
	return w, nil
}

// Watch follows names on the default node.
func Watch(ctx *astral.Context, names []string) (*Watcher, error) {
	return Default().Watch(ctx, names)
}

// WatchIn follows names on the default node with an explicit reach.
func WatchIn(ctx *astral.Context, reach services.Reach, names []string) (*Watcher, error) {
	return Default().WatchIn(ctx, reach, names)
}

func (w *Watcher) run(ctx *astral.Context, events <-chan Event) {
	defer close(w.done)
	// why: a stream can end without Close, and the watcher's context stays
	// tied to the caller's until it is cancelled.
	defer w.cancel()
	for ev := range events {
		w.apply(ev)
	}
	w.end(ctx)
}

func (w *Watcher) apply(ev Event) {
	w.mu.Lock()
	switch {
	case ev.Update != nil:
		k := watchKey{ev.Update.ProviderID.String(), string(ev.Update.Name)}
		if bool(ev.Update.Available) {
			w.offerings[k] = ev.Update
		} else {
			delete(w.offerings, k)
		}
	case ev.Removed != nil:
		for _, rk := range ev.Removed {
			delete(w.offerings, watchKey{rk.ProviderID.String(), string(rk.Name)})
		}
	case ev.Initial != nil:
		w.initial <- *ev.Initial
		close(w.initial)
		w.settled = true
	case ev.Err != nil:
		w.err = ev.Err
	}
	w.mu.Unlock()

	w.signal()
}

// end settles the watcher once the stream is gone: Err is set, and Initial
// closes when the outcome never came.
func (w *Watcher) end(ctx *astral.Context) {
	w.mu.Lock()
	// why: the reader drops its terminal event when ctx has ended, so the
	// context is the reason when no event named one.
	if w.err == nil {
		w.err = ctx.Err()
	}
	if w.err == nil {
		w.err = ErrStreamEnded
	}
	// why: a caller waiting on Initial would otherwise wait forever on a
	// stream that failed before its outcome.
	if !w.settled {
		close(w.initial)
		w.settled = true
	}
	w.mu.Unlock()

	w.signal()
}

func (w *Watcher) signal() {
	select {
	case w.changed <- struct{}{}:
	default:
	}
}

// Offerings returns the current available offerings, ordered by provider and
// name.
func (w *Watcher) Offerings() []*services.Update {
	w.mu.Lock()
	defer w.mu.Unlock()
	keys := make([]watchKey, 0, len(w.offerings))
	for k := range w.offerings {
		keys = append(keys, k)
	}
	sort.Slice(keys, func(i, j int) bool {
		if keys[i].provider != keys[j].provider {
			return keys[i].provider < keys[j].provider
		}
		return keys[i].name < keys[j].name
	})
	list := make([]*services.Update, 0, len(keys))
	for _, k := range keys {
		list = append(list, w.offerings[k])
	}
	return list
}

// Initial receives the outcome of the initial attempt once, then closes. When
// the stream ends before the outcome, Initial closes without one and Err says
// why.
func (w *Watcher) Initial() <-chan InitialOutcome { return w.initial }

// Changed signals, coalesced, that the set or the watcher's state changed.
func (w *Watcher) Changed() <-chan struct{} { return w.changed }

// Done is closed when the stream ends.
func (w *Watcher) Done() <-chan struct{} { return w.done }

// Close ends the stream and waits until the watcher has stopped. Err is then
// context.Canceled, unless the stream had already ended. Closing again does
// nothing.
func (w *Watcher) Close() error {
	w.cancel()
	<-w.done
	return nil
}

// Err is the terminal error, or nil while the stream is open. It is set before
// Done or Initial closes on a failure.
func (w *Watcher) Err() error {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.err
}
