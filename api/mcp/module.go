/*
Package mcp describes a module that registers AI agents on a node and serves
them the astral network over the Model Context Protocol.

An agent is a messaging participant: an identity the messaging module mints,
with its signed relay contract, optional alias and access token, which the
agent presents to the node's MCP endpoint as a bearer credential. Its mail
lives in the messaging module and is described by package messaging; this
module keeps the agent's record and serves the endpoint.

One node holds the agents of many tenants and knows no relation between them,
so it holds no reachability of its own: a query an agent starts through the
astral-query tool or a declared tool crosses mod.mcp.call_agent_action, and the
auth module answers it.

Every operation is local-only. A query arriving over a link is rejected, and the
shell module — the sole mount point for every module's operations — rejects a
query carrying the mcp origin, so an agent reaches none of these operations on
its own host node.
*/
package mcp

const (
	MethodCreateAgent = "mcp.create_agent"
	MethodAgent       = "mcp.agent"
	MethodDeleteAgent = "mcp.delete_agent"
	MethodListAgents  = "mcp.list_agents"
)
