package user

import (
	"github.com/astralp2p/astral-go/api/auth"
	"github.com/astralp2p/astral-go/api/user"
	"github.com/astralp2p/astral-go/astral"
	"github.com/astralp2p/astral-go/astral/channel"
	"github.com/astralp2p/astral-go/lib/query"
)

// NewContract asks the remote user module for an unsigned node contract, the
// node's PrivilegedMembership from the user identified by alias, which the
// user key signs in the setup ceremony.
func (client *Client) NewContract(ctx *astral.Context, alias string) (contract *auth.Contract, err error) {
	ch, err := client.queryCh(ctx, user.OpNewNodeContract, query.Args{"user": alias})
	if err != nil {
		return
	}
	defer ch.Close()
	err = ctxErr(ctx, ch.Switch(channel.Expect(&contract), channel.PassErrors, channel.WithContext(ctx)))
	return
}
