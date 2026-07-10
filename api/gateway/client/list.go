package gateway

import (
	gw "github.com/astralp2p/astral-go/api/gateway"
	"github.com/astralp2p/astral-go/astral"
	"github.com/astralp2p/astral-go/astral/channel"
	"github.com/astralp2p/astral-go/lib/query"
)

func (c *Client) List(ctx *astral.Context) ([]*astral.Identity, error) {
	ch, err := c.queryCh(ctx, gw.MethodNodeList, query.Args{})
	if err != nil {
		return nil, err
	}
	defer ch.Close()

	var list []*astral.Identity
	err = ch.Switch(
		channel.Collect(&list),
		channel.BreakOnEOS,
		channel.PassErrors,
	)

	return list, err
}
