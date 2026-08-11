package objects

import (
	"github.com/astralp2p/astral-go/api/objects"
	"github.com/astralp2p/astral-go/astral"
	"github.com/astralp2p/astral-go/astral/channel"
	"github.com/astralp2p/astral-go/lib/query"
)

// RemoveRepository removes the named repository from the node's configuration.
func (client *Client) RemoveRepository(ctx *astral.Context, name string) (err error) {
	ch, err := client.queryCh(ctx, objects.MethodRemoveRepository, query.Args{"name": name})
	if err != nil {
		return
	}
	defer ch.Close()

	return ch.Switch(channel.ExpectAck, channel.PassErrors)
}

// RemoveRepository calls the operation on the default client.
func RemoveRepository(ctx *astral.Context, name string) error {
	return Default().RemoveRepository(ctx, name)
}
