package apphost

import (
	"github.com/astralp2p/astral-go/api/apphost"
	"github.com/astralp2p/astral-go/api/auth"
	"github.com/astralp2p/astral-go/astral"
	"github.com/astralp2p/astral-go/astral/channel"
	"github.com/astralp2p/astral-go/lib/query"
)

func NewAppContract(ctx *astral.Context, id *astral.Identity, duration astral.Duration) (*auth.Contract, error) {
	return Default().NewAppContract(ctx, id, duration)
}

func (client *Client) NewAppContract(ctx *astral.Context, id *astral.Identity, duration astral.Duration) (contract *auth.Contract, err error) {
	ch, err := client.queryCh(ctx, apphost.MethodNewAppContract, query.Args{"ID": id, "Duration": duration})
	if err != nil {
		return
	}
	defer ch.Close()
	err = ch.Switch(channel.Expect(&contract), channel.PassErrors)
	return
}
