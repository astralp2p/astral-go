package messaging

import (
	"io"

	"github.com/astralp2p/astral-go/api/auth"
	"github.com/astralp2p/astral-go/astral"
)

// SendAction requests permission for Actor to send a message to ToID. The
// sender's node asks it before the message is sent.
//
// The actor is the sending participant, and the action asks whom that
// participant may write to. Whether the recipient takes the message is
// ReceiveAction. A message is delivered only when both are granted, and neither
// party's permission decides the other's.
type SendAction struct {
	auth.Action
	ToID *astral.Identity
}

func (SendAction) ObjectType() string { return "mod.messaging.send_action" }

func (a SendAction) WriteTo(w io.Writer) (n int64, err error) {
	return astral.Objectify(&a).WriteTo(w)
}

func (a *SendAction) ReadFrom(r io.Reader) (n int64, err error) {
	return astral.Objectify(a).ReadFrom(r)
}

// json
//
// why declared and not left to the default: an embedded struct marshals inline
// unless something names it, so without these the action would reach a json
// peer as a flat object while the spec documents it as an Action beside
// ToID. Contract and Permit declare the same pair for the same reason.

func (a SendAction) MarshalJSON() ([]byte, error) {
	return astral.Objectify(&a).MarshalJSON()
}

func (a *SendAction) UnmarshalJSON(b []byte) error {
	return astral.Objectify(a).UnmarshalJSON(b)
}

// ApplyConstraints refuses a permit that carries any constraint. This action
// does not evaluate constraints, and an action that does not evaluate them is
// permitted regardless of them — so a permit narrowed by its issuer would be
// honoured in full. Refusing is the bar that keeps the deferral safe until
// constraints are implemented.
func (a SendAction) ApplyConstraints(cs *astral.Bundle) bool {
	return cs == nil || len(cs.Objects()) == 0
}

func init() { astral.MustAdd(&SendAction{}) }
