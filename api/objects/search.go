package objects

import (
	"github.com/astralp2p/astral-go/astral"
)

// Searcher is used to search for objects matching a query
type Searcher interface {
	SearchObject(ctx *astral.Context, query SearchQuery) (<-chan *SearchResult, error)
}

// Search is one search run: the caller, the query, and the nodes the caller names as sources.
type Search struct {
	CallerID    *astral.Identity
	Query       SearchQuery
	Sources     []*astral.Identity
	SourcesOnly bool // ask only Sources, not the registered searchers
}
