/*
Package mcp describes a module that registers AI agents on a node and serves
them the astral network over the Model Context Protocol.

An agent is a node-minted identity, a signed relay contract, an optional alias
and an access token the agent presents to the node's MCP endpoint as a bearer
credential. One node holds the agents of many tenants and knows no relation
between them, so it holds no reachability of its own: a call between two agents
crosses mod.mcp.call_agent_action and mod.mcp.answer_agent_action, and the auth
module answers both.

Every operation is local-only. A query arriving over a link is rejected, and the
shell module — the sole mount point for every module's operations — rejects a
query carrying the mcp origin, so an agent reaches none of these operations on
its own host node.

An agent answers one query of its own. MethodMessage carries a Message to the
agent's identity, and the agent's node stores it in that agent's inbox. It is
addressed to an agent rather than to a node, so it is the one query here a
caller reaches over a link.
*/
package mcp

const (
	MethodCreateAgent     = "mcp.create_agent"
	MethodAgent           = "mcp.agent"
	MethodDisconnectAgent = "mcp.disconnect_agent"
	MethodDeleteAgent     = "mcp.delete_agent"
	MethodListAgents      = "mcp.list_agents"
)

// MethodMessage is the query that delivers a Message, addressed to the
// recipient agent's identity. It is not an operation: no node serves it, and
// the agent's own node answers it on the agent's behalf.
const MethodMessage = "mcp.message"
