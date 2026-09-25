package messaging

import (
	"github.com/astralp2p/astral-go/api/messaging"
	"github.com/astralp2p/astral-go/astral"
	"github.com/astralp2p/astral-go/astral/channel"
)

// ReadMessages reads whole messages from the caller's own mail, with their
// direct replies. Reading an inbox message's body stamps it read and owes its
// sender a receipt. The request is sent after the channel is established, not
// as query args.
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
		return nil, errNoAnswer
	}
	return res, nil
}

// ReadMessages calls the operation on the default client.
func ReadMessages(ctx *astral.Context, req *messaging.ReadMessagesRequest) (*messaging.ReadMessagesResult, error) {
	return Default().ReadMessages(ctx, req)
}
