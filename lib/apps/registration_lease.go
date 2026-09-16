package apps

import (
	"sync"
	"time"

	"github.com/astralp2p/astral-go/api/objects"
	"github.com/astralp2p/astral-go/astral"
	"github.com/astralp2p/astral-go/sig"
)

// DefaultRegistrationLease is the lease an app requests when it registers as an
// external searcher, describer or finder. The node clamps the request to its own
// maximum, so this is what the app asks for, never what it is guaranteed.
const DefaultRegistrationLease = astral.Duration(time.Hour)

// renewDivisor divides a granted lease to get the renewal interval. Renewing at
// a third of the lease leaves room for two consecutive failures — a reconnect, a
// node briefly busy — before the registration actually lapses.
const renewDivisor = 3

// minRenewInterval floors the renewal interval. A node granting a very short
// lease would otherwise have an app renewing in a tight loop, which costs more
// than the stale registration the lease exists to prevent.
const minRenewInterval = astral.Duration(time.Second)

// registerFunc is the shape of the three objects client registration calls.
type registerFunc func(*astral.Context, astral.Duration) (*objects.RegistrationLease, error)

// leaseKeeper registers an app as an external provider and keeps that
// registration alive, so an app author writes no renewal code.
//
// Hook is a RegistrationHook: the registrar runs it on first connect and on
// every reconnect. Registering is synchronous there, so a failure aborts the
// reconnect cycle as the hook contract requires. The renewal loop starts on the
// first run and outlives individual reconnects, reading the lease the node last
// granted each time round.
type leaseKeeper struct {
	register  registerFunc
	requested astral.Duration
	granted   sig.Value[astral.Duration]
	once      sync.Once
}

func newLeaseKeeper(register registerFunc, requested astral.Duration) *leaseKeeper {
	return &leaseKeeper{register: register, requested: requested}
}

// Hook registers the app now and, the first time it runs, starts the renewal
// loop.
//
// The registrar hands every hook run the same context — the one given to
// NewAppRegistrar — so the loop started here lives as long as the registrar
// itself rather than as long as one connection.
func (k *leaseKeeper) Hook(ctx *astral.Context) error {
	if err := k.renew(ctx); err != nil {
		return err
	}

	k.once.Do(func() { go k.run(ctx) })

	return nil
}

// renew registers, which is also how a registration is renewed: the node
// refreshes the caller's existing entry rather than adding a second one.
func (k *leaseKeeper) renew(ctx *astral.Context) error {
	lease, err := k.register(ctx, k.requested)
	if err != nil {
		return err
	}

	// why the nil check: a node that answers without a lease body leaves the
	// last known grant in place rather than resetting the interval to zero.
	if lease != nil {
		k.granted.Set(lease.Duration)
	}

	return nil
}

func (k *leaseKeeper) run(ctx *astral.Context) {
	for {
		timer := time.NewTimer(time.Duration(k.interval()))

		select {
		case <-ctx.Done():
			timer.Stop()
			return
		case <-timer.C:
		}

		// why the error is dropped: a renewal failing against a node that has gone
		// away is not this loop's to fix. The registrar's reconnect cycle re-runs
		// Hook and re-registers; until then there is no registration to renew.
		// Trying again on the next tick is the whole of the recovery.
		_ = k.renew(ctx)
	}
}

// interval is a third of the lease the node last granted, floored so a short
// lease cannot spin.
func (k *leaseKeeper) interval() astral.Duration {
	granted := k.granted.Get()
	if granted <= 0 {
		granted = k.requested
	}
	if granted <= 0 {
		granted = DefaultRegistrationLease
	}

	if interval := granted / renewDivisor; interval > minRenewInterval {
		return interval
	}

	return minRenewInterval
}
