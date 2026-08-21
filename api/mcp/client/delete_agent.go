package mcp

import (
	"github.com/astralp2p/astral-go/api/mcp"
	"github.com/astralp2p/astral-go/astral"
	"github.com/astralp2p/astral-go/astral/channel"
	"github.com/astralp2p/astral-go/lib/query"
)

// DeleteAgent removes an agent: revokes its access token, unsets its alias and
// deletes its record. The agent's queued queries are dropped and its live
// sessions closed. The signed relay contract stays indexed until it expires.
// id takes a hex public key or an alias resolved via the directory.
func (client *Client) DeleteAgent(ctx *astral.Context, id string) error {
	ch, err := client.queryCh(ctx, mcp.MethodDeleteAgent, query.Args{"id": id})
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
