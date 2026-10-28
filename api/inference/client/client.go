package inference

import (
	"github.com/astralp2p/astral-go/astral"
	"github.com/astralp2p/astral-go/astral/channel"
	"github.com/astralp2p/astral-go/lib/astrald"
)

// Client sends inference queries to one target. Models and Complete address a
// provider; Discover, PushBlueprints and CheckBlueprints address a node. A nil
// target is the local node.
type Client struct {
	astral   *astrald.Client
	targetID *astral.Identity
}

var defaultClient *Client

func New(targetID *astral.Identity, client *astrald.Client) *Client {
	if client == nil {
		client = astrald.Default()
	}
	return &Client{astral: client, targetID: targetID}
}

// Default addresses the local node.
func Default() *Client {
	if defaultClient == nil {
		defaultClient = New(nil, astrald.Default())
	}
	return defaultClient
}

func (client *Client) queryCh(ctx *astral.Context, method string, args any) (*channel.Channel, error) {
	return client.astral.WithTarget(client.targetID).QueryChannel(ctx, method, args)
}

// ctxErr prefers the end of ctx over the read error it caused.
func ctxErr(ctx *astral.Context, err error) error {
	if ctx.Err() != nil {
		return ctx.Err()
	}
	return err
}
