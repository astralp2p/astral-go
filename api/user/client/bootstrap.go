package user

import (
	"github.com/astralp2p/astral-go/api/user"
	"github.com/astralp2p/astral-go/astral"
	"github.com/astralp2p/astral-go/astral/channel"
	"github.com/astralp2p/astral-go/lib/query"
)

// Bootstrap claims the target node for the user that words derive, and returns
// that user. An empty words asks the node to generate the words; the result then
// carries them, and no other reply ever does. An empty passphrase derives the
// seed without one. Mirrors the astrald op OpBootstrap.
//
// The node rejects with code 2 when it already has an active contract, with
// code 4 when the caller is not the node itself, and rejects a query of any
// origin other than local.
func (client *Client) Bootstrap(ctx *astral.Context, words string, passphrase string) (result *user.BootstrapResult, err error) {
	args := query.Args{}
	if words == "" {
		args["generate"] = true
	}
	if passphrase != "" {
		args["passphrase"] = passphrase
	}

	ch, err := client.queryCh(ctx, user.OpBootstrap, args)
	if err != nil {
		return
	}
	defer ch.Close()

	// why: a query string is logged, the stream is not
	if words != "" {
		if err = ch.Send(astral.NewString16(words)); err != nil {
			return
		}
	}

	err = ch.Switch(channel.Expect(&result), channel.PassErrors)
	return
}
