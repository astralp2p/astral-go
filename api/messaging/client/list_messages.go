package messaging

import (
	"github.com/astralp2p/astral-go/api/messaging"
	"github.com/astralp2p/astral-go/astral"
	"github.com/astralp2p/astral-go/astral/channel"
	"github.com/astralp2p/astral-go/lib/query"
)

// ListMessages streams one of the caller's lists as envelopes, without bodies.
// Pass messaging.NextSince of the answer as the next Since to page the inbox.
func (client *Client) ListMessages(ctx *astral.Context, req messaging.ListMessagesRequest) (list []*messaging.Envelope, err error) {
	ch, err := client.queryCh(ctx, messaging.MethodListMessages, listArgs(req))
	if err != nil {
		return
	}
	defer ch.Close()

	err = ch.Switch(
		channel.Collect[*messaging.Envelope](&list),
		channel.PassErrors,
		channel.BreakOnEOS,
	)
	return
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
	return args
}
