package messaging

import (
	"io"

	"github.com/astralp2p/astral-go/api/auth"
	"github.com/astralp2p/astral-go/astral"
)

// ReadMailboxAction requests permission for Actor to read the mailbox of
// MailboxID. The actor is the reading identity. MailboxID is the identity the
// mailbox belongs to. The node hosting that mailbox asks it before it answers a
// MethodListMessages or MethodReadMessages that names a mailbox other than the
// caller's own.
//
// No root rule answers the action, and no permit carries a delegated read:
// ApplyConstraints refuses every permit. Only a handler registered for the
// action, or the external authority the node names for it, asked about the
// reader itself, grants the read. A node with neither refuses every request.
// A contract naming the action carries no read, whoever its issuer is: a node
// holds the key of every identity it minted and signs a contract with any key
// it holds, so no contract signed with such a key is evidence of its signer's
// consent.
//
// The action grants reading only. A delegated read never stamps: the messages
// it answers, and their replies under ChildrenFull, are not marked read, and no
// sender is told a body was collected, by receipt or by a stamp on its outbox.
// The action grants no sending, archiving or waiting, and no hosting: that is
// HostMailboxAction, and neither grants the other.
type ReadMailboxAction struct {
	auth.Action
	MailboxID *astral.Identity
}

func (ReadMailboxAction) ObjectType() string { return "mod.messaging.read_mailbox_action" }

func (a ReadMailboxAction) WriteTo(w io.Writer) (n int64, err error) {
	return astral.Objectify(&a).WriteTo(w)
}

func (a *ReadMailboxAction) ReadFrom(r io.Reader) (n int64, err error) {
	return astral.Objectify(a).ReadFrom(r)
}

// json
//
// why declared and not left to the default: an embedded struct marshals inline
// unless something names it, so without these the action would reach a json
// peer as a flat object while the spec documents it as an Action beside
// MailboxID. Contract and Permit declare the same pair for the same reason.

func (a ReadMailboxAction) MarshalJSON() ([]byte, error) {
	return astral.Objectify(&a).MarshalJSON()
}

func (a *ReadMailboxAction) UnmarshalJSON(b []byte) error {
	return astral.Objectify(a).UnmarshalJSON(b)
}

// ApplyConstraints refuses every permit, constrained or not, so Permit.Allows
// answers false for the action and no contract carries it: an auth chain walk
// passes no contract link for a delegated read, and the authority is asked
// about the reader alone.
func (ReadMailboxAction) ApplyConstraints(*astral.Bundle) bool {
	return false
}

func init() { astral.MustAdd(&ReadMailboxAction{}) }
