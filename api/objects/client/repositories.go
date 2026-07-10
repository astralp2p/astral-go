package objects

import (
	"github.com/astralp2p/astral-go/api/objects"
	"github.com/astralp2p/astral-go/astral"
	"github.com/astralp2p/astral-go/astral/channel"
)

func (client *Client) Repositories(ctx *astral.Context) (repos []*objects.RepositoryInfo, err error) {
	ch, err := client.queryCh(ctx, objects.MethodRepositories, nil)
	if err != nil {
		return nil, err
	}
	defer ch.Close()

	// collect repo names
	err = ch.Switch(channel.Collect(&repos), channel.BreakOnEOS, channel.PassErrors, channel.WithContext(ctx))
	return
}
