package dir

import (
	"strings"

	"github.com/astralp2p/astral-go/api/dir"
	"github.com/astralp2p/astral-go/astral"
	"github.com/astralp2p/astral-go/astral/channel"
	"github.com/astralp2p/astral-go/lib/query"
)

func ApplyFilters(ctx *astral.Context, identity *astral.Identity, filters ...string) (bool, error) {
	return Default().ApplyFilters(ctx, identity, filters...)
}

// ApplyFilters tests identity against the named server-side filters and returns
// true if the identity matches any of them. Unknown filter names are ignored; an
// empty filter list returns false. A nil identity tests the caller's own identity.
func (client *Client) ApplyFilters(ctx *astral.Context, identity *astral.Identity, filters ...string) (bool, error) {
	ch, err := client.queryCh(ctx, dir.MethodApplyFilters, query.Args{
		"id":      identity,
		"filters": strings.Join(filters, ","),
	})
	if err != nil {
		return false, err
	}

	var match *astral.Bool
	err = ch.Switch(channel.Expect(&match), channel.PassErrors)
	if err != nil {
		return false, err
	}

	// why: Switch returns nil on a clean EOF, so a server that accepts and closes
	// without answering leaves match nil and the conversion below dereferences it.
	if match == nil {
		return false, astral.NewError("dir.apply_filters: no response")
	}

	return (bool)(*match), nil
}
