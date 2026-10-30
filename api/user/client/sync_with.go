package user

import (
	"github.com/astralp2p/astral-go/api/user"
	"github.com/astralp2p/astral-go/astral"
	"github.com/astralp2p/astral-go/astral/channel"
	"github.com/astralp2p/astral-go/lib/query"
)

// SyncWith asks the target node to run an outbound asset sync with node, and
// blocks until it completes. The target node resumes from the height it
// recorded after its last completed sync with node.
func (client *Client) SyncWith(ctx *astral.Context, node *astral.Identity) (err error) {
	ch, err := client.queryCh(ctx, user.OpSyncWith, query.Args{"identity": node})
	if err != nil {
		return
	}
	defer ch.Close()

	return ctxErr(ctx, ch.Switch(channel.ExpectAck, channel.PassErrors, channel.WithContext(ctx)))
}

// SyncWith calls the operation on the default client.
func SyncWith(ctx *astral.Context, node *astral.Identity) error {
	return Default().SyncWith(ctx, node)
}
