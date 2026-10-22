package user

import (
	"github.com/astralp2p/astral-go/api/auth"
	"github.com/astralp2p/astral-go/api/user"
	"github.com/astralp2p/astral-go/astral"
	"github.com/astralp2p/astral-go/astral/channel"
	"github.com/astralp2p/astral-go/lib/query"
)

// Join makes the unclaimed target node request membership from node, a node of
// a swarm, and returns the membership contract the target now holds as its
// active contract. Mirrors the astrald op OpJoin.
//
// The target rejects with code 2 when it already has an active contract, with
// code 3 when node does not resolve or names the target, with code 4 when the
// caller is not the target itself, and rejects a query of any origin other
// than local.
func (client *Client) Join(ctx *astral.Context, node string) (signed *auth.SignedContract, err error) {
	ch, err := client.queryCh(ctx, user.OpJoin, query.Args{"node": node})
	if err != nil {
		return
	}
	defer ch.Close()

	err = ch.Switch(channel.Expect(&signed), channel.PassErrors)
	return
}

// Join calls the operation on the default client.
func Join(ctx *astral.Context, node string) (*auth.SignedContract, error) {
	return Default().Join(ctx, node)
}
