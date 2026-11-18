package objects

import (
	"github.com/astralp2p/astral-go/api/objects"
	"github.com/astralp2p/astral-go/astral"
	"github.com/astralp2p/astral-go/astral/channel"
)

// Push sends the object and fails with objects.ErrPushRejected if the node
// receives it and does not accept it.
func (client *Client) Push(ctx *astral.Context, object astral.Object) error {
	ch, err := client.queryCh(ctx, objects.MethodPush, nil)
	if err != nil {
		return err
	}
	defer ch.Close()

	err = ch.Send(object)
	if err != nil {
		return err
	}

	return ch.Switch(
		func(result *astral.Bool) error {
			if *result {
				return channel.ErrBreak
			}
			return objects.ErrPushRejected
		}, channel.PassErrors, channel.WithContext(ctx))
}

func Push(ctx *astral.Context, object astral.Object) error {
	return Default().Push(ctx, object)
}
