package apphost

import (
	"github.com/astralp2p/astral-go/api/apphost"
	"github.com/astralp2p/astral-go/astral"
	"github.com/astralp2p/astral-go/astral/channel"
	"github.com/astralp2p/astral-go/lib/query"
)

// RegisterHandler registers a new handler for incoming queries.
func (client *Client) RegisterHandler(ctx *astral.Context, endpoint string, authToken astral.Nonce) error {
	ch, err := client.queryCh(ctx, apphost.MethodRegisterHandler, query.Args{
		"endpoint": endpoint,
		"token":    authToken,
	})
	if err != nil {
		return err
	}
	defer ch.Close()

	return ch.Switch(channel.ExpectAck, channel.PassErrors)
}

func RegisterHandler(ctx *astral.Context, endpoint string, authToken astral.Nonce) error {
	return Default().RegisterHandler(ctx, endpoint, authToken)
}
