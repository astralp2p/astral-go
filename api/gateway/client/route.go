package gateway

import (
	gw "github.com/astralp2p/astral-go/api/gateway"
	"github.com/astralp2p/astral-go/astral"
	"github.com/astralp2p/astral-go/lib/query"
)

// Route asks the gateway to establish a routed connection to target and
// returns the raw connection. When target is the gateway itself, the gateway
// accepts the connection as an inbound link; otherwise it forwards the
// connection to the next hop and pipes both sides. This is the low-level
// primitive nodes use to relay links through a gateway — most callers want
// Connect instead.
func (c *Client) Route(ctx *astral.Context, target *astral.Identity) (astral.Conn, error) {
	return c.query(ctx, gw.MethodNodeRoute, query.Args{"target": target})
}
