package apphost

import (
	"github.com/astralp2p/astral-go/api/apphost"
	"github.com/astralp2p/astral-go/astral"
	"github.com/astralp2p/astral-go/lib/query"
)

// RevokeGrant deletes the one node-local grant named id from identity on the
// target node. id comes from ListGrants and must name a grant identity holds.
// The caller must hold mod.auth.admin_manage_apps_action. An answer that ends
// without an ack is an error, and the read ends with ctx.
func (client *Client) RevokeGrant(ctx *astral.Context, identity *astral.Identity, id astral.Uint64) error {
	ch, err := client.queryCh(ctx, apphost.MethodRevoke, query.Args{"identity": identity.String(), "id": id})
	if err != nil {
		return err
	}
	defer ch.Close()

	return expectAck(ctx, ch)
}

// RevokeAction deletes every node-local grant identity holds for action on
// the target node. The caller must hold mod.auth.admin_manage_apps_action. An
// answer that ends without an ack is an error, and the read ends with ctx.
func (client *Client) RevokeAction(ctx *astral.Context, identity *astral.Identity, action string) error {
	ch, err := client.queryCh(ctx, apphost.MethodRevoke, query.Args{"identity": identity.String(), "action": action})
	if err != nil {
		return err
	}
	defer ch.Close()

	return expectAck(ctx, ch)
}

// RevokeGrant calls the operation on the default client.
func RevokeGrant(ctx *astral.Context, identity *astral.Identity, id astral.Uint64) error {
	return Default().RevokeGrant(ctx, identity, id)
}

// RevokeAction calls the operation on the default client.
func RevokeAction(ctx *astral.Context, identity *astral.Identity, action string) error {
	return Default().RevokeAction(ctx, identity, action)
}
