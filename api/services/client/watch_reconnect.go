package services

import (
	"errors"
	"time"

	"github.com/astralp2p/astral-go/api/services"
	"github.com/astralp2p/astral-go/astral"
	"github.com/astralp2p/astral-go/sig"
)

// Reconnect backoff bounds, the same as the app registrar's: the first follow
// opens at once, and a reopen waits between them.
const (
	defaultMinBackoff = 1 * time.Second
	defaultMaxBackoff = 30 * time.Second
)

// backoff runs follows one after another until one ends for good.
type backoff struct {
	min, max time.Duration
	wait     func(*astral.Context, time.Duration) error
}

func (b backoff) withDefaults() backoff {
	if b.min <= 0 {
		b.min = defaultMinBackoff
	}
	if b.max <= 0 {
		b.max = defaultMaxBackoff
	}
	b.max = max(b.max, b.min)
	if b.wait == nil {
		b.wait = sleep
	}
	return b
}

// run calls attempt until ctx ends or the first attempt is refused before
// its boundary, and returns the error that stopped it. ended runs after every
// other attempt, before the wait.
//
// why the reset: a node that admits a follow and drops it at once would
// otherwise be reopened in a tight loop; only a follow that reached its
// boundary proves the node is back.
//
// why only the first refusal stops: a refusal before the first boundary is an
// admission error that does not fix itself, while a restarting node can
// briefly refuse a reopened follow until its modules are loaded.
func (b backoff) run(ctx *astral.Context, attempt func() (bool, error), ended func(error)) error {
	delay := b.min
	for first := true; ; first = false {
		reached, err := attempt()
		if ctx.Err() != nil {
			return ctx.Err()
		}
		if first && !reached && isRefusal(err) {
			return err
		}
		ended(err)
		if reached {
			delay = b.min
		}
		if err := b.wait(ctx, delay); err != nil {
			return err
		}
		delay = min(delay*2, b.max)
	}
}

// isRefusal reports whether err is an error the node sent on the stream.
func isRefusal(err error) bool {
	var em *astral.ErrorMessage
	return errors.As(err, &em)
}

func sleep(ctx *astral.Context, d time.Duration) error {
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-t.C:
		return nil
	}
}

// checkFollowNames rejects the names a follow cannot send.
func checkFollowNames(names []string) error {
	if len(names) == 0 {
		return ErrFollowWithoutNames
	}
	_, err := services.ParseNames(services.JoinNames(names))
	return err
}

func (w *Watcher) runReconnecting(ctx *astral.Context, open func(*astral.Context) (<-chan Event, error), b backoff) {
	defer close(w.done)
	defer w.cancel()
	err := b.run(ctx, func() (bool, error) {
		events, err := open(ctx)
		if err != nil {
			return false, err
		}
		return w.follow(events)
	}, w.reopen)
	w.end(ctx, err)
}

// FollowReconnecting follows names with reach and reopens the follow with
// backoff after its stream ends, like a Watcher with WithReconnect (opts set
// the backoff; the default is 1 s doubling to 30 s). The events read as one
// follow: the first follow's pass through; a reopened follow's are held until
// its boundary and then sent as the difference to what was sent before — a
// NotGranted notice, an Update per offering it shows, one Removed per cause
// for the offerings it no longer shows (only after a complete outcome; an
// offering the reopened follow removed keeps the node's cause, one it left
// out has cause zero) — followed by
// its Initial. Initial therefore arrives at every follow's boundary. The
// channel closes after an Err: the end of ctx, or the node's refusal of the
// very first follow before its boundary.
func (client *Client) FollowReconnecting(ctx *astral.Context, reach services.Reach, names []string, opts ...WatchOption) (<-chan Event, error) {
	if err := checkFollowNames(names); err != nil {
		return nil, err
	}
	b := newWatchConfig(opts).backoff
	out := make(chan Event)
	go func() {
		defer close(out)
		r := &relay{set: newFollowSet(), out: out}
		err := b.run(ctx, func() (bool, error) {
			events, err := client.DiscoverIn(ctx, reach, names, true)
			if err != nil {
				return false, err
			}
			return r.follow(ctx, events)
		}, func(error) { r.set.reopen() })
		_ = sig.Send(ctx, out, Event{Err: err})
	}()
	return out, nil
}

// FollowReconnecting runs FollowReconnecting on the default client.
func FollowReconnecting(ctx *astral.Context, reach services.Reach, names []string, opts ...WatchOption) (<-chan Event, error) {
	return Default().FollowReconnecting(ctx, reach, names, opts...)
}

// relay passes one reconnecting follow's events on, as differences.
type relay struct {
	set followSet
	out chan<- Event
}

// follow relays one stream's events until it closes. It reports whether the
// stream reached its boundary, and the stream's terminal error.
func (r *relay) follow(ctx *astral.Context, events <-chan Event) (reached bool, err error) {
	for ev := range events {
		var batch []Event
		switch {
		case ev.Err != nil:
			err = ev.Err
		case ev.Initial != nil:
			reached = true
			batch = append(r.set.settle(*ev.Initial), ev)
		case r.set.add(ev):
			batch = []Event{ev}
		}
		for _, e := range batch {
			if err := sig.Send(ctx, r.out, e); err != nil {
				return reached, err
			}
		}
	}
	return reached, err
}
