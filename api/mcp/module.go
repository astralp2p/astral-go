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

An agent answers two queries of its own. MethodMessage carries a Message to the
agent's identity, and the agent's node stores it in that agent's inbox.
MethodReceipt carries a Receipt back to a sender, and the sender's node stamps
the message collected. Both are addressed to an agent rather than to a node, so
they are the queries here a caller reaches over a link.
*/
package mcp

const (
	MethodCreateAgent = "mcp.create_agent"
	MethodAgent       = "mcp.agent"
	MethodDeleteAgent = "mcp.delete_agent"
	MethodListAgents  = "mcp.list_agents"
)

// MethodMessage is the query that delivers a Message, addressed to the
// recipient agent's identity. It is not an operation: no node serves it, and
// the agent's own node answers it on the agent's behalf.
const MethodMessage = "mcp.message"

// MethodReceipt is the query that carries a Receipt, addressed to the original
// sender agent's identity. Like MethodMessage it is not an operation: no node
// serves it, and the sender's own node answers it on the sender's behalf.
//
// why neither is an operation: an operation is addressed to a node's identity,
// and both of these are addressed to an agent's. A node mounts its modules'
// operations behind that check, so an operation carrying a receipt would be
// unreachable by the only caller that ever sends one.
//
// why it is the reverse of MethodMessage: the recipient calls and the sender is
// the target, so the pair of identities on the route is the same pair the
// delivery carried, exchanged.
const MethodReceipt = "mcp.receipt"
