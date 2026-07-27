package apphost

import (
	"github.com/astralp2p/astral-go/api/apphost"
	"github.com/astralp2p/astral-go/astral"
	"github.com/astralp2p/astral-go/astral/channel"
	"github.com/astralp2p/astral-go/lib/query"
)

// DeleteToken removes an access token so it no longer authenticates.
func (client *Client) DeleteToken(ctx *astral.Context, token string) error {
	ch, err := client.queryCh(ctx, apphost.MethodDeleteToken, query.Args{
		"token": token,
	})
	if err != nil {
		return err
	}
	defer ch.Close()

	return ch.Switch(channel.ExpectAck, channel.PassErrors, channel.WithContext(ctx))
}

func DeleteToken(ctx *astral.Context, token string) error {
	return Default().DeleteToken(ctx, token)
}
