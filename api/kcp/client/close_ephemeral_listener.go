package kcp

import (
	"github.com/astralp2p/astral-go/api/kcp"
	"github.com/astralp2p/astral-go/astral"
	"github.com/astralp2p/astral-go/astral/channel"
	"github.com/astralp2p/astral-go/lib/query"
)

func (client *Client) CloseEphemeralListener(ctx *astral.Context, port astral.Uint16) error {
	ch, err := client.queryCh(ctx, kcp.MethodCloseEphemeralListener, query.Args{
		"port": port,
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
