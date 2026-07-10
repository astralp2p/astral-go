package objects

import (
	"github.com/astralp2p/astral-go/api/objects"
	"github.com/astralp2p/astral-go/astral"
	"github.com/astralp2p/astral-go/astral/channel"
)

// Blueprints returns every type name registered on the target node in
// dependency order (struct blueprints + alias blueprints), terminated
// server-side by astral.EOS.
func (client *Client) Blueprints(ctx *astral.Context) (names []string, err error) {
	ch, err := client.queryCh(ctx, objects.MethodBlueprints, nil)
	if err != nil {
		return
	}
	defer ch.Close()

	err = ch.Switch(
		channel.CollectStrings[*astral.String8](&names),
		channel.BreakOnEOS,
		channel.PassErrors,
		channel.WithContext(ctx),
	)
	return
}

func Blueprints(ctx *astral.Context) ([]string, error) {
	return Default().Blueprints(ctx)
}
