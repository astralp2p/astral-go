package dir

import (
	"github.com/astralp2p/astral-go/api/dir"
	"github.com/astralp2p/astral-go/astral"
	"github.com/astralp2p/astral-go/astral/channel"
	"github.com/astralp2p/astral-go/lib/query"
)

func SetAlias(ctx *astral.Context, identity *astral.Identity, alias string) error {
	return Default().SetAlias(ctx, identity, alias)
}

func (client *Client) SetAlias(ctx *astral.Context, identity *astral.Identity, alias string) error {
	ch, err := client.queryCh(ctx, dir.MethodSetAlias, query.Args{
		"id":    identity,
		"alias": alias,
	})
	if err != nil {
		return err
	}

	return ch.Switch(channel.ExpectAck, channel.PassErrors)
}
