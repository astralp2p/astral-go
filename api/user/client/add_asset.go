package user

import (
	"github.com/astralp2p/astral-go/api/user"
	"github.com/astralp2p/astral-go/astral"
	"github.com/astralp2p/astral-go/astral/channel"
	"github.com/astralp2p/astral-go/lib/query"
)

// AddAsset adds id to the target node's user asset list.
func (client *Client) AddAsset(ctx *astral.Context, id *astral.ObjectID) (err error) {
	ch, err := client.queryCh(ctx, user.OpAddAsset, query.Args{"id": id})
	if err != nil {
		return
	}
	defer ch.Close()

	return ch.Switch(channel.ExpectAck, channel.PassErrors)
}

// AddAsset calls the operation on the default client.
func AddAsset(ctx *astral.Context, id *astral.ObjectID) error {
	return Default().AddAsset(ctx, id)
}
