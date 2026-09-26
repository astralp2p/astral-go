package messaging

import (
	"io"

	"github.com/astralp2p/astral-go/api/messaging"
	"github.com/astralp2p/astral-go/astral"
	"github.com/astralp2p/astral-go/astral/channel"
	"github.com/astralp2p/astral-go/lib/query"
)

// ListMessages streams one list of a mailbox as envelopes, without bodies. The
// mailbox is the caller's own unless req.Mailbox names another, which the node
// answers only to a caller holding messaging.ReadMailboxAction for it. Pass
// messaging.NextSince of the answer as the next Since to page the inbox.
//
// A listing the node refuses, on the caller's own mailbox or on another, is
// rejected before it is accepted and returns *astral.ErrRejected. A stream the
// node ends before its eos returns io.ErrUnexpectedEOF and no envelopes.
//
// why a stream without its eos is an error: the eos is what says the list is
// whole, and a stream cut short reads exactly like one that ended.
func (client *Client) ListMessages(ctx *astral.Context, req messaging.ListMessagesRequest) ([]*messaging.Envelope, error) {
	ch, err := client.queryCh(ctx, messaging.MethodListMessages, listArgs(req))
	if err != nil {
		return nil, err
	}
	defer ch.Close()

	var list []*messaging.Envelope
	var sawEOS bool
	err = ch.Switch(
		channel.Collect[*messaging.Envelope](&list),
		channel.PassErrors,
		channel.MarkEOS(&sawEOS),
	)
	switch {
	case err != nil:
		return nil, err
	case !sawEOS:
		return nil, io.ErrUnexpectedEOF
	}
	return list, nil
}

// ListMessages calls the operation on the default client.
func ListMessages(ctx *astral.Context, req messaging.ListMessagesRequest) ([]*messaging.Envelope, error) {
	return Default().ListMessages(ctx, req)
}

// listArgs carries only what the request sets: the node refuses a narrowing a
// list cannot apply, so an argument sent at its zero value could be refused.
func listArgs(req messaging.ListMessagesRequest) query.Args {
	args := query.Args{}
	if req.List != "" {
		args["list"] = req.List
	}
	if req.From != "" {
		args["from"] = req.From
	}
	if req.To != "" {
		args["to"] = req.To
	}
	if req.Since != 0 {
		args["since"] = req.Since
	}
	if req.UnreadOnly {
		args["unread_only"] = true
	}
	if req.AwaitingPickup {
		args["awaiting_pickup"] = true
	}
	if req.Mailbox != "" {
		args["mailbox"] = req.Mailbox
	}
	return args
}
