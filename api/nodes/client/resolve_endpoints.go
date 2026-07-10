package nodes

import (
	"github.com/astralp2p/astral-go/api/nodes"
	"github.com/astralp2p/astral-go/astral"
	"github.com/astralp2p/astral-go/astral/channel"
	"github.com/astralp2p/astral-go/lib/query"
)

func (client *Client) ResolveEndpoints(ctx *astral.Context, identity *astral.Identity) ([]*nodes.EndpointWithTTL, error) {
	ch, err := client.queryCh(ctx, nodes.MethodResolveEndpoints, query.Args{
		"id": identity,
	})
	if err != nil {
		return nil, err
	}

	var endpoints []*nodes.EndpointWithTTL
	err = ch.Switch(
		channel.Collect(&endpoints),
		channel.BreakOnEOS,
		func(msg *astral.ErrorMessage) error {
			return msg
		},
	)

	return endpoints, err
}
