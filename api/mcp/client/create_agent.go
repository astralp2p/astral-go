package mcp

import (
	"github.com/astralp2p/astral-go/api/mcp"
	"github.com/astralp2p/astral-go/astral"
	"github.com/astralp2p/astral-go/astral/channel"
	"github.com/astralp2p/astral-go/lib/query"
)

// CreateAgent mints a new agent: a fresh identity with a signed relay contract,
// an optional alias, and the access token the agent presents to the MCP
// endpoint. The returned Agent carries that token, and it is the only response
// that does — ListAgents is the sole way to recover it afterwards.
//
// An empty alias binds none; the node generates no alias, because an alias is
// node-global and a name the caller did not choose contends in a namespace it
// does not own. A zero duration leaves the token's lifetime to the node's
// configured default. exposed opens the agent to other callers at creation;
// false leaves it closed until SetExposed opens it.
func (client *Client) CreateAgent(ctx *astral.Context, alias string, duration astral.Duration, exposed bool) (agent *mcp.Agent, err error) {
	// why the keys are lower-case: the node snake-cases and lower-cases op
	// argument names and binds by that name. A capitalised key reaches the wire
	// verbatim, matches nothing, and is dropped without complaint.
	args := query.Args{"exposed": exposed}
	if alias != "" {
		args["alias"] = alias
	}
	if duration != 0 {
		args["duration"] = duration
	}

	ch, err := client.queryCh(ctx, mcp.MethodCreateAgent, args)
	if err != nil {
		return
	}
	defer ch.Close()

	err = ch.Switch(channel.Expect(&agent), channel.PassErrors)
	return
}

// CreateAgent calls the operation on the default client.
func CreateAgent(ctx *astral.Context, alias string, duration astral.Duration, exposed bool) (*mcp.Agent, error) {
	return Default().CreateAgent(ctx, alias, duration, exposed)
}
