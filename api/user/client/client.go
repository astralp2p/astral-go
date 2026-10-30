package user

import (
	"github.com/astralp2p/astral-go/astral"
	"github.com/astralp2p/astral-go/astral/channel"
	"github.com/astralp2p/astral-go/lib/astrald"
)

// Client sends user queries to one target. Every call stops waiting for the
// target's answer when its ctx ends, and returns ctx's error.
type Client struct {
	targetID *astral.Identity
	astral   *astrald.Client
}

var defaultClient *Client

func New(targetID *astral.Identity, client *astrald.Client) *Client {
	if client == nil {
		client = astrald.Default()
	}

	return &Client{
		astral:   client,
		targetID: targetID,
	}
}

func Default() *Client {
	if defaultClient == nil {
		defaultClient = New(nil, astrald.Default())
	}

	return defaultClient
}

func (client *Client) queryCh(ctx *astral.Context, method string, args any) (*channel.Channel, error) {
	return client.astral.WithTarget(client.targetID).QueryChannel(ctx, method, args)
}

// ctxErr prefers the end of ctx over what the read that ctx cut short
// returned.
//
// why: WithContext ends a read by closing the channel, and Switch reads that
// as a clean end, which leaves an answer empty or a list short with no error,
// or as a transport error. A caller could not tell its own cancel or timeout
// from a node that answered wrong. An ack or a list's eos ends a Switch the
// way a closed channel does, so a read that ctx ends right after it completed
// reports the end of ctx too.
func ctxErr(ctx *astral.Context, err error) error {
	if ctx.Err() != nil {
		return ctx.Err()
	}
	return err
}
