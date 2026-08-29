package mcp

import (
	"io"

	"github.com/astralp2p/astral-go/astral"
)

// Receipt is the recipient's node telling a sender that a message's body was
// handed out. Carried by the MethodReceipt query.
//
// why it names neither party: the recipient is the query's caller and the
// original sender its target — the reverse of a Message. A field would be a
// second claim about a fact the route already holds.
//
// The receipt carries one attempt and no state of its own. The fact it reports
// is already true and durable on the node that sends it, so a receipt lost in
// transit costs the sender a stamp and nothing else.
type Receipt struct {
	ID MessageID
}

// astral

var _ astral.Object = &Receipt{}

func (r Receipt) ObjectType() string { return "mcp.receipt" }

func (r Receipt) WriteTo(w io.Writer) (n int64, err error) {
	return astral.Objectify(&r).WriteTo(w)
}

func (r *Receipt) ReadFrom(rd io.Reader) (n int64, err error) {
	return astral.Objectify(r).ReadFrom(rd)
}

func init() {
	astral.MustAdd(&Receipt{})
}
