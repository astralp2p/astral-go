package tree

import (
	"github.com/astralp2p/astral-go/api/tree"
	"github.com/astralp2p/astral-go/astral"
	"github.com/astralp2p/astral-go/lib/query"
)

// MountRemote mounts target's tree, rooted at target's root (or target's own
// root when root is ""), at path on the target node's tree.
func (client *Client) MountRemote(ctx *astral.Context, path string, target *astral.Identity, root string) error {
	args := query.Args{
		"path":   path,
		"target": target.String(),
	}
	if root != "" {
		args["root"] = root
	}

	ch, err := client.queryCh(ctx, tree.MethodMountRemote, args)
	if err != nil {
		return err
	}
	defer ch.Close()

	msg, err := ch.Receive()
	switch msg := msg.(type) {
	case *astral.Ack:
		return nil
	case nil:
		return err
	case *astral.ErrorMessage:
		return msg
	default:
		return astral.NewErrUnexpectedObject(msg)
	}
}
