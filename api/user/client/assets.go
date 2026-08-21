package user

import (
	"github.com/astralp2p/astral-go/api/user"
	"github.com/astralp2p/astral-go/astral"
	"github.com/astralp2p/astral-go/astral/channel"
)

// Assets lists the object IDs the target node currently holds as user assets.
func (client *Client) Assets(ctx *astral.Context) ([]*astral.ObjectID, error) {
	ch, err := client.queryCh(ctx, user.OpAssets, nil)
	if err != nil {
		return nil, err
	}
	defer ch.Close()

	var assets []*astral.ObjectID
	err = ch.Switch(
		channel.Collect(&assets),
		channel.BreakOnEOS,
		channel.PassErrors,
	)

	return assets, err
}

// Assets calls the operation on the default client.
func Assets(ctx *astral.Context) ([]*astral.ObjectID, error) {
	return Default().Assets(ctx)
}
