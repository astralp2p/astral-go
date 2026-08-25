package mcp

import (
	"github.com/astralp2p/astral-go/api/mcp"
	"github.com/astralp2p/astral-go/astral"
	"github.com/astralp2p/astral-go/astral/channel"
	"github.com/astralp2p/astral-go/lib/query"
)

// DisconnectAgent ends an agent's live traffic: it closes the conversations the
// agent is in and drops the queries waiting for it. id takes a hex public key or
// an alias resolved via the directory.
//
// It carries no permission and writes none. What an agent permits is held by
// whoever owns it and is answered on the next call the node asks about; the
// traffic already flowing is the one part of that change its owner cannot make
// elsewhere.
//
// The agent's parked astral-listen is left alone: a listener is the agent
// waiting to be called, not a caller it is talking to.
func (client *Client) DisconnectAgent(ctx *astral.Context, id string) error {
	ch, err := client.queryCh(ctx, mcp.MethodDisconnectAgent, query.Args{"id": id})
	if err != nil {
		return err
	}
	defer ch.Close()

	return ch.Switch(channel.ExpectAck, channel.PassErrors)
}

// DisconnectAgent calls the operation on the default client.
func DisconnectAgent(ctx *astral.Context, id string) error {
	return Default().DisconnectAgent(ctx, id)
}
