package apps

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/astralp2p/astral-go/api/objects"
	"github.com/astralp2p/astral-go/astral"
)

// TestLeaseKeeperIntervalFollowsTheGrantNotTheRequest covers the reason the node
// returns a lease at all: it clamps what the app asked for, so an app that timed
// its renewals off its own request would renew long after the registration it is
// renewing had already lapsed.
func TestLeaseKeeperIntervalFollowsTheGrantNotTheRequest(t *testing.T) {
	k := newLeaseKeeper(nil, astral.Duration(time.Hour))
	k.granted.Set(astral.Duration(30 * time.Minute))

	if got, want := k.interval(), astral.Duration(10*time.Minute); got != want {
		t.Fatalf("interval: got %v, want %v", time.Duration(got), time.Duration(want))
	}
}

// TestLeaseKeeperIntervalFallsBackToTheRequestBeforeAnyGrant: interval is read
// once before the first registration answers, and a zero there would mean a timer
// firing immediately in a tight loop.
func TestLeaseKeeperIntervalFallsBackToTheRequestBeforeAnyGrant(t *testing.T) {
	k := newLeaseKeeper(nil, astral.Duration(time.Hour))

	if got, want := k.interval(), astral.Duration(20*time.Minute); got != want {
		t.Fatalf("interval: got %v, want %v", time.Duration(got), time.Duration(want))
	}
}

// TestLeaseKeeperIntervalFloorsAShortLease: a node granting a lease of a few
// milliseconds would otherwise have the app renewing continuously, which costs
// more than the stale registration the lease exists to prevent.
func TestLeaseKeeperIntervalFloorsAShortLease(t *testing.T) {
	k := newLeaseKeeper(nil, 0)
	k.granted.Set(astral.Duration(3 * time.Millisecond))

	if got := k.interval(); got != minRenewInterval {
		t.Fatalf("interval: got %v, want the floor %v", time.Duration(got), time.Duration(minRenewInterval))
	}
}

// TestLeaseKeeperHookKeepsRegisteringWithoutAppCode is the point of the keeper:
// an app that registers once must stay registered across lease after lease
// without its author writing a renewal loop. The granted lease is tiny so the
// test spans several of them in real time.
func TestLeaseKeeperHookKeepsRegisteringWithoutAppCode(t *testing.T) {
	var (
		mu    sync.Mutex
		calls int
	)

	register := func(ctx *astral.Context, requested astral.Duration) (*objects.RegistrationLease, error) {
		mu.Lock()
		calls++
		mu.Unlock()

		return &objects.RegistrationLease{
			Duration:  astral.Duration(3 * time.Millisecond),
			ExpiresAt: astral.Time(time.Now().Add(3 * time.Millisecond)),
		}, nil
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	k := newLeaseKeeper(register, 0)

	// the granted lease is far below minRenewInterval, so the keeper renews at the
	// floor. The waits below are sized to that floor, not to the lease.
	if err := k.Hook(astral.NewContext(ctx)); err != nil {
		t.Fatalf("hook: %v", err)
	}

	mu.Lock()
	afterHook := calls
	mu.Unlock()

	if afterHook != 1 {
		t.Fatalf("registrations after the hook: got %v, want exactly 1", afterHook)
	}

	// one renewal at the floor, plus margin
	time.Sleep(time.Duration(minRenewInterval) + 500*time.Millisecond)

	mu.Lock()
	total := calls
	mu.Unlock()

	if total < 2 {
		t.Fatalf("registrations after one renewal interval: got %v, want at least 2 — the keeper is not renewing", total)
	}
}

// TestLeaseKeeperHookReportsAFailedRegistration: the registrar's hook contract
// makes an error abort the reconnect cycle and retry the connection, so a
// registration that failed must not be reported as one that succeeded.
func TestLeaseKeeperHookReportsAFailedRegistration(t *testing.T) {
	wantErr := errors.New("not authorized")

	register := func(ctx *astral.Context, requested astral.Duration) (*objects.RegistrationLease, error) {
		return nil, wantErr
	}

	k := newLeaseKeeper(register, DefaultRegistrationLease)

	if err := k.Hook(astral.NewContext(context.Background())); !errors.Is(err, wantErr) {
		t.Fatalf("hook: got %v, want %v", err, wantErr)
	}
}

// TestLeaseKeeperHookStartsOneRenewalLoop: the registrar re-runs every hook on
// each reconnect, and a loop started per run would multiply the renewal traffic
// by the number of reconnects the app has survived.
func TestLeaseKeeperHookStartsOneRenewalLoop(t *testing.T) {
	var (
		mu    sync.Mutex
		calls int
	)

	register := func(ctx *astral.Context, requested astral.Duration) (*objects.RegistrationLease, error) {
		mu.Lock()
		calls++
		mu.Unlock()

		return &objects.RegistrationLease{Duration: astral.Duration(time.Hour)}, nil
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	k := newLeaseKeeper(register, DefaultRegistrationLease)

	// three reconnects, so three hook runs
	for i := 0; i < 3; i++ {
		if err := k.Hook(astral.NewContext(ctx)); err != nil {
			t.Fatalf("hook %v: %v", i, err)
		}
	}

	// the granted lease is an hour, so no renewal is due; every call so far is a
	// hook registering, and any extra would be a second loop firing early.
	mu.Lock()
	total := calls
	mu.Unlock()

	if total != 3 {
		t.Fatalf("registrations after 3 hook runs: got %v, want 3", total)
	}
}
