package mcp

import (
	"github.com/astralp2p/astral-go/api/mcp"
	"github.com/astralp2p/astral-go/astral"
	"github.com/astralp2p/astral-go/astral/channel"
	"github.com/astralp2p/astral-go/lib/query"
)

// SetVisible opens or closes an agent to callers other than itself. id takes a
// hex public key or an alias resolved via the directory.
//
// The flag is the whole of an agent's reachability: a query addressed to a
// closed agent is answered route_not_found. Closing an open agent takes effect
// on conversations already under way — the agent's queued queries are dropped
// and its live sessions closed.
func (client *Client) SetVisible(ctx *astral.Context, id string, visible bool) error {
	ch, err := client.queryCh(ctx, mcp.MethodSetVisible, query.Args{
		"id":      id,
		"visible": visible,
	})
	if err != nil {
		return err
	}
	defer ch.Close()

	return ch.Switch(channel.ExpectAck, channel.PassErrors)
}

// SetVisible calls the operation on the default client.
func SetVisible(ctx *astral.Context, id string, visible bool) error {
	return Default().SetVisible(ctx, id, visible)
}
