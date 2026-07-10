package apphost

import (
	"github.com/astralp2p/astral-go/api/apphost"
	"github.com/astralp2p/astral-go/astral"
	"github.com/astralp2p/astral-go/astral/channel"
	"github.com/astralp2p/astral-go/lib/query"
)

// HoldObject pins the object in the apphost for as long as ctx is live.
func (client *Client) HoldObject(ctx *astral.Context, objectID *astral.ObjectID) error {
	ch, err := client.queryCh(ctx, apphost.MethodHoldObject, query.Args{
		"id": objectID,
	})
	if err != nil {
		return err
	}
	defer ch.Close()

	return ch.Switch(channel.ExpectAck, channel.PassErrors, channel.WithContext(ctx))
}

func HoldObject(ctx *astral.Context, objectID *astral.ObjectID) error {
	return Default().HoldObject(ctx, objectID)
}
