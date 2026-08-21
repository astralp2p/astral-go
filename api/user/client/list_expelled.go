package user

import (
	"github.com/astralp2p/astral-go/api/user"
	"github.com/astralp2p/astral-go/astral"
	"github.com/astralp2p/astral-go/astral/channel"
)

// ListExpelled lists the signed bans issued by the target node's active swarm
// user. Readable by any caller, matching SwarmStatus / ListSiblings.
//
// The target rejects with code 2 when it holds no active contract.
func (client *Client) ListExpelled(ctx *astral.Context) ([]*user.SignedExpulsion, error) {
	ch, err := client.queryCh(ctx, user.OpListExpelled, nil)
	if err != nil {
		return nil, err
	}
	defer ch.Close()

	var expelled []*user.SignedExpulsion
	err = ch.Switch(
		channel.Collect(&expelled),
		channel.BreakOnEOS,
		channel.PassErrors,
	)

	return expelled, err
}

// ListExpelled calls the operation on the default client.
func ListExpelled(ctx *astral.Context) ([]*user.SignedExpulsion, error) {
	return Default().ListExpelled(ctx)
}
