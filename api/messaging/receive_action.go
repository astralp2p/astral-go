package messaging

import (
	"io"

	"github.com/astralp2p/astral-go/api/auth"
	"github.com/astralp2p/astral-go/astral"
)

// ReceiveAction requests permission for Actor to take a message from FromID.
// The recipient's node asks it before it stores a delivery; a refusal answers
// the sender RejectNotAdmitted. A receipt asks no action: the outbox row it
// names admits it.
//
// The actor is the recipient, not the sender. Every action type names what its
// actor does, and taking a message is the recipient's act.
//
// why it matters beyond the name: auth resolves an action no handler grants by
// walking the contracts the actor is subject to, re-entering as each issuer. An
// action naming the sender as actor would search the sender's delegations for a
// permission the recipient's side holds, and a stranger's contracts would
// decide what this participant takes.
//
// Whom the sender may write to is SendAction. A message is delivered only when
// both are granted, and neither party's permission decides the other's.
type ReceiveAction struct {
	auth.Action
	FromID *astral.Identity
}

func (ReceiveAction) ObjectType() string { return "mod.messaging.receive_action" }

func (a ReceiveAction) WriteTo(w io.Writer) (n int64, err error) {
	return astral.Objectify(&a).WriteTo(w)
}

func (a *ReceiveAction) ReadFrom(r io.Reader) (n int64, err error) {
	return astral.Objectify(a).ReadFrom(r)
}

// json
//
// why declared and not left to the default: an embedded struct marshals inline
// unless something names it, so without these the action would reach a json
// peer as a flat object while the spec documents it as an Action beside
// FromID. Contract and Permit declare the same pair for the same reason.

func (a ReceiveAction) MarshalJSON() ([]byte, error) {
	return astral.Objectify(&a).MarshalJSON()
}

func (a *ReceiveAction) UnmarshalJSON(b []byte) error {
	return astral.Objectify(a).UnmarshalJSON(b)
}

// ApplyConstraints refuses a permit that carries any constraint, as SendAction
// does and for the same reason: nothing evaluates them, so a permit its issuer
// narrowed would otherwise be honoured in full.
func (a ReceiveAction) ApplyConstraints(cs *astral.Bundle) bool {
	return cs == nil || len(cs.Objects()) == 0
}

func init() { astral.MustAdd(&ReceiveAction{}) }
