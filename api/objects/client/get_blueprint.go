package objects

import (
	"github.com/astralp2p/astral-go/api/objects"
	"github.com/astralp2p/astral-go/astral"
	"github.com/astralp2p/astral-go/astral/channel"
	"github.com/astralp2p/astral-go/lib/query"
)

// GetBlueprint returns the blueprint the target node holds for typeName: a
// runtime blueprint as registered, or the one derived from a type compiled
// into the node. Referenced types are not included. A primitive or unknown
// name fails with the node's error_message.
func (client *Client) GetBlueprint(ctx *astral.Context, typeName string) (bp *astral.Blueprint, err error) {
	ch, err := client.queryCh(ctx, objects.MethodGetBlueprint, query.Args{"type": typeName})
	if err != nil {
		return
	}
	defer ch.Close()

	err = ch.Switch(channel.Expect(&bp), channel.PassErrors, channel.FailOnEOF(), channel.WithContext(ctx))
	return
}

func GetBlueprint(ctx *astral.Context, typeName string) (*astral.Blueprint, error) {
	return Default().GetBlueprint(ctx, typeName)
}
