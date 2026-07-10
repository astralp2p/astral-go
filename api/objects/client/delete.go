package objects

import (
	"github.com/astralp2p/astral-go/api/objects"
	"github.com/astralp2p/astral-go/astral"
	"github.com/astralp2p/astral-go/astral/channel"
	"github.com/astralp2p/astral-go/lib/query"
)

func (client *Client) Delete(ctx *astral.Context, objectID *astral.ObjectID, repo string) error {
	ch, err := client.queryCh(ctx, objects.MethodDelete, query.Args{
		"id":   objectID,
		"repo": repo,
	})
	if err != nil {
		return err
	}
	defer ch.Close()

	return ch.Switch(channel.ExpectAck, channel.PassErrors, channel.WithContext(ctx))
}

func Delete(ctx *astral.Context, objectID *astral.ObjectID, repo string) error {
	return Default().Delete(ctx, objectID, repo)
}
