package services

import (
	"sort"
	"sync"

	"github.com/astralp2p/astral-go/api/services"
	"github.com/astralp2p/astral-go/astral"
)

// Watcher follows a discovery and keeps the current set of available
// offerings. It has no reconnect policy: after Err the app starts a new one.
type Watcher struct {
	mu        sync.Mutex
	offerings map[watchKey]*services.Update
	err       error

	initial chan InitialOutcome
	changed chan struct{}
	done    chan struct{}
}

type watchKey struct{ provider, name string }

// Watch follows names on the target node.
func (client *Client) Watch(ctx *astral.Context, names []string) (*Watcher, error) {
	events, err := client.Discover(ctx, names, true)
	if err != nil {
		return nil, err
	}
	w := &Watcher{
		offerings: map[watchKey]*services.Update{},
		initial:   make(chan InitialOutcome, 1),
		changed:   make(chan struct{}, 1),
		done:      make(chan struct{}),
	}
	go w.run(events)
	return w, nil
}

// Watch follows names on the default node.
func Watch(ctx *astral.Context, names []string) (*Watcher, error) {
	return Default().Watch(ctx, names)
}

func (w *Watcher) run(events <-chan Event) {
	defer close(w.done)
	for ev := range events {
		w.apply(ev)
	}
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
	case ev.Err != nil:
		w.err = ev.Err
	}
	w.mu.Unlock()

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

// Initial receives the outcome of the initial attempt once, then closes.
func (w *Watcher) Initial() <-chan InitialOutcome { return w.initial }

// Changed signals, coalesced, that the set or the watcher's state changed.
func (w *Watcher) Changed() <-chan struct{} { return w.changed }

// Done is closed when the stream ends.
func (w *Watcher) Done() <-chan struct{} { return w.done }

// Err is the terminal error, or nil while the stream is open.
func (w *Watcher) Err() error {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.err
}
