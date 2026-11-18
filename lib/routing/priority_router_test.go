package routing

import (
	"context"
	"errors"
	"io"
	"testing"

	"github.com/astralp2p/astral-go/astral"
	"github.com/astralp2p/astral-go/lib/query"
)

// PriorityRouter walks its entries in ascending priority order. The walk stops
// on the first success, on a hard rejection, and on a dead context; it
// continues past any other error, so a failing router cannot mask a
// lower-priority one that would have answered.

// orderedRouter records the order in which routers were consulted.
type orderedRouter struct {
	name  string
	err   error
	order *[]string
}

func (r *orderedRouter) RouteQuery(*astral.Context, *astral.InFlightQuery, io.WriteCloser) (io.WriteCloser, error) {
	*r.order = append(*r.order, r.name)
	if r.err != nil {
		return nil, r.err
	}
	return nopWriteCloser{}, nil
}

func TestPriorityRouter_WalksInAscendingPriority(t *testing.T) {
	var order []string
	router := NewPriorityRouter("test")

	// added out of order; the router sorts
	router.Add(&orderedRouter{name: "low", err: astral.NewErrRouteNotFound(), order: &order}, 30)
	router.Add(&orderedRouter{name: "high", err: astral.NewErrRouteNotFound(), order: &order}, 10)
	router.Add(&orderedRouter{name: "medium", err: astral.NewErrRouteNotFound(), order: &order}, 20)

	routeTo(t, router, "op")

	want := []string{"high", "medium", "low"}
	if len(order) != len(want) {
		t.Fatalf("want %v, got %v", want, order)
	}
	for i := range want {
		if order[i] != want[i] {
			t.Fatalf("want %v, got %v", want, order)
		}
	}
}

// The first router that answers ends the walk.
func TestPriorityRouter_FirstSuccessWins(t *testing.T) {
	var order []string
	router := NewPriorityRouter("test")

	router.Add(&orderedRouter{name: "first", order: &order}, 10)
	router.Add(&orderedRouter{name: "second", order: &order}, 20)

	w, err := routeTo(t, router, "op")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if w == nil {
		t.Fatal("want a writer, got nil")
	}

	if len(order) != 1 || order[0] != "first" {
		t.Fatalf("want only the first router consulted, got %v", order)
	}
}

// A hard rejection is an answer: the target refused, so no lower-priority
// router is consulted.
func TestPriorityRouter_RejectionStopsTheWalk(t *testing.T) {
	var order []string
	router := NewPriorityRouter("test")

	router.Add(&orderedRouter{name: "first", err: astral.NewErrRejected(7), order: &order}, 10)
	router.Add(&orderedRouter{name: "second", order: &order}, 20)

	_, err := routeTo(t, router, "op")

	var rejected *astral.ErrRejected
	if !errors.As(err, &rejected) {
		t.Fatalf("want ErrRejected, got %v", err)
	}
	if rejected.Code != 7 {
		t.Fatalf("Code: want 7, got %d", rejected.Code)
	}

	if len(order) != 1 || order[0] != "first" {
		t.Fatalf("want the walk stopped at the rejection, got %v", order)
	}
}

// Any other error is not an answer: the walk continues.
func TestPriorityRouter_ContinuesPastNonRejectionErrors(t *testing.T) {
	var order []string
	router := NewPriorityRouter("test")

	router.Add(&orderedRouter{name: "first", err: errors.New("transport failed"), order: &order}, 10)
	router.Add(&orderedRouter{name: "second", order: &order}, 20)

	w, err := routeTo(t, router, "op")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if w == nil {
		t.Fatal("want the second router to answer, got nil")
	}

	if len(order) != 2 {
		t.Fatalf("want both routers consulted, got %v", order)
	}
}

// When nothing answers, the first router that tried and failed answers: its
// error outranks a later router that never knew the target.
func TestPriorityRouter_ExhaustedWalkReportsTheFirstFailure(t *testing.T) {
	var order []string
	router := NewPriorityRouter("test")
	failure := errors.New("transport failed")

	router.Add(&orderedRouter{name: "first", err: astral.NewErrRouteNotFound(), order: &order}, 10)
	router.Add(&orderedRouter{name: "second", err: failure, order: &order}, 20)
	router.Add(&orderedRouter{name: "third", err: errors.New("later failure"), order: &order}, 30)
	router.Add(&orderedRouter{name: "fourth", err: astral.NewErrRouteNotFound(), order: &order}, 40)

	w, err := routeTo(t, router, "op")

	if !errors.Is(err, failure) {
		t.Fatalf("want %v, got %v", failure, err)
	}
	if w != nil {
		t.Fatal("want no writer, got one")
	}
	if len(order) != 4 {
		t.Fatalf("want every router consulted, got %v", order)
	}
}

// Only a walk on which every router said not-found reports no route.
func TestPriorityRouter_EveryNotFoundReportsRouteNotFound(t *testing.T) {
	var order []string
	router := NewPriorityRouter("test")

	router.Add(&orderedRouter{name: "first", err: astral.NewErrRouteNotFound(), order: &order}, 10)
	router.Add(&orderedRouter{name: "second", err: astral.NewErrRouteNotFound(), order: &order}, 20)

	_, err := routeTo(t, router, "op")

	if !errors.Is(err, &astral.ErrRouteNotFound{}) {
		t.Fatalf("want ErrRouteNotFound, got %v", err)
	}
}

// A context that ended stops the walk with its own error: no later router is
// asked, and a router's not-found does not hide the cancel.
func TestPriorityRouter_DeadContextStopsTheWalk(t *testing.T) {
	var order []string
	router := NewPriorityRouter("test")

	ctx, cancel := astral.NewContext(nil).WithCancel()
	router.Add(&cancelingRouter{cancel: cancel, order: &order}, 10)
	router.Add(&orderedRouter{name: "second", order: &order}, 20)

	id := astral.GenerateIdentity()
	_, err := router.RouteQuery(ctx, astral.Launch(query.New(id, id, "op", nil)), nopWriteCloser{})

	if !errors.Is(err, context.Canceled) {
		t.Fatalf("want context.Canceled, got %v", err)
	}
	if len(order) != 1 {
		t.Fatalf("want the walk stopped at the cancel, got %v", order)
	}
}

// cancelingRouter cancels the routing context, then answers not-found.
type cancelingRouter struct {
	cancel context.CancelFunc
	order  *[]string
}

func (r *cancelingRouter) RouteQuery(*astral.Context, *astral.InFlightQuery, io.WriteCloser) (io.WriteCloser, error) {
	*r.order = append(*r.order, "canceling")
	r.cancel()
	return query.RouteNotFound()
}

// An empty router reports no route rather than succeeding vacuously.
func TestPriorityRouter_EmptyReportsRouteNotFound(t *testing.T) {
	_, err := routeTo(t, NewPriorityRouter("test"), "op")

	if !errors.Is(err, &astral.ErrRouteNotFound{}) {
		t.Fatalf("want ErrRouteNotFound, got %v", err)
	}
}

// Two routers at the same priority both run; neither is dropped.
func TestPriorityRouter_EqualPrioritiesBothRun(t *testing.T) {
	var order []string
	router := NewPriorityRouter("test")

	router.Add(&orderedRouter{name: "a", err: astral.NewErrRouteNotFound(), order: &order}, 10)
	router.Add(&orderedRouter{name: "b", err: astral.NewErrRouteNotFound(), order: &order}, 10)

	routeTo(t, router, "op")

	if len(order) != 2 {
		t.Fatalf("want both routers consulted, got %v", order)
	}
}

func TestPriorityRouter_String(t *testing.T) {
	if got := NewPriorityRouter("named").String(); got != "named" {
		t.Fatalf("want %q, got %q", "named", got)
	}
	if got := NewPriorityRouter("").String(); got != "PriorityRouter" {
		t.Fatalf("want %q, got %q", "PriorityRouter", got)
	}
}

var _ PriorityAdder = &PriorityRouter{}
