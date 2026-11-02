package apphost

import (
	"fmt"

	"github.com/astralp2p/astral-go/api/apphost"
	"github.com/astralp2p/astral-go/astral"
)

// Grant asks the target node to write the node-local grants req names. Each
// permit, with its constraints, becomes one grant for req.Identity that
// authorizes on that node alone and travels nowhere. A zero req.Duration
// writes grants that last until revoked.
//
// Grants add up: a grant never replaces another. Granting a permit identical
// to one already held sets that grant's expiry to the new value. Narrowing
// is granting the narrow permit, then revoking the wide one with RevokeGrant.
//
// The caller must hold mod.auth.admin_manage_apps_action. The node rejects
// any other caller, and any query that arrived over a link, before it answers.
// A failure past the accept gate arrives as an error: an invalid permit as
// auth.ErrInvalidPermit, and an answer that ends without an ack as
// io.ErrUnexpectedEOF. A node that does not ack before ctx ends fails Grant
// with ctx's error; the grants may still be written. A nil req is refused with
// apphost.ErrProtocolError before anything is sent.
func (client *Client) Grant(ctx *astral.Context, req *apphost.GrantRequest) error {
	if req == nil {
		return fmt.Errorf("%w: nil grant request", apphost.ErrProtocolError)
	}

	ch, err := client.queryCh(ctx, apphost.MethodGrant, nil)
	if err != nil {
		return err
	}
	defer ch.Close()

	if err = ch.Send(req); err != nil {
		return err
	}

	return expectAck(ctx, ch)
}

// Grant calls the operation on the default client.
func Grant(ctx *astral.Context, req *apphost.GrantRequest) error {
	return Default().Grant(ctx, req)
}
