package objects

import (
	"github.com/astralp2p/astral-go/api/objects"
	"github.com/astralp2p/astral-go/astral"
	"github.com/astralp2p/astral-go/astral/channel"
	"github.com/astralp2p/astral-go/lib/query"
)

// RegisterFinder registers the caller as a finder provider and blocks until the
// node answers with the lease it granted.
//
// The registration lasts only for that lease. Calling this again before it ends
// renews the caller's registration in place rather than adding a second one;
// letting it end removes the registration silently, with no notice from the
// node. The node clamps the requested duration to its own maximum, so the
// granted lease may be shorter than the one asked for.
//
// A zero duration leaves the lease to the node's configured default, as it does
// for apphost CreateToken.
func (client *Client) RegisterFinder(ctx *astral.Context, duration astral.Duration) (lease *objects.RegistrationLease, err error) {
	// why the key is lower-case: the node snake-cases and lower-cases op
	// argument names and binds by that name. A capitalised key reaches the wire
	// verbatim, matches nothing, and is dropped without complaint.
	var args query.Args
	if duration != 0 {
		args = query.Args{"duration": duration}
	}

	ch, err := client.queryCh(ctx, objects.MethodRegisterFinder, args)
	if err != nil {
		return
	}
	defer ch.Close()

	err = ch.Switch(channel.Expect(&lease), channel.PassErrors, channel.WithContext(ctx))
	return
}

func RegisterFinder(ctx *astral.Context, duration astral.Duration) (*objects.RegistrationLease, error) {
	return Default().RegisterFinder(ctx, duration)
}
