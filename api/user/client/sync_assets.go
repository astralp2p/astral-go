package user

import (
	"github.com/astralp2p/astral-go/api/user"
	"github.com/astralp2p/astral-go/astral"
	"github.com/astralp2p/astral-go/astral/channel"
	"github.com/astralp2p/astral-go/lib/query"
)

// SyncAssets streams the target node's asset delta log from start, and returns
// the height to resume from on the next call. When nothing is new at or above
// start, nextHeight echoes start back so the caller can safely re-poll.
func (client *Client) SyncAssets(ctx *astral.Context, start uint64) (updates []*user.OpUpdate, nextHeight uint64, err error) {
	ch, err := client.queryCh(ctx, user.OpSyncAssets, query.Args{"start": start})
	if err != nil {
		return
	}
	defer ch.Close()

	var height *astral.Uint64
	err = ch.Switch(
		channel.Collect(&updates),
		channel.Expect(&height),
		channel.PassErrors,
	)
	if height != nil {
		nextHeight = uint64(*height)
	}
	return
}

// SyncAssets calls the operation on the default client.
func SyncAssets(ctx *astral.Context, start uint64) ([]*user.OpUpdate, uint64, error) {
	return Default().SyncAssets(ctx, start)
}
