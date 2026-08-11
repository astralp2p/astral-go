package apphost

import (
	"github.com/astralp2p/astral-go/api/apphost"
	"github.com/astralp2p/astral-go/astral"
	"github.com/astralp2p/astral-go/astral/channel"
	"github.com/astralp2p/astral-go/lib/query"
)

// Cancel cancels the en-route query identified by id, optionally reporting
// cause to whatever is waiting on it.
func (client *Client) Cancel(ctx *astral.Context, id astral.Nonce, cause string) (err error) {
	args := query.Args{"id": id}
	if cause != "" {
		args["cause"] = cause
	}

	ch, err := client.queryCh(ctx, apphost.MethodCancel, args)
	if err != nil {
		return
	}
	defer ch.Close()

	return ch.Switch(channel.ExpectAck, channel.PassErrors)
}

// Cancel calls the operation on the default client.
func Cancel(ctx *astral.Context, id astral.Nonce, cause string) error {
	return Default().Cancel(ctx, id, cause)
}
