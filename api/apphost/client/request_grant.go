package apphost

import (
	"fmt"

	"github.com/astralp2p/astral-go/api/apphost"
	"github.com/astralp2p/astral-go/astral"
)

// RequestGrant asks the target node to grant the calling session the permits
// req names, with their constraints. The grantee is the identity of the
// session's access token. A zero req.Duration asks for grants that last until
// revoked. Approved grants add to those the session already holds.
//
// The node accepts the query before its grant request policy decides, and it
// asks the policy on every call, even for permits the session already holds.
// A delegated policy may wait on a person. The wait ends with ctx: RequestGrant
// closes the query and returns ctx's error.
//
// A refusal arrives as apphost.ErrGrantDeclined, an invalid permit as
// auth.ErrInvalidPermit, and an answer that ends without an ack as
// io.ErrUnexpectedEOF. A nil req is refused with apphost.ErrProtocolError
// before anything is sent.
func (client *Client) RequestGrant(ctx *astral.Context, req *apphost.RequestGrantRequest) error {
	if req == nil {
		return fmt.Errorf("%w: nil request grant request", apphost.ErrProtocolError)
	}

	ch, err := client.queryCh(ctx, apphost.MethodRequestGrant, nil)
	if err != nil {
		return err
	}
	defer ch.Close()

	if err = ch.Send(req); err != nil {
		return err
	}

	return expectAck(ctx, ch)
}

// RequestGrant calls the operation on the default client.
func RequestGrant(ctx *astral.Context, req *apphost.RequestGrantRequest) error {
	return Default().RequestGrant(ctx, req)
}
