package objects

import (
	"github.com/astralp2p/astral-go/api/objects"
	"github.com/astralp2p/astral-go/astral"
	"github.com/astralp2p/astral-go/astral/channel"
	"github.com/astralp2p/astral-go/lib/query"
)

func (client *Client) NewMem(ctx *astral.Context, name string, size int64) error {
	// send the query
	ch, err := client.queryCh(ctx, objects.MethodNewMem, query.Args{
		"name": name,
		"size": size,
	})
	if err != nil {
		return err
	}
	defer ch.Close()

	// wait for ack
	return ch.Switch(channel.ExpectAck, channel.PassErrors, channel.WithContext(ctx))
}
