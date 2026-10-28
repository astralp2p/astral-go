package apphost

import (
	"fmt"

	"github.com/astralp2p/astral-go/api/apphost"
	"github.com/astralp2p/astral-go/astral"
	"github.com/astralp2p/astral-go/astral/channel"
)

// Whoami returns the identity the node authenticated this session as. With an
// app's token that is the app's identity, so an app holding a stored token
// learns that the node still accepts it, and for whom.
//
// The node checks the token when the session opens, before any query. A token
// it does not accept fails Whoami with the *apphost.ErrorMsg the session
// received, whose Code is apphost.ErrCodeAuthFailed. Under the node's default
// anonymous allowlist a session without a token is denied
// (apphost.ErrCodeDenied). A node that does not answer before ctx ends fails
// Whoami with ctx's error.
func (client *Client) Whoami(ctx *astral.Context) (*astral.Identity, error) {
	ch, err := client.queryCh(ctx, apphost.MethodWhoami, nil)
	if err != nil {
		return nil, err
	}
	defer ch.Close()

	var id *astral.Identity
	err = ch.Switch(channel.Expect(&id), channel.PassErrors, channel.WithContext(ctx))

	// why: WithContext ends the read by closing the channel, which reads as an
	// answer naming nobody or as a transport error, not as the end of ctx.
	if id == nil && ctx.Err() != nil {
		return nil, ctx.Err()
	}
	if err != nil {
		return nil, err
	}

	// why: an answer that names nobody would read as a session of the
	// anonymous identity, which the op never answers.
	if id.IsZero() {
		return nil, fmt.Errorf("%w: apphost.whoami returned no identity", apphost.ErrProtocolError)
	}

	return id, nil
}

// Whoami calls the operation on the default client.
func Whoami(ctx *astral.Context) (*astral.Identity, error) {
	return Default().Whoami(ctx)
}
