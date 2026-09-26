# mcp

Wire types and op-name constants for the `mcp` protocol — AI agent registration
on a node, and the MCP endpoint that serves those agents their mail and the
deployment's declared tools; `client/` is the protocol's RPC client.

An agent is a messaging participant. Its mail lives in
[`api/messaging`](../messaging).

The MCP endpoint serves an agent exactly the five mail tools and the tools the
node's configuration declares. A declared tool asks no authorization action: it
sends its query as the agent, carrying the MCP origin, and the target service
decides whether to answer its caller.

Every operation is local-only, so a client reaching a node over the network is
refused whatever identity it holds.

Protocol spec:

* [astral-docs/protocols/mcp](https://github.com/astralp2p/astral-docs/tree/master/protocols/mcp) — overview and op specs
