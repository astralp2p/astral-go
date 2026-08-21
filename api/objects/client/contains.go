package objects

import (
	"github.com/astralp2p/astral-go/api/objects"
	"github.com/astralp2p/astral-go/astral"
	"github.com/astralp2p/astral-go/astral/channel"
	"github.com/astralp2p/astral-go/lib/query"
)

// Contains reports whether repo holds the object identified by objectID.
func (client *Client) Contains(ctx *astral.Context, repo string, objectID *astral.ObjectID) (has bool, err error) {
	ch, err := client.queryCh(ctx, objects.MethodContains, query.Args{
		"repo": repo,
		"id":   objectID,
	})
	if err != nil {
		return false, err
	}
	defer ch.Close()

	var result *astral.Bool
	err = ch.Switch(channel.Expect(&result), channel.PassErrors)
	if result != nil {
		has = bool(*result)
	}
	return
}

// Contains calls the operation on the default client.
func Contains(ctx *astral.Context, repo string, objectID *astral.ObjectID) (bool, error) {
	return Default().Contains(ctx, repo, objectID)
}
