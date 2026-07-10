package objects

import (
	"github.com/astralp2p/astral-go/api/objects"
	"github.com/astralp2p/astral-go/astral"
	"github.com/astralp2p/astral-go/astral/channel"
	"github.com/astralp2p/astral-go/lib/query"
)

func (client *Client) Store(ctx *astral.Context, repo string, object astral.Object) (id *astral.ObjectID, err error) {
	ch, err := client.queryCh(ctx, objects.MethodStore, query.Args{"repo": repo})
	if err != nil {
		return
	}
	defer ch.Close()
	if err = ch.Send(object); err != nil {
		return
	}
	err = ch.Switch(channel.Expect(&id), channel.PassErrors)
	return
}
