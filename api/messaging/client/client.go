package messaging

import (
	"errors"

	"github.com/astralp2p/astral-go/astral"
	"github.com/astralp2p/astral-go/astral/channel"
	"github.com/astralp2p/astral-go/lib/astrald"
)

// Client sends messaging RPC requests to a node.
//
// Every operation is local-only: the node rejects these queries when they
// arrive over a link, and rejects them again when they carry the mcp origin an
// agent's own queries carry. A client reaching a node over the network is
// refused, whatever identity it holds. The mail operations act on the caller's
// own boxes, so the node must host the caller's mailbox.
type Client struct {
	astral   *astrald.Client
	targetID *astral.Identity
}

var defaultClient *Client

// errNoAnswer is a query the node closed without the object the operation
// answers.
var errNoAnswer = errors.New("the node closed the query without an answer")

func New(targetID *astral.Identity, client *astrald.Client) *Client {
	if client == nil {
		client = astrald.Default()
	}
	return &Client{astral: client, targetID: targetID}
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
