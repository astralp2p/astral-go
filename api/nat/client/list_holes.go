package nat

import (
	"github.com/astralp2p/astral-go/api/nat"
	"github.com/astralp2p/astral-go/astral"
	"github.com/astralp2p/astral-go/astral/channel"
	"github.com/astralp2p/astral-go/lib/query"
)

// ListHoles returns known NAT holes, optionally filtered to those involving the peer identity string with.
func (client *Client) ListHoles(ctx *astral.Context, with string) ([]*nat.Hole, error) {
	args := query.Args{}
	if with != "" {
		args["with"] = with
	}

	ch, err := client.queryCh(ctx, nat.MethodListHoles, args)
	if err != nil {
		return nil, err
	}
	defer ch.Close()

	var holes []*nat.Hole

	err = ch.Switch(
		channel.Collect(&holes),
		channel.BreakOnEOS,
		func(msg *astral.ErrorMessage) error {
			return msg
		},
	)

	return holes, err
}
