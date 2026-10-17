package nearby

import (
	"github.com/astralp2p/astral-go/api/nearby"
	"github.com/astralp2p/astral-go/astral"
	"github.com/astralp2p/astral-go/astral/channel"
)

// Broadcast makes the target node push its status to nearby nodes at once.
// The node acks without pushing when its mode is silent, or when it is
// stealth and the status carries no attachment.
//
// The target rejects a caller that does not hold mod.auth.admin_network_action.
func (client *Client) Broadcast(ctx *astral.Context) error {
	ch, err := client.queryCh(ctx, nearby.MethodBroadcast, nil)
	if err != nil {
		return err
	}
	defer ch.Close()

	return ch.Switch(channel.ExpectAck, channel.PassErrors)
}

// Broadcast calls the operation on the default client.
func Broadcast(ctx *astral.Context) error {
	return Default().Broadcast(ctx)
}
