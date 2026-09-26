package messaging

import (
	"github.com/astralp2p/astral-go/api/messaging"
	"github.com/astralp2p/astral-go/astral"
	"github.com/astralp2p/astral-go/astral/channel"
	"github.com/astralp2p/astral-go/lib/query"
)

// Archive puts one of the caller's messages away, or back when undo is set. It
// reports whether this call moved the message; false means it was already
// where the call asked for, or the caller does not hold it.
func (client *Client) Archive(ctx *astral.Context, ref messaging.MessageRef, undo bool) (bool, error) {
	args := query.Args{"box": string(ref.Box), "id": ref.ID}
	if undo {
		args["undo"] = true
	}

	ch, err := client.queryCh(ctx, messaging.MethodArchive, args)
	if err != nil {
		return false, err
	}
	defer ch.Close()

	var res *messaging.ArchiveResult
	if err = ch.Switch(channel.Expect(&res), channel.PassErrors); err != nil {
		return false, err
	}
	if res == nil {
		return false, ErrNoAnswer
	}
	return bool(res.Changed), nil
}

// Archive calls the operation on the default client.
func Archive(ctx *astral.Context, ref messaging.MessageRef, undo bool) (bool, error) {
	return Default().Archive(ctx, ref, undo)
}
