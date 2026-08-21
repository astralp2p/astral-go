package user

import (
	"github.com/astralp2p/astral-go/api/user"
	"github.com/astralp2p/astral-go/astral"
	"github.com/astralp2p/astral-go/astral/channel"
	"github.com/astralp2p/astral-go/lib/query"
)

// SyncWith asks the target node to run an outbound asset sync with node,
// starting from height start, and blocks until it completes.
func (client *Client) SyncWith(ctx *astral.Context, node *astral.Identity, start uint64) (err error) {
	ch, err := client.queryCh(ctx, user.OpSyncWith, query.Args{"node": node, "start": start})
	if err != nil {
		return
	}
	defer ch.Close()

	return ch.Switch(channel.ExpectAck, channel.PassErrors)
}

// SyncWith calls the operation on the default client.
func SyncWith(ctx *astral.Context, node *astral.Identity, start uint64) error {
	return Default().SyncWith(ctx, node, start)
}
