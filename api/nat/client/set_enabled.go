package nat

import (
	"github.com/astralp2p/astral-go/api/nat"
	"github.com/astralp2p/astral-go/astral"
	"github.com/astralp2p/astral-go/astral/channel"
	"github.com/astralp2p/astral-go/lib/query"
)

func (client *Client) SetEnabled(ctx *astral.Context, enabled bool) error {
	ch, err := client.queryCh(ctx, nat.MethodSetEnabled, query.Args{
		"arg": enabled,
	})
	if err != nil {
		return err
	}
	defer ch.Close()

	return ch.Switch(
		channel.ExpectAck,
		func(msg *astral.ErrorMessage) error {
			return msg
		},
	)
}
