package routing

import (
	"cmp"
	"errors"
	"io"

	"github.com/astralp2p/astral-go/astral"
	"github.com/astralp2p/astral-go/lib/query"
	"github.com/astralp2p/astral-go/sig"
)

var _ astral.Router = &PriorityRouter{}

// PriorityRouter tries registered routers in ascending priority order; the
// first successful match or hard rejection (ErrRejected) wins.
type PriorityRouter struct {
	Name    string
	entries sig.Set[*Entry]
}

type PriorityAdder interface {
	Add(r astral.Router, prio int) error
}

var _ PriorityAdder = &PriorityRouter{}

type Entry struct {
	Router astral.Router
	Prio   int
}

func NewPriorityRouter(name string) *PriorityRouter {
	return &PriorityRouter{Name: name}
}

// RouteQuery iterates entries in priority order and stops on success or
// ErrRejected. When no router answers, it returns the first error that is not
// ErrRouteNotFound, and ErrRouteNotFound only when every router said so.
//
// why the first real failure wins: a router that tried and failed (a dead
// link, a deadline) tells the caller more than a later router that never knew
// the target; shell, for one, answers ErrRouteNotFound for every remote target.
func (router *PriorityRouter) RouteQuery(ctx *astral.Context, q *astral.InFlightQuery, w io.WriteCloser) (io.WriteCloser, error) {
	var failed error

	for _, r := range router.entries.Clone() {
		rw, err := r.Router.RouteQuery(ctx, q, w)
		switch {
		case err == nil, errors.Is(err, &astral.ErrRejected{}):
			return rw, err
		case ctx.Err() != nil:
			// why: a dead context fails every later router too.
			return nil, ctx.Err()
		case failed == nil && !errors.Is(err, &astral.ErrRouteNotFound{}):
			failed = err
		}
	}

	if failed != nil {
		return nil, failed
	}

	return query.RouteNotFound()
}

// Add registers a router at the given priority and re-sorts the entry set so
// iteration order matches ascending priority.
func (router *PriorityRouter) Add(r astral.Router, prio int) error {
	router.entries.Add(&Entry{Router: r, Prio: prio})

	router.entries.Sort(func(a, b *Entry) int {
		return cmp.Compare(a.Prio, b.Prio)
	})

	return nil
}

func (router *PriorityRouter) String() string {
	if router.Name == "" {
		return "PriorityRouter"
	}
	return router.Name
}
