package auth

import (
	"github.com/astralp2p/astral-go/astral"
)

// AuthorizeAsk puts one authorization question to an authority and answers what
// it said.
//
// An implementation carries the question to whoever decides it, by whatever
// means that authority is reached — a request, a query, a local lookup. What it
// does not do is decide: the answer is the authority's.
//
// An error is a question that could not be put, which is not an answer. A caller
// that cannot reach its authority has been permitted nothing, so an error
// refuses rather than defaults.
type AuthorizeAsk interface {
	Ask(ctx *astral.Context, action ActionObject) (allow bool, err error)
}
