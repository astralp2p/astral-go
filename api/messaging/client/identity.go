package messaging

import (
	"github.com/astralp2p/astral-go/api/messaging"
	"github.com/astralp2p/astral-go/astral"
	"github.com/astralp2p/astral-go/astral/channel"
	"github.com/astralp2p/astral-go/lib/query"
)

// Identity returns one participant without its credential. id takes a hex
// public key or an alias resolved via the directory.
//
// The node answers "unknown identity" when id resolves to no identity, and
// "identity not found" when it resolves but is not a participant.
func (client *Client) Identity(ctx *astral.Context, id string) (*messaging.IdentityInfo, error) {
	ch, err := client.queryCh(ctx, messaging.MethodIdentity, query.Args{"identity": id})
	if err != nil {
		return nil, err
	}
	defer ch.Close()

	var info *messaging.IdentityInfo
	if err = ch.Switch(channel.Expect(&info), channel.PassErrors); err != nil {
		return nil, err
	}
	if info == nil {
		return nil, errNoAnswer
	}
	return info, nil
}

// Identity calls the operation on the default client.
func Identity(ctx *astral.Context, id string) (*messaging.IdentityInfo, error) {
	return Default().Identity(ctx, id)
}
