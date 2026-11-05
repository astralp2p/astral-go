package user

import (
	"github.com/astralp2p/astral-go/api/auth"
	"github.com/astralp2p/astral-go/api/user"
	"github.com/astralp2p/astral-go/astral"
	"github.com/astralp2p/astral-go/astral/channel"
)

// Contract returns the target node's active contract. Mirrors the astrald op
// OpContract.
//
// The target answers only a caller directly linked to it, and refuses others
// with auth.ErrNotPermitted. It refuses without an active contract with
// user.ErrNoActiveContract.
func (client *Client) Contract(ctx *astral.Context) (contract *auth.SignedContract, err error) {
	ch, err := client.queryCh(ctx, user.OpContract, nil)
	if err != nil {
		return
	}
	defer ch.Close()

	err = ctxErr(ctx, ch.Switch(channel.Expect(&contract), channel.PassErrors, channel.WithContext(ctx)))
	return
}

// Contract calls the operation on the default client.
func Contract(ctx *astral.Context) (*auth.SignedContract, error) {
	return Default().Contract(ctx)
}
