package user

import (
	"github.com/astralp2p/astral-go/api/user"
	"github.com/astralp2p/astral-go/astral"
	"github.com/astralp2p/astral-go/astral/channel"
)

// SwarmStatus lists every node in the user's swarm, each with its alias and
// whether a link to it stands. A node belongs to the swarm when it holds an
// active swarm-membership contract issued by the same user as the target node.
//
// The target node is rejected with code 2 when it holds no active contract:
// swarm membership is what the answer is derived from, so a node belonging to
// nobody has nothing to report.
func (client *Client) SwarmStatus(ctx *astral.Context) ([]*user.SwarmMember, error) {
	ch, err := client.queryCh(ctx, user.OpSwarmStatus, nil)
	if err != nil {
		return nil, err
	}
	defer ch.Close()

	var members []*user.SwarmMember
	err = ch.Switch(
		channel.Collect(&members),
		channel.BreakOnEOS,
		func(msg *astral.ErrorMessage) error {
			return msg
		},
	)

	return members, err
}

// SwarmStatus calls the operation on the default client.
func SwarmStatus(ctx *astral.Context) ([]*user.SwarmMember, error) {
	return Default().SwarmStatus(ctx)
}
