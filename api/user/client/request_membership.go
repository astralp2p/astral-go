package user

import (
	"github.com/astralp2p/astral-go/api/auth"
	"github.com/astralp2p/astral-go/api/user"
	"github.com/astralp2p/astral-go/astral"
	"github.com/astralp2p/astral-go/astral/channel"
)

// RequestMembership asks the target node's active user to admit the caller into
// its swarm and returns the signed membership contract.
//
// The target rejects with code 2 when it holds no active contract to invite
// under, and replies with user.ErrRequestDeclined when its join-request policy
// refuses the caller.
func (client *Client) RequestMembership(ctx *astral.Context) (signed *auth.SignedContract, err error) {
	ch, err := client.queryCh(ctx, user.OpRequestMembership, nil)
	if err != nil {
		return
	}
	defer ch.Close()

	err = ch.Switch(channel.Expect(&signed), channel.PassErrors)
	return
}

// RequestMembership calls the operation on the default client.
func RequestMembership(ctx *astral.Context) (*auth.SignedContract, error) {
	return Default().RequestMembership(ctx)
}
