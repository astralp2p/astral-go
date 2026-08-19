package user

import (
	"github.com/astralp2p/astral-go/api/auth"
	"github.com/astralp2p/astral-go/api/user"
	"github.com/astralp2p/astral-go/astral"
	"github.com/astralp2p/astral-go/astral/channel"
	"github.com/astralp2p/astral-go/lib/query"
)

// Adopt issues target a swarm-membership contract signed by the target node's
// active user, indexes it, and pushes it to the local swarm.
//
// Requires an active contract (code 2 otherwise) and authorization for
// user.AdminSwarmAction (code 4 otherwise) — the user is always authorized, other
// identities via authorizers.
func (client *Client) Adopt(ctx *astral.Context, target string) (signed *auth.SignedContract, err error) {
	ch, err := client.queryCh(ctx, user.OpAdopt, query.Args{"target": target})
	if err != nil {
		return
	}
	defer ch.Close()

	err = ch.Switch(channel.Expect(&signed), channel.PassErrors)
	return
}

// Adopt calls the operation on the default client.
func Adopt(ctx *astral.Context, target string) (*auth.SignedContract, error) {
	return Default().Adopt(ctx, target)
}
