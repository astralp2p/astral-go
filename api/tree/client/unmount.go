package tree

import (
	"github.com/astralp2p/astral-go/api/tree"
	"github.com/astralp2p/astral-go/astral"
	"github.com/astralp2p/astral-go/lib/query"
)

// Unmount unmounts whatever is mounted at path on the target node's tree.
func (client *Client) Unmount(ctx *astral.Context, path string) error {
	ch, err := client.queryCh(ctx, tree.MethodUnmount, query.Args{"path": path})
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
