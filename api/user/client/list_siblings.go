package user

import (
	"github.com/astralp2p/astral-go/api/user"
	"github.com/astralp2p/astral-go/astral"
	"github.com/astralp2p/astral-go/astral/channel"
	"github.com/astralp2p/astral-go/lib/query"
)

// ListSiblings lists the identities of the swarm nodes the target node
// currently holds a link with. Where SwarmStatus reports membership and
// liveness together, this reports the linked subset alone.
//
// zone is the zone mask the target includes while enumerating; an empty zone
// leaves the target's own default in place.
func (client *Client) ListSiblings(ctx *astral.Context, zone astral.Zone) ([]*astral.Identity, error) {
	args := query.Args{}
	if zone != 0 {
		args["zone"] = zone
	}

	ch, err := client.queryCh(ctx, user.OpListSiblings, args)
	if err != nil {
		return nil, err
	}
	defer ch.Close()

	var siblings []*astral.Identity
	err = ch.Switch(
		channel.Collect(&siblings),
		channel.BreakOnEOS,
		func(msg *astral.ErrorMessage) error {
			return msg
		},
	)

	return siblings, err
}

// ListSiblings calls the operation on the default client.
func ListSiblings(ctx *astral.Context, zone astral.Zone) ([]*astral.Identity, error) {
	return Default().ListSiblings(ctx, zone)
}
