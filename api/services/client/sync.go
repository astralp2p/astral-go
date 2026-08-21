package services

import (
	"github.com/astralp2p/astral-go/api/services"
	"github.com/astralp2p/astral-go/astral"
	"github.com/astralp2p/astral-go/astral/channel"
	"github.com/astralp2p/astral-go/lib/query"
)

// Sync fetches and caches id's services from the network, and blocks until the
// fetch completes. When follow is true the node keeps the cache updated after
// the initial fetch.
func (client *Client) Sync(ctx *astral.Context, id string, follow bool) (err error) {
	ch, err := client.queryCh(ctx, services.MethodSync, query.Args{"id": id, "follow": follow})
	if err != nil {
		return
	}
	defer ch.Close()

	return ch.Switch(channel.ExpectAck, channel.PassErrors)
}

// Sync calls the operation on the default client.
func Sync(ctx *astral.Context, id string, follow bool) error {
	return Default().Sync(ctx, id, follow)
}
