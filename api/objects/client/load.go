package objects

import (
	"github.com/astralp2p/astral-go/api/objects"
	"github.com/astralp2p/astral-go/astral"
	"github.com/astralp2p/astral-go/astral/channel"
	"github.com/astralp2p/astral-go/lib/query"
)

// Load loads and parses the object identified by objectID from repo, verifying
// its hash. repo may be "" for the node's default read repository.
func (client *Client) Load(ctx *astral.Context, objectID *astral.ObjectID, repo string) (astral.Object, error) {
	args := query.Args{"id": objectID}
	if repo != "" {
		args["repo"] = repo
	}

	ch, err := client.queryCh(ctx, objects.MethodLoad, args)
	if err != nil {
		return nil, err
	}
	defer ch.Close()

	var obj astral.Object
	// why: PassErrors must run before the catch-all Expect — Expect[astral.Object]
	// matches any decoded object, including error objects, since every registered
	// error type is itself an astral.Object.
	err = ch.Switch(channel.PassErrors, channel.Expect[astral.Object](&obj))
	return obj, err
}

// Load calls the operation on the default client.
func Load(ctx *astral.Context, objectID *astral.ObjectID, repo string) (astral.Object, error) {
	return Default().Load(ctx, objectID, repo)
}
