package user

import (
	"github.com/astralp2p/astral-go/astral"
	"github.com/astralp2p/astral-go/lib/query"
)

// localRouter routes queries in-process, so the real client talks to the real op
// over a pipe with no node involved.
type localRouter struct {
	inner astral.Router
	id    *astral.Identity
}

func (r *localRouter) RouteQuery(ctx *astral.Context, q *astral.InFlightQuery) (astral.Conn, error) {
	return query.RouteInFlight(ctx, r.inner, q)
}

func (r *localRouter) GuestID() *astral.Identity { return r.id }
func (r *localRouter) HostID() *astral.Identity  { return r.id }
