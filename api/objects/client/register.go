package objects

import (
	"fmt"
	"strings"

	"github.com/astralp2p/astral-go/api/objects"
	"github.com/astralp2p/astral-go/astral"
	"github.com/astralp2p/astral-go/astral/channel"
)

// Register pushes a runtime *astral.Blueprint (struct or alias kind) to the remote node and
// returns its content-addressed ObjectID. The wait for the node's answer ends with ctx, and
// returns ctx's error.
func (client *Client) Register(ctx *astral.Context, o astral.Object) (id *astral.ObjectID, err error) {
	ch, err := client.queryCh(ctx, objects.MethodRegisterBlueprint, nil)
	if err != nil {
		return
	}
	defer ch.Close()

	if err = ch.Send(o); err != nil {
		return
	}

	err = ch.Switch(channel.Expect(&id), channel.PassErrors, channel.WithContext(ctx))

	// why: WithContext ends the read by closing the channel, which reads as an
	// empty answer or a transport error, not as the end of ctx.
	if id == nil && ctx.Err() != nil {
		return nil, ctx.Err()
	}
	if err != nil && strings.HasPrefix(err.Error(), astral.ErrAlreadyRegistered.Error()) {
		return nil, fmt.Errorf("%w: %s", astral.ErrAlreadyRegistered, typeNameOf(o))
	}
	return
}

func Register(ctx *astral.Context, o astral.Object) (*astral.ObjectID, error) {
	return Default().Register(ctx, o)
}

// typeNameOf returns the blueprint-side type name of a *astral.Blueprint for use in error
// messages. Falls back to ObjectType for any other input.
func typeNameOf(o astral.Object) string {
	if v, ok := o.(*astral.Blueprint); ok {
		return v.Type.String()
	}
	return o.ObjectType()
}
