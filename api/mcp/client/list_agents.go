package mcp

import (
	"github.com/astralp2p/astral-go/api/mcp"
	"github.com/astralp2p/astral-go/astral"
	"github.com/astralp2p/astral-go/astral/channel"
)

// ListAgents streams every registered agent, access tokens included.
//
// The token is streamed by design: CreateAgent returns a token once, and this
// operation is the only way to recover one that was lost. The result is a
// credential-bearing enumeration of every tenant's agents on the node, and must
// be handled as the tokens themselves are. A read that does not need the tokens
// is Agent, which answers per agent and withholds them.
func (client *Client) ListAgents(ctx *astral.Context) (agents []*mcp.Agent, err error) {
	ch, err := client.queryCh(ctx, mcp.MethodListAgents, nil)
	if err != nil {
		return
	}
	defer ch.Close()

	err = ch.Switch(
		channel.Collect[*mcp.Agent](&agents),
		channel.PassErrors,
		channel.BreakOnEOS,
	)
	return
}

// ListAgents calls the operation on the default client.
func ListAgents(ctx *astral.Context) ([]*mcp.Agent, error) {
	return Default().ListAgents(ctx)
}
