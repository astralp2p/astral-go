package services

import (
	"context"
	"sync"
	"time"

	"github.com/astralp2p/astral-go/api/services"
	"github.com/astralp2p/astral-go/astral"
)

// Watcher follows a discovery and keeps the current set of available
// offerings. Without WithReconnect it follows one stream: after Err the app
// starts a new one. With WithReconnect it reopens the follow with backoff
// after the stream ends, keeps the set it held marked Stale until the new
// follow's boundary, and then applies the difference in one step. Close ends
// it.
type Watcher struct {
	mu      sync.Mutex
	set     followSet
	err     error
	settled bool // Initial has delivered its outcome or closed without one

	cancel  context.CancelFunc
	initial chan InitialOutcome
	changed chan struct{}
	synced  chan struct{}
	done    chan struct{}
}

// WatchOption configures WatchIn and FollowReconnecting.
type WatchOption func(*watchConfig)

type watchConfig struct {
	reconnect bool
	backoff   backoff
}

// WithReconnect makes a Watcher reopen its follow after the stream ends. The
// first wait is min and doubles up to max; it returns to min after a follow
// that reached its boundary. A zero min is 1 s and a zero max is 30 s.
//
// The Watcher stops only when Close or the caller's context ends it, or when
// the node refuses the very first follow before its boundary.
func WithReconnect(min, max time.Duration) WatchOption {
	return func(cfg *watchConfig) {
		cfg.reconnect = true
		cfg.backoff.min, cfg.backoff.max = min, max
	}
}

// withWait replaces the backoff wait, for tests.
func withWait(wait func(*astral.Context, time.Duration) error) WatchOption {
	return func(cfg *watchConfig) { cfg.backoff.wait = wait }
}

func newWatchConfig(opts []WatchOption) watchConfig {
	var cfg watchConfig
	for _, opt := range opts {
		opt(&cfg)
	}
	cfg.backoff = cfg.backoff.withDefaults()
	return cfg
}

type watchKey struct{ provider, name string }

// Removal is an offering the stream removed and has not offered again.
type Removal struct {
	Offering *services.Update      // the last view shown before the removal
	Cause    services.RemovalCause // why the node removed it; zero when a reopened follow no longer shows it
}

// Watch follows names on the target node.
func (client *Client) Watch(ctx *astral.Context, names []string, opts ...WatchOption) (*Watcher, error) {
	return client.WatchIn(ctx, services.ReachLocal, names, opts...)
}

// WatchIn follows names with an explicit reach. Without WithReconnect it
// returns the error of opening the follow. With WithReconnect it checks the
// names and returns at once; the Watcher opens the follow, and opens it again
// after a failure.
func (client *Client) WatchIn(ctx *astral.Context, reach services.Reach, names []string, opts ...WatchOption) (*Watcher, error) {
	cfg := newWatchConfig(opts)
	open := func(ctx *astral.Context) (<-chan Event, error) {
		return client.DiscoverIn(ctx, reach, names, true)
	}
	if cfg.reconnect {
		if err := checkFollowNames(names); err != nil {
			return nil, err
		}
	}

	// why: the watcher owns a context of its own, so Close ends its stream
	// without ending the caller's.
	ctx, cancel := ctx.WithCancel()
	w := newWatcher(cancel)
	if cfg.reconnect {
		go w.runReconnecting(ctx, open, cfg.backoff)
		return w, nil
	}
	events, err := open(ctx)
	if err != nil {
		cancel()
		return nil, err
	}
	go w.run(ctx, events)
	return w, nil
}

// Watch follows names on the default node.
func Watch(ctx *astral.Context, names []string, opts ...WatchOption) (*Watcher, error) {
	return Default().Watch(ctx, names, opts...)
}

// WatchIn follows names on the default node with an explicit reach.
func WatchIn(ctx *astral.Context, reach services.Reach, names []string, opts ...WatchOption) (*Watcher, error) {
	return Default().WatchIn(ctx, reach, names, opts...)
}

func newWatcher(cancel context.CancelFunc) *Watcher {
	return &Watcher{
		cancel:  cancel,
		set:     newFollowSet(),
		initial: make(chan InitialOutcome, 1),
		changed: make(chan struct{}, 1),
		synced:  make(chan struct{}, 1),
		done:    make(chan struct{}),
	}
}

func (w *Watcher) run(ctx *astral.Context, events <-chan Event) {
	defer close(w.done)
	// why: a stream can end without Close, and the watcher's context stays
	// tied to the caller's until it is cancelled.
	defer w.cancel()
	_, err := w.follow(events)
	w.end(ctx, err)
}

// follow applies one stream's events until it closes. It reports whether the
// stream reached its boundary, and the stream's terminal error.
func (w *Watcher) follow(events <-chan Event) (reached bool, err error) {
	for ev := range events {
		switch {
		case ev.Err != nil:
			err = ev.Err
		case ev.Initial != nil:
			reached = true
			w.boundary(*ev.Initial)
		default:
			w.apply(ev)
		}
	}
	return reached, err
}

// apply takes an update, a removal or a not_granted notice.
func (w *Watcher) apply(ev Event) {
	w.mu.Lock()
	changed := w.set.add(ev)
	w.mu.Unlock()
	if changed {
		notify(w.changed)
	}
}

// boundary ends a follow's initial attempt: a staged follow's difference is
// applied, the first boundary settles Initial, and every boundary is Synced.
func (w *Watcher) boundary(o InitialOutcome) {
	w.mu.Lock()
	w.set.settle(o)
	if !w.settled {
		w.initial <- o
		close(w.initial)
		w.settled = true
	}
	w.mu.Unlock()

	notify(w.changed)
	notify(w.synced)
}

// reopen marks the set stale before the next follow.
func (w *Watcher) reopen(error) {
	w.mu.Lock()
	turned := !w.set.stale()
	w.set.reopen()
	w.mu.Unlock()
	if turned {
		notify(w.changed)
	}
}

// end settles the watcher once it has stopped: Err is set, and Initial closes
// when the outcome never came.
func (w *Watcher) end(ctx *astral.Context, err error) {
	w.mu.Lock()
	w.err = err
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

	notify(w.changed)
}

// notify sends a coalesced signal on ch.
func notify(ch chan struct{}) {
	select {
	case ch <- struct{}{}:
	default:
	}
}

// NotGranted returns the watched services the caller holds no discovery
// permit for on some contributing node. Their private offerings there are
// filtered; an app can ask for a discovery permit in its registration or
// contract request to see them.
func (w *Watcher) NotGranted() []string {
	w.mu.Lock()
	defer w.mu.Unlock()
	return append([]string(nil), w.set.ungranted...)
}

// Offerings returns the current available offerings, ordered by provider and
// name.
func (w *Watcher) Offerings() []*services.Update {
	w.mu.Lock()
	defer w.mu.Unlock()
	keys := w.set.keys()
	list := make([]*services.Update, 0, len(keys))
	for _, k := range keys {
		list = append(list, w.set.offerings[k])
	}
	return list
}

// Removals returns the offerings the stream removed and has not offered
// again, with the cause of each removal, ordered by provider and name. An app
// can keep a removal with services.CauseMemberLost as reconnecting, since the
// node retries that member. With WithReconnect, an offering a reopened
// follow's complete boundary no longer shows is a removal with cause zero.
//
// A removal stays until the stream offers its key again. A member_lost
// removal is not superseded when the member later leaves the swarm: the node
// sends no member_left for keys it no longer shows, so the removal keeps
// member_lost. An app that shows reconnecting bounds it with its own timeout.
// The list grows with every key removed and never offered again over the
// watcher's life.
func (w *Watcher) Removals() []Removal {
	w.mu.Lock()
	defer w.mu.Unlock()
	keys := make([]watchKey, 0, len(w.set.removals))
	for k := range w.set.removals {
		keys = append(keys, k)
	}
	sortWatchKeys(keys)
	list := make([]Removal, 0, len(keys))
	for _, k := range keys {
		list = append(list, w.set.removals[k])
	}
	return list
}

// Stale reports whether Offerings holds offerings no follow has confirmed
// since a stream ended: true from the end of a stream until a reopened
// follow's complete boundary. An incomplete boundary keeps the offerings it
// did not show, and Stale stays true until each of them is updated or
// removed, or a later boundary is complete.
func (w *Watcher) Stale() bool {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.set.stale()
}

// Initial receives the outcome of the watcher's first boundary once, then
// closes. When the watcher stops before any boundary, Initial closes without
// an outcome and Err says why. A reopened follow's boundary is reported on
// Synced, not here.
func (w *Watcher) Initial() <-chan InitialOutcome { return w.initial }

// Synced signals, coalesced, that a follow reached its boundary and its
// outcome is applied: the first follow's and every reopened follow's.
func (w *Watcher) Synced() <-chan struct{} { return w.synced }

// Changed signals, coalesced, that the set or the watcher's state changed.
func (w *Watcher) Changed() <-chan struct{} { return w.changed }

// Done is closed when the watcher has stopped: when the stream ends, or with
// WithReconnect when the watcher gives up.
func (w *Watcher) Done() <-chan struct{} { return w.done }

// Close ends the stream and waits until the watcher has stopped. Err is then
// context.Canceled, unless the watcher had already stopped. Closing again
// does nothing.
func (w *Watcher) Close() error {
	w.cancel()
	<-w.done
	return nil
}

// Err is the terminal error, or nil while the watcher runs; with
// WithReconnect it stays nil while the watcher reopens its follow. It is set
// before Done or Initial closes on a failure.
func (w *Watcher) Err() error {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.err
}
