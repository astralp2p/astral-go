package dir

import (
	"github.com/astralp2p/astral-go/api/dir"
	"github.com/astralp2p/astral-go/astral"
	"github.com/astralp2p/astral-go/astral/channel"
)

func AliasMap(ctx *astral.Context) (*dir.AliasMap, error) {
	return Default().AliasMap(ctx)
}

func (client *Client) AliasMap(ctx *astral.Context) (am *dir.AliasMap, err error) {
	// query
	ch, err := client.queryCh(ctx, dir.MethodAliasMap, nil)
	if err != nil {
		return nil, err
	}

	// response
	err = ch.Switch(channel.Expect(&am), channel.PassErrors)
	return
}
