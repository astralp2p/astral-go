package messaging

import (
	"github.com/astralp2p/astral-go/api/messaging"
	"github.com/astralp2p/astral-go/astral"
	"github.com/astralp2p/astral-go/astral/channel"
	"github.com/astralp2p/astral-go/lib/query"
)

// CreateIdentity mints a new participant: a fresh identity with a signed relay
// contract, a signed contract letting the node host its mailbox, an optional
// alias, and an access token that authenticates it. The returned credential
// carries that token, and it is the only messaging response that does;
// apphost.list_tokens lists the participant's tokens.
//
// An empty alias binds none; the node generates no alias, because an alias is
// node-global and a name the caller did not choose contends in a namespace it
// does not own. A zero duration leaves the token's lifetime to the node's
// configured default.
func (client *Client) CreateIdentity(ctx *astral.Context, alias string, duration astral.Duration) (*messaging.IdentityCredential, error) {
	// why the keys are lower-case: the node snake-cases and lower-cases op
	// argument names and binds by that name. A capitalised key reaches the wire
	// verbatim, matches nothing, and is dropped without complaint.
	args := query.Args{}
	if alias != "" {
		args["alias"] = alias
	}
	if duration != 0 {
		args["duration"] = duration
	}

	ch, err := client.queryCh(ctx, messaging.MethodCreateIdentity, args)
	if err != nil {
		return nil, err
	}
	defer ch.Close()

	var cred *messaging.IdentityCredential
	if err = ch.Switch(channel.Expect(&cred), channel.PassErrors); err != nil {
		return nil, err
	}
	if cred == nil {
		return nil, errNoAnswer
	}
	return cred, nil
}

// CreateIdentity calls the operation on the default client.
func CreateIdentity(ctx *astral.Context, alias string, duration astral.Duration) (*messaging.IdentityCredential, error) {
	return Default().CreateIdentity(ctx, alias, duration)
}
