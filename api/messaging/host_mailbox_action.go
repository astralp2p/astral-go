package messaging

import (
	"io"

	"github.com/astralp2p/astral-go/api/auth"
	"github.com/astralp2p/astral-go/astral"
)

// HostMailboxAction requests permission for Actor to host the mailbox of
// MailboxID. The actor is the hosting node. MailboxID is the identity the
// mailbox belongs to. A node asks it before it serves that mailbox.
//
// The mailbox identity holds the action itself. The root rule, where every
// chain for this action ends, allows it exactly when MailboxID is nonzero and
// equals the actor. A node holds it only under a contract the mailbox identity
// signs: Issuer MailboxID, Subject the node, and a permit for this action with
// Delegation 0. Auth resolves the node's request by re-entering as the issuer
// of each contract the node is subject to, so the root rule allows the request
// only when that issuer is MailboxID. A contract from another identity, or one
// the node issues to itself, does not authorize it. Delegation 0 keeps the node
// from handing the hosting on.
//
// The action asks only whether the node may host the mailbox. It names no
// correspondent and no operation. Whom the owner writes to and takes mail from
// is SendAction and ReceiveAction; a grant of either is not a grant of this, and
// a grant of this is not a grant of either. A relay permit
// (mod.nodes.relay_for_action) does not grant hosting, and a hosting permit does
// not grant relaying.
type HostMailboxAction struct {
	auth.Action
	MailboxID *astral.Identity
}

func (HostMailboxAction) ObjectType() string { return "mod.messaging.host_mailbox_action" }

func (a HostMailboxAction) WriteTo(w io.Writer) (n int64, err error) {
	return astral.Objectify(&a).WriteTo(w)
}

func (a *HostMailboxAction) ReadFrom(r io.Reader) (n int64, err error) {
	return astral.Objectify(a).ReadFrom(r)
}

// json
//
// why declared and not left to the default: an embedded struct marshals inline
// unless something names it, so without these the action would reach a json
// peer as a flat object while the spec documents it as an Action beside
// MailboxID. Contract and Permit declare the same pair for the same reason.

func (a HostMailboxAction) MarshalJSON() ([]byte, error) {
	return astral.Objectify(&a).MarshalJSON()
}

func (a *HostMailboxAction) UnmarshalJSON(b []byte) error {
	return astral.Objectify(a).UnmarshalJSON(b)
}

// ApplyConstraints refuses a permit that carries any constraint, as SendAction
// does. The hosting permit carries none: the root rule binds MailboxID to the
// contract's issuer, so no constraint names the mailbox. Every constraint is
// therefore unknown to this action, and an unknown or malformed constraint is
// refused rather than ignored.
func (a HostMailboxAction) ApplyConstraints(cs *astral.Bundle) bool {
	return cs == nil || len(cs.Objects()) == 0
}

func init() { astral.MustAdd(&HostMailboxAction{}) }
