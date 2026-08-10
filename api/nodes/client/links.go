package nodes

import (
	"github.com/astralp2p/astral-go/api/nodes"
	"github.com/astralp2p/astral-go/astral"
	"github.com/astralp2p/astral-go/astral/channel"
)

// Links lists the node's currently active links, ordered by creation time.
//
// The answer is a snapshot. A caller that needs to know when a link appears
// reads it again: the op carries no follow mode, so link liveness is polled
// rather than subscribed to.
func (client *Client) Links(ctx *astral.Context) ([]*nodes.LinkInfo, error) {
	ch, err := client.queryCh(ctx, nodes.MethodLinks, nil)
	if err != nil {
		return nil, err
	}
	defer ch.Close()

	var links []*nodes.LinkInfo
	err = ch.Switch(
		channel.Collect(&links),
		channel.BreakOnEOS,
		func(msg *astral.ErrorMessage) error {
			return msg
		},
	)

	return links, err
}

// Links calls the operation on the default client.
func Links(ctx *astral.Context) ([]*nodes.LinkInfo, error) {
	return Default().Links(ctx)
}
