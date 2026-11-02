package apphost

import (
	"github.com/astralp2p/astral-go/api/apphost"
	"github.com/astralp2p/astral-go/astral"
	"github.com/astralp2p/astral-go/astral/channel"
	"github.com/astralp2p/astral-go/lib/query"
)

// ListGrants returns the node-local grants identity holds on the target node.
// Each grant's ID names it to RevokeGrant. A listing that ends before the
// node's end-of-stream marker is an error, not a complete list. The read ends
// with ctx, and returns ctx's error then.
func (client *Client) ListGrants(ctx *astral.Context, identity *astral.Identity) (grants []*apphost.Grant, err error) {
	ch, err := client.queryCh(ctx, apphost.MethodListGrants, query.Args{"identity": identity.String()})
	if err != nil {
		return nil, err
	}
	defer ch.Close()

	err = ch.Switch(
		channel.Collect[*apphost.Grant](&grants),
		channel.PassErrors,
		channel.BreakOnEOS,
		channel.FailOnEOF(),
		channel.WithContext(ctx),
	)

	// why: WithContext ends the read by closing the channel, which reads as a
	// listing cut short, not as the end of ctx.
	if err != nil && ctx.Err() != nil {
		return nil, ctx.Err()
	}
	if err != nil {
		return nil, err
	}
	return grants, nil
}

// ListGrants calls the operation on the default client.
func ListGrants(ctx *astral.Context, identity *astral.Identity) ([]*apphost.Grant, error) {
	return Default().ListGrants(ctx, identity)
}
