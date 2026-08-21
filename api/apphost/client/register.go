package apphost

import (
	"fmt"
	"strings"

	"github.com/astralp2p/astral-go/api/apphost"
	"github.com/astralp2p/astral-go/astral"
	"github.com/astralp2p/astral-go/astral/channel"
	"github.com/astralp2p/astral-go/lib/query"
)

// Register provisions a fresh guest identity on the default node.
func Register(ctx *astral.Context, permits ...string) (*apphost.AccessToken, error) {
	return Default().Register(ctx, permits...)
}

// Register provisions a fresh guest identity end to end: the node generates a
// keypair, signs an app contract between the new identity and itself, and
// mints an access token for it. This is how an app bootstraps itself on first
// run, so the session issuing the query is normally anonymous — the credential
// being asked for is the one the caller does not have yet.
//
// permits names the actions the new identity asks to hold, e.g.
// `mod.user.info_action`. Asking is not receiving: the node's register policy
// decides what it grants, and the answer carries the token alone, so a caller
// learns what it holds by using it. Asking for nothing sends no permits
// argument.
//
// The node rejects the query outright when its register policy refuses; a
// failure past the accept gate arrives instead as an error object, which
// surfaces here as an error. Both mean refused.
func (client *Client) Register(ctx *astral.Context, permits ...string) (*apphost.AccessToken, error) {
	args, err := registerArgs(permits)
	if err != nil {
		return nil, err
	}

	ch, err := client.queryCh(ctx, apphost.MethodRegister, args)
	if err != nil {
		return nil, err
	}
	defer ch.Close()

	return readAccessToken(ch)
}

// registerArgs joins the permits into the op's single argument. A permit
// carrying the separator is refused rather than sent: on the wire it would
// split into two, so the node would read an action name the caller never
// asked for.
func registerArgs(permits []string) (query.Args, error) {
	if len(permits) == 0 {
		return query.Args{}, nil
	}

	for _, permit := range permits {
		if strings.Contains(permit, ",") {
			return nil, fmt.Errorf("%w: permit name contains a comma: %q", apphost.ErrProtocolError, permit)
		}
	}

	return query.Args{"permits": strings.Join(permits, ",")}, nil
}

// readAccessToken reads the node's answer: the minted token, or the error
// object a failed step streams. An answer carrying neither is a protocol
// error — registration has no empty success, and a nil token returned with a
// nil error would strand the caller at the one call it cannot retry blind.
func readAccessToken(ch *channel.Channel) (token *apphost.AccessToken, err error) {
	if err = ch.Switch(channel.Expect(&token), channel.PassErrors); err != nil {
		return nil, err
	}

	if token == nil {
		return nil, fmt.Errorf("%w: apphost.register returned no access token", apphost.ErrProtocolError)
	}

	return token, nil
}
