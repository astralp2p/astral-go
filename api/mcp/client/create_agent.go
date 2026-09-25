package mcp

import (
	"github.com/astralp2p/astral-go/api/mcp"
	"github.com/astralp2p/astral-go/astral"
	"github.com/astralp2p/astral-go/astral/channel"
	"github.com/astralp2p/astral-go/lib/query"
)

// CreateAgent mints a new agent: a messaging participant — a fresh identity
// with a signed relay contract, a signed contract letting the node host its
// mailbox, an optional alias and an access token — and the agent record that
// keeps the token. The agent presents that token to the MCP endpoint, which
// admits any valid apphost access token. The returned Agent carries it, and it
// is the only response that does — ListAgents is the sole way to recover it
// afterwards. When the record cannot be stored, the node deletes the
// participant again and answers the record's error.
//
// An empty alias binds none; the node generates no alias, because an alias is
// node-global and a name the caller did not choose contends in a namespace it
// does not own. A zero duration leaves the token's lifetime to the node's
// configured default.
//
// The agent it mints reaches and is reached by nobody until something permits
// it: the node holds no reachability, so an agent sends mail, takes mail or
// starts a query where a handler, a contract or an external authority says so.
func (client *Client) CreateAgent(ctx *astral.Context, alias string, duration astral.Duration) (agent *mcp.Agent, err error) {
	// why the keys are lower-case: the node snake-cases and lower-cases op
	// argument names and binds by that name. A capitalised key reaches the wire
	// verbatim, matches nothing, and is dropped without complaint.
	args := query.Args{}
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
func CreateAgent(ctx *astral.Context, alias string, duration astral.Duration) (*mcp.Agent, error) {
	return Default().CreateAgent(ctx, alias, duration)
}
