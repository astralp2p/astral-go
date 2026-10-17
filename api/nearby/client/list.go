package nearby

import (
	"github.com/astralp2p/astral-go/api/nearby"
	"github.com/astralp2p/astral-go/astral"
	"github.com/astralp2p/astral-go/astral/channel"

	// why: a Status bundle carries attachments of any type (mod.dir.alias,
	// mod.nodes.endpoint_with_ttl, ...), and decoding one needs its blueprint.
	_ "github.com/astralp2p/astral-go"
)

// List lists the nearby nodes the target node holds a status of, one Status
// each. A status expires five minutes after the node received it.
//
// The target rejects a caller that does not hold mod.auth.admin_network_action.
func (client *Client) List(ctx *astral.Context) ([]*nearby.Status, error) {
	ch, err := client.queryCh(ctx, nearby.MethodList, nil)
	if err != nil {
		return nil, err
	}
	defer ch.Close()

	var list []*nearby.Status
	err = ch.Switch(
		channel.Collect(&list),
		channel.BreakOnEOS,
		channel.PassErrors,
	)

	return list, err
}

// List calls the operation on the default client.
func List(ctx *astral.Context) ([]*nearby.Status, error) {
	return Default().List(ctx)
}
