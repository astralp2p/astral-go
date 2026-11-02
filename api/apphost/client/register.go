package apphost

import (
	"fmt"

	"github.com/astralp2p/astral-go/api/apphost"
	"github.com/astralp2p/astral-go/api/auth"
	"github.com/astralp2p/astral-go/astral"
	"github.com/astralp2p/astral-go/astral/channel"
)

// Register provisions a fresh guest identity on the default node.
func Register(ctx *astral.Context, permits ...*auth.Permit) (*apphost.AccessToken, error) {
	return Default().Register(ctx, permits...)
}

// RegisterWith provisions a fresh guest identity on the default node.
func RegisterWith(ctx *astral.Context, req *apphost.RegisterRequest) (*apphost.AccessToken, error) {
	return Default().RegisterWith(ctx, req)
}

// Register provisions a fresh guest identity end to end: the node generates a
// keypair, signs an app contract between the new identity and itself, and
// mints an access token for it. This is how an app bootstraps itself on first
// run, so the session issuing the query is normally anonymous — the credential
// being asked for is the one the caller does not have yet.
//
// permits are the permits the new identity asks to hold as node-local grants,
// each a full auth.Permit with its constraints. A grant authorizes on this
// node alone, and grants add up. Asking is not receiving: the node's register
// policy decides what it writes, and the answer carries the token alone, so a
// caller learns what it holds by using it.
//
// Register asks for grants alone, so nothing it names becomes portable.
// RegisterWith also names the app and asks for contract permits.
//
// The node accepts the query before its register policy decides, so a refusal
// arrives as the error object apphost.ErrRegistrationDeclined, which surfaces
// here as an error, as does any failure past the accept gate. A delegated
// policy may wait on a person. The wait ends with ctx: Register closes the
// query, which tells the policy that nobody waits for its answer, and returns
// ctx's error.
func (client *Client) Register(ctx *astral.Context, permits ...*auth.Permit) (*apphost.AccessToken, error) {
	return client.RegisterWith(ctx, &apphost.RegisterRequest{GrantPermits: permits})
}

// RegisterWith is Register with the whole registration: the app's name and
// both permit lists, each permit with its constraints. The node's register
// policy decides what it writes. The query carries no arguments; req travels
// on the channel as one object, even when it is empty. A nil req is refused
// with apphost.ErrProtocolError before anything is sent.
func (client *Client) RegisterWith(ctx *astral.Context, req *apphost.RegisterRequest) (*apphost.AccessToken, error) {
	if req == nil {
		return nil, fmt.Errorf("%w: nil register request", apphost.ErrProtocolError)
	}

	ch, err := client.queryCh(ctx, apphost.MethodRegister, nil)
	if err != nil {
		return nil, err
	}
	defer ch.Close()

	if err = ch.Send(req); err != nil {
		return nil, err
	}

	return readAccessToken(ctx, ch)
}

// readAccessToken reads the node's answer: the minted token, or the error
// object a failed step streams. An answer carrying neither is a protocol
// error — registration has no empty success, and a nil token returned with a
// nil error would strand the caller at the one call it cannot retry blind.
// The read ends with ctx, and a token that arrived stands however ctx ends.
func readAccessToken(ctx *astral.Context, ch *channel.Channel) (token *apphost.AccessToken, err error) {
	err = ch.Switch(channel.Expect(&token), channel.PassErrors, channel.WithContext(ctx))

	// why: WithContext ends the read by closing the channel, which reads as an
	// answer without a token or as a transport error, not as the end of ctx.
	if token == nil && ctx.Err() != nil {
		return nil, ctx.Err()
	}
	if err != nil {
		return nil, err
	}

	if token == nil {
		return nil, fmt.Errorf("%w: apphost.register returned no access token", apphost.ErrProtocolError)
	}

	return token, nil
}
