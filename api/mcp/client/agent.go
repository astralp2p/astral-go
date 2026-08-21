package mcp

import (
	"github.com/astralp2p/astral-go/api/mcp"
	"github.com/astralp2p/astral-go/astral"
	"github.com/astralp2p/astral-go/astral/channel"
	"github.com/astralp2p/astral-go/lib/query"
)

// Agent returns one agent's record without its access token. id takes a hex
// public key or an alias resolved via the directory.
//
// The node answers "unknown identity" when id resolves to no identity, and
// "agent not found" when it resolves but no agent is registered under it.
func (client *Client) Agent(ctx *astral.Context, id string) (info *mcp.AgentInfo, err error) {
	ch, err := client.queryCh(ctx, mcp.MethodAgent, query.Args{"id": id})
	if err != nil {
		return
	}
	defer ch.Close()

	err = ch.Switch(channel.Expect(&info), channel.PassErrors)
	return
}

// Agent calls the operation on the default client.
func Agent(ctx *astral.Context, id string) (*mcp.AgentInfo, error) {
	return Default().Agent(ctx, id)
}
