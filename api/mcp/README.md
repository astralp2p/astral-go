# mcp

Wire types and op-name constants for the `mcp` protocol — AI agent registration
on a node, and the MCP endpoint that serves those agents the astral network;
`client/` is the protocol's RPC client.

Every operation is local-only, so a client reaching a node over the network is
refused whatever identity it holds.

Protocol spec:

* [astral-docs/protocols/mcp](https://github.com/astralp2p/astral-docs/tree/master/protocols/mcp) — overview and op specs
