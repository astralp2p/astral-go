package apphost

import (
	"github.com/astralp2p/astral-go/api/apphost"
	"github.com/astralp2p/astral-go/astral"
	"github.com/astralp2p/astral-go/astral/channel"
	"github.com/astralp2p/astral-go/lib/query"
)

func (client *Client) UnholdObject(ctx *astral.Context, objectID *astral.ObjectID) error {
	ch, err := client.queryCh(ctx, apphost.MethodUnholdObject, query.Args{
		"id": objectID,
	})
	if err != nil {
		return err
	}
	defer ch.Close()

	return ch.Switch(channel.ExpectAck, channel.PassErrors, channel.WithContext(ctx))
}

func UnholdObject(ctx *astral.Context, objectID *astral.ObjectID) error {
	return Default().UnholdObject(ctx, objectID)
}
