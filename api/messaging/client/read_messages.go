package messaging

import (
	"github.com/astralp2p/astral-go/api/messaging"
	"github.com/astralp2p/astral-go/astral"
	"github.com/astralp2p/astral-go/astral/channel"
)

// ReadMessages reads whole messages from one mailbox, with their direct
// replies. The request is sent after the channel is established, not as query
// args.
//
// With req.Mailbox nil, or naming the caller, the mailbox is the caller's own.
// Reading an inbox message there stamps it read and tells its sender the body
// was collected — directly when the node hosts the sender's mailbox, by one
// receipt otherwise — even when the answer leaves the body out for room.
//
// With req.Mailbox naming another identity the read is delegated: the node
// answers it only to a caller holding messaging.ReadMailboxAction for that
// mailbox. It stamps nothing read and tells no sender a body was collected.
//
// The node reads the request after it accepts the query, so a refusal
// arrives in one of three forms. A refused delegated read returns ErrNoAnswer.
// A read of the caller's own mailbox, where the node does not host it, returns
// the node's error object, "not a messaging participant". A caller the node
// refuses before it reads the request — over a link, from an agent over MCP,
// the zero identity, or the node itself — is rejected and returns
// *astral.ErrRejected.
func (client *Client) ReadMessages(ctx *astral.Context, req *messaging.ReadMessagesRequest) (*messaging.ReadMessagesResult, error) {
	ch, err := client.queryCh(ctx, messaging.MethodReadMessages, nil)
	if err != nil {
		return nil, err
	}
	defer ch.Close()

	if err = ch.Send(req); err != nil {
		return nil, err
	}

	var res *messaging.ReadMessagesResult
	if err = ch.Switch(channel.Expect(&res), channel.PassErrors); err != nil {
		return nil, err
	}
	if res == nil {
		return nil, ErrNoAnswer
	}
	return res, nil
}

// ReadMessages calls the operation on the default client.
func ReadMessages(ctx *astral.Context, req *messaging.ReadMessagesRequest) (*messaging.ReadMessagesResult, error) {
	return Default().ReadMessages(ctx, req)
}
