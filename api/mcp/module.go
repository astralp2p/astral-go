/*
Package mcp describes a module that registers AI agents on a node and serves
them their mail and the deployment's declared tools over the Model Context
Protocol.

An agent is a messaging participant: an identity the messaging module mints,
with its signed relay contract, its signed hosting contract, an optional alias
and an access token, which the agent presents to the node's MCP endpoint as a
bearer credential. The endpoint admits any valid apphost access token and reads
no agent record. Its mail lives in the messaging module and is described by
package messaging; this module keeps the agent's record and serves the
endpoint.

One node holds the agents of many tenants and knows no relation between them,
so it holds no reachability of its own. The endpoint serves an agent exactly
the five mail tools and the tools the node's configuration declares; no
built-in tool sends a query. A declared tool asks no authorization action: it
sends its query as the agent, carrying the mcp origin, and the target service
decides whether to answer its caller.

Every operation is local-only. A query arriving over a link is rejected, and the
shell module — the sole mount point for every module's operations — rejects a
query carrying the mcp origin, so an agent reaches none of these operations on
its own host node, not even through a declared tool.
*/
package mcp

const (
	MethodCreateAgent = "mcp.create_agent"
	MethodAgent       = "mcp.agent"
	MethodDeleteAgent = "mcp.delete_agent"
	MethodListAgents  = "mcp.list_agents"
)
