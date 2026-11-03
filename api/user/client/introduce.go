package user

import (
	"github.com/astralp2p/astral-go/api/user"
	"github.com/astralp2p/astral-go/astral"
	"github.com/astralp2p/astral-go/astral/channel"
	"github.com/astralp2p/astral-go/lib/query"
)

// Introduce asks the target node to push its active node contract to node and
// then sync with node. node indexes the contract, so the target joins node's
// local swarm. The call blocks until node accepted the contract; the sync runs
// after the answer. Mirrors the astrald op OpIntroduce.
//
// The target refuses a query of network origin, refuses without an active
// contract with user.ErrNoActiveContract, refuses node naming the target or its
// user with user.ErrIntroduceSelf, and refuses a caller without
// user.AdminSwarmAction over node with auth.ErrNotPermitted.
func (client *Client) Introduce(ctx *astral.Context, node *astral.Identity) (err error) {
	ch, err := client.queryCh(ctx, user.OpIntroduce, query.Args{"identity": node})
	if err != nil {
		return
	}
	defer ch.Close()

	return ctxErr(ctx, ch.Switch(channel.ExpectAck, channel.PassErrors, channel.WithContext(ctx)))
}

// Introduce calls the operation on the default client.
func Introduce(ctx *astral.Context, node *astral.Identity) error {
	return Default().Introduce(ctx, node)
}
