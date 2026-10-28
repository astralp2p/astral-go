package inference

import (
	"github.com/astralp2p/astral-go/api/inference"
	"github.com/astralp2p/astral-go/astral"
	"github.com/astralp2p/astral-go/astral/channel"
)

// Models returns the target provider's inventory for the caller. A provider
// that answers inference.error fails with that *inference.Error.
func (client *Client) Models(ctx *astral.Context) (models *inference.Models, err error) {
	ch, err := client.queryCh(ctx, inference.OpModels, nil)
	if err != nil {
		return nil, err
	}
	defer ch.Close()

	err = ch.Switch(channel.Expect(&models), channel.PassErrors, channel.FailOnEOF(), channel.WithContext(ctx))
	if err != nil {
		return nil, ctxErr(ctx, err)
	}
	return models, nil
}
