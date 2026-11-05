package nearby

import (
	"github.com/astralp2p/astral-go/api/nearby"
	"github.com/astralp2p/astral-go/astral"
	"github.com/astralp2p/astral-go/astral/channel"
)

// Scan makes the target node broadcast a mod.nearby.scan_message, which nearby
// nodes answer with their status. The node acks without sending when its mode
// is silent.
//
// The target rejects a caller that does not hold mod.auth.admin_network_action.
func (client *Client) Scan(ctx *astral.Context) error {
	ch, err := client.queryCh(ctx, nearby.MethodScan, nil)
	if err != nil {
		return err
	}
	defer ch.Close()

	return ch.Switch(channel.ExpectAck, channel.PassErrors)
}

// Scan calls the operation on the default client.
func Scan(ctx *astral.Context) error {
	return Default().Scan(ctx)
}
