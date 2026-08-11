package objects

import (
	"fmt"

	"github.com/astralp2p/astral-go/api/objects"
	"github.com/astralp2p/astral-go/astral"
	"github.com/astralp2p/astral-go/astral/channel"
	"github.com/astralp2p/astral-go/lib/query"
)

// New asks the node to construct a zero-value object of the given registered
// type and returns it. Returns an error when typ is not a registered type.
//
// No package-level shorthand: New collides with the client constructor of the
// same name. Call Default().New(ctx, typ) instead.
func (client *Client) New(ctx *astral.Context, typ string) (astral.Object, error) {
	ch, err := client.queryCh(ctx, objects.MethodNew, query.Args{"type": typ})
	if err != nil {
		return nil, err
	}
	defer ch.Close()

	var obj astral.Object
	// why: PassErrors must run before the catch-all Expect — Expect[astral.Object]
	// matches any decoded object, including error objects, since every registered
	// error type is itself an astral.Object.
	err = ch.Switch(channel.PassErrors, channel.Expect[astral.Object](&obj))
	if err != nil {
		return nil, err
	}
	if _, isNil := obj.(*astral.Nil); isNil {
		return nil, fmt.Errorf("objects: unknown type %q", typ)
	}

	return obj, nil
}
