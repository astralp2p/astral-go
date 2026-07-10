package auth

import (
	"github.com/astralp2p/astral-go/api/auth"
	"github.com/astralp2p/astral-go/astral"
	"github.com/astralp2p/astral-go/astral/channel"
	"github.com/astralp2p/astral-go/lib/query"
)

func (c *Client) IndexContract(ctx *astral.Context, objectID *astral.ObjectID) error {
	ch, err := c.queryCh(ctx, auth.MethodIndex, query.Args{"id": objectID})
	if err != nil {
		return err
	}
	defer ch.Close()

	var ack *astral.Ack
	return ch.Switch(channel.Expect(&ack), channel.PassErrors)
}

func IndexContract(ctx *astral.Context, objectID *astral.ObjectID) error {
	return Default().IndexContract(ctx, objectID)
}
