package mcp

import (
	"github.com/astralp2p/astral-go/api/mcp"
	"github.com/astralp2p/astral-go/astral"
	"github.com/astralp2p/astral-go/astral/channel"
	"github.com/astralp2p/astral-go/lib/query"
)

// SetExposed opens or closes an agent to callers other than itself. id takes a
// hex public key or an alias resolved via the directory.
//
// The flag is the whole of an agent's reachability: a query addressed to a
// closed agent is answered route_not_found. Closing an open agent takes effect
// on conversations already under way — the agent's queued queries are dropped
// and its live sessions closed.
func (client *Client) SetExposed(ctx *astral.Context, id string, exposed bool) error {
	ch, err := client.queryCh(ctx, mcp.MethodSetExposed, query.Args{
		"id":      id,
		"exposed": exposed,
	})
	if err != nil {
		return err
	}
	defer ch.Close()

	return ch.Switch(channel.ExpectAck, channel.PassErrors)
}

// SetExposed calls the operation on the default client.
func SetExposed(ctx *astral.Context, id string, exposed bool) error {
	return Default().SetExposed(ctx, id, exposed)
}
