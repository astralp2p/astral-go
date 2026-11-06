package sys

import (
	"fmt"

	"github.com/astralp2p/astral-go/api/apphost"
	"github.com/astralp2p/astral-go/api/sys"
	"github.com/astralp2p/astral-go/astral"
	"github.com/astralp2p/astral-go/astral/channel"
)

// RegisterManager appoints the node's manager on the default node.
func RegisterManager(ctx *astral.Context, req *sys.RegisterManagerRequest) (*apphost.AccessToken, error) {
	return Default().RegisterManager(ctx, req)
}

// RegisterManager provisions a fresh App identity as apphost.register does and
// appoints it the node's manager: the node stores a delegate for each policy
// req.Policies names, with the new identity as delegate identity, and records
// the appointment. Only a fresh node appoints a manager. The node's gate may
// deny the query before the op runs (the default gate denies it on an
// unclaimed node); a node the gate admits that is not fresh answers
// sys.ErrNotFresh. Either surfaces here as an error. The query carries no
// arguments; req travels on the channel as one object. A nil req is refused
// with sys.ErrProtocolError before anything is sent.
func (client *Client) RegisterManager(ctx *astral.Context, req *sys.RegisterManagerRequest) (*apphost.AccessToken, error) {
	if req == nil {
		return nil, fmt.Errorf("%w: nil register manager request", sys.ErrProtocolError)
	}

	ch, err := client.queryCh(ctx, sys.MethodRegisterManager, nil)
	if err != nil {
		return nil, err
	}
	defer ch.Close()

	if err = ch.Send(req); err != nil {
		return nil, err
	}

	return readAccessToken(ctx, ch)
}

// readAccessToken reads the node's answer: the manager's token, or the error
// object a failed step streams. An answer carrying neither is a protocol
// error. The read ends with ctx, and a token that arrived stands however ctx
// ends.
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
		return nil, fmt.Errorf("%w: sys.register_manager returned no access token", sys.ErrProtocolError)
	}

	return token, nil
}
