package mcp

import (
	"github.com/astralp2p/astral-go/api/mcp"
	"github.com/astralp2p/astral-go/astral"
	"github.com/astralp2p/astral-go/astral/channel"
	"github.com/astralp2p/astral-go/lib/query"
)

// DeleteAgent removes an agent: deletes the messaging participant under it —
// every access token and grant, its alias, the node's hosting of its mailbox
// and its mail — and then its record. A participant the node no longer hosts
// is not an error: the record is deleted all the same. The MCP endpoint checks
// the token on every request, so a later request presenting it is refused; a
// request already running is not cut short. The signed relay and hosting
// contracts stay valid until they expire. id takes a hex public key or an alias
// resolved via the directory.
//
// The node answers "unknown identity" when id resolves to no identity, and
// "agent not found" when it resolves but no agent is registered under it.
func (client *Client) DeleteAgent(ctx *astral.Context, id string) error {
	ch, err := client.queryCh(ctx, mcp.MethodDeleteAgent, query.Args{"identity": id})
	if err != nil {
		return err
	}
	defer ch.Close()

	return ch.Switch(channel.ExpectAck, channel.PassErrors)
}

// DeleteAgent calls the operation on the default client.
func DeleteAgent(ctx *astral.Context, id string) error {
	return Default().DeleteAgent(ctx, id)
}
