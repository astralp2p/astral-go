package mcp

import (
	"encoding/json"
	"io"

	"github.com/astralp2p/astral-go/astral"
)

// Agent is an AI agent registered with the mcp module: an identity minted by
// the node plus the access token the agent presents to the MCP endpoint.
//
// The token is a credential. AgentInfo is the same record without it, and is
// what a caller reads about an agent whose credentials it does not hold.
// Returned by mcp.create_agent and mcp.list_agents.
type Agent struct {
	Identity  *astral.Identity
	Alias     astral.String8
	Token     astral.String8
	ExpiresAt astral.Time
}

// astral

var _ astral.Object = &Agent{}

func (a Agent) ObjectType() string { return "mcp.agent" }

func (a Agent) WriteTo(w io.Writer) (n int64, err error) {
	return astral.Objectify(&a).WriteTo(w)
}

func (a *Agent) ReadFrom(r io.Reader) (n int64, err error) {
	return astral.Objectify(a).ReadFrom(r)
}

// json

func (a Agent) MarshalJSON() ([]byte, error) {
	type alias Agent
	return json.Marshal(alias(a))
}

func (a *Agent) UnmarshalJSON(bytes []byte) error {
	type alias Agent
	var v alias

	err := json.Unmarshal(bytes, &v)
	if err != nil {
		return err
	}

	*a = Agent(v)
	return nil
}

func init() {
	astral.MustAdd(&Agent{})
}
