package messaging

import (
	"github.com/astralp2p/astral-go/api/messaging"
	"github.com/astralp2p/astral-go/astral"
	"github.com/astralp2p/astral-go/astral/channel"
)

// SendMessage sends one message from the calling participant and returns the
// id the node minted for it. The request is sent after the channel is
// established, not as query args.
func (client *Client) SendMessage(ctx *astral.Context, req *messaging.SendMessageRequest) (messaging.MessageID, error) {
	ch, err := client.queryCh(ctx, messaging.MethodSendMessage, nil)
	if err != nil {
		return messaging.MessageID{}, err
	}
	defer ch.Close()

	if err = ch.Send(req); err != nil {
		return messaging.MessageID{}, err
	}

	var id *messaging.MessageID
	if err = ch.Switch(channel.Expect(&id), channel.PassErrors); err != nil {
		return messaging.MessageID{}, err
	}
	if id == nil {
		return messaging.MessageID{}, ErrNoAnswer
	}
	return *id, nil
}

// SendMessage calls the operation on the default client.
func SendMessage(ctx *astral.Context, req *messaging.SendMessageRequest) (messaging.MessageID, error) {
	return Default().SendMessage(ctx, req)
}
