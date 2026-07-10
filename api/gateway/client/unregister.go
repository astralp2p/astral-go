package gateway

import (
	gw "github.com/astralp2p/astral-go/api/gateway"
	"github.com/astralp2p/astral-go/astral"
	"github.com/astralp2p/astral-go/astral/channel"
	"github.com/astralp2p/astral-go/lib/query"
)

func (c *Client) Unregister(ctx *astral.Context) error {
	ch, err := c.queryCh(ctx, gw.MethodNodeUnregister, query.Args{})
	if err != nil {
		return err
	}
	defer ch.Close()

	return ch.Switch(
		channel.ExpectAck,
		channel.PassErrors,
	)
}
