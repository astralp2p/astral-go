package apphost

import (
	"github.com/astralp2p/astral-go/api/apphost"
	"github.com/astralp2p/astral-go/astral"
	"github.com/astralp2p/astral-go/astral/channel"
	"github.com/astralp2p/astral-go/lib/query"
)

// CreateToken mints an access token authenticating the given identity, valid
// for the given duration. The returned AccessToken carries the secret and the
// moment it stops being accepted.
//
// A zero duration leaves the token's lifetime to the node's configured
// default, as it does for CreateAgent.
func (client *Client) CreateToken(ctx *astral.Context, identity *astral.Identity, duration astral.Duration) (token *apphost.AccessToken, err error) {
	// why the keys are lower-case: the node snake-cases and lower-cases op
	// argument names and binds by that name. A capitalised key reaches the wire
	// verbatim, matches nothing, and is dropped without complaint.
	args := query.Args{"id": identity.String()}
	if duration != 0 {
		args["duration"] = duration
	}

	ch, err := client.queryCh(ctx, apphost.MethodCreateToken, args)
	if err != nil {
		return
	}
	defer ch.Close()

	err = ch.Switch(channel.Expect(&token), channel.PassErrors)
	return
}

// CreateToken calls the operation on the default client.
func CreateToken(ctx *astral.Context, identity *astral.Identity, duration astral.Duration) (*apphost.AccessToken, error) {
	return Default().CreateToken(ctx, identity, duration)
}
