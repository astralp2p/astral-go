package user

import (
	"github.com/astralp2p/astral-go/api/user"
	"github.com/astralp2p/astral-go/astral"
	"github.com/astralp2p/astral-go/astral/channel"
	"github.com/astralp2p/astral-go/lib/query"
)

// DeriveIdentity returns the user that words derive on the target node, with
// the derivation of Bootstrap, and claims nothing. An empty words asks the node
// to generate the words; the result then carries them, and no other reply ever
// does. An empty passphrase derives the seed without one. Mirrors the astrald
// op OpDeriveIdentity.
//
// The node rejects a query of any origin other than local.
func (client *Client) DeriveIdentity(ctx *astral.Context, words string, passphrase string) (result *user.DeriveIdentityResult, err error) {
	args := query.Args{}
	if words == "" {
		args["generate"] = true
	}
	if passphrase != "" {
		args["passphrase"] = passphrase
	}

	ch, err := client.queryCh(ctx, user.OpDeriveIdentity, args)
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

	err = ctxErr(ctx, ch.Switch(channel.Expect(&result), channel.PassErrors, channel.WithContext(ctx)))
	return
}
