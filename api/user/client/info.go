package user

import (
	"github.com/astralp2p/astral-go/api/user"
	"github.com/astralp2p/astral-go/astral"
	"github.com/astralp2p/astral-go/astral/channel"
)

// Info returns the target node's active-contract identity and metadata.
//
// The target rejects with code 2 when it holds no active contract, and with
// code 4 when the caller is not authorized for user.InfoAction (the user and
// swarm members always are, other identities via authorizers).
func (client *Client) Info(ctx *astral.Context) (info *user.Info, err error) {
	ch, err := client.queryCh(ctx, user.OpInfo, nil)
	if err != nil {
		return
	}
	defer ch.Close()

	err = ch.Switch(channel.Expect(&info), channel.PassErrors)
	return
}

// Info calls the operation on the default client.
func Info(ctx *astral.Context) (*user.Info, error) {
	return Default().Info(ctx)
}
