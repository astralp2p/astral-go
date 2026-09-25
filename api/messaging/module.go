/*
Package messaging describes a module that keeps mail for the identities whose
mailboxes a node hosts.

A participant is an identity with a mailbox: an inbox and an outbox. The node
mints it with MethodCreateIdentity and knows no role beside it: an MCP agent and
an account owner are both participants, and what makes them either is the
product holding their credentials.

A node hosts a mailbox only under a contract the mailbox identity signs: Issuer
the mailbox identity, Subject the node, and a permit for
mod.messaging.host_mailbox_action (HostMailboxAction) with Delegation 0 and no
constraints. The mailbox identity holds that action itself: the root rule
allows it exactly when the action's MailboxID is nonzero and equals its actor.
The node's request therefore reaches the root only through a contract the
mailbox identity issued; a contract from another identity, or one the node
issues to itself, grants no hosting. MethodCreateIdentity signs the hosting
contract with the key it mints, beside the identity's relay contract, and
neither contract grants what the other does. MethodDeleteIdentity withdraws
hosting on the node alone: it revokes no contract, and the hosting contract
stays valid until it expires.

Two participants exchange mail across mod.messaging.send_action, asked of the
sender, and mod.messaging.receive_action, asked of the recipient; the auth
module answers both. Hosting is a separate question: a hosting grant admits no
message, and neither mail action grants hosting.

Every operation is local-only. A query arriving over a link is rejected, and so
is a query carrying the mcp origin, so an agent reaches none of these
operations on its own host node. A caller of the mail operations acts on its own
boxes, and the node must host the caller's mailbox.

A participant answers two queries of its own. MethodMessage carries a Message to
the participant's identity, and its node stores it in that participant's inbox.
MethodReceipt carries a Receipt back to a sender, and the sender's node stamps
the message collected. Both are addressed to a participant rather than to a
node, so they are the queries here a caller reaches over a link. The node
sending either query routes it as itself, with the participant the query comes
from as its caller — the message's sender for a delivery, its recipient for a
receipt — so a link carries both as relay queries. A node takes either only when
it arrived over a link or came from the node's own send path, and rejects any
other copy whatever its target; it answers them only for a mailbox it hosts,
and any other path addressed to that mailbox is a missing route.
*/
package messaging

const (
	MethodCreateIdentity = "messaging.create_identity"
	MethodIdentity       = "messaging.identity"
	MethodDeleteIdentity = "messaging.delete_identity"
	MethodSendMessage    = "messaging.send_message"
	MethodListMessages   = "messaging.list_messages"
	MethodReadMessages   = "messaging.read_messages"
	MethodWait           = "messaging.wait"
	MethodArchive        = "messaging.archive"
)

// MethodMessage is the query that delivers a Message, addressed to the
// recipient participant's identity. It is not an operation: no node serves it,
// and the participant's own node answers it on the participant's behalf.
const MethodMessage = "messaging.message"

// MethodReceipt is the query that carries a Receipt, addressed to the original
// sender's identity. Like MethodMessage it is not an operation: no node serves
// it, and the sender's own node answers it on the sender's behalf.
//
// why neither is an operation: an operation is addressed to a node's identity,
// and both of these are addressed to a participant's. A node mounts its
// modules' operations behind that check, so an operation carrying a receipt
// would be unreachable by the only caller that ever sends one.
//
// why it is the reverse of MethodMessage: the recipient calls and the sender is
// the target, so the pair of identities on the route is the same pair the
// delivery carried, exchanged.
const MethodReceipt = "messaging.receipt"

// RejectNotAdmitted is the reject code a participant's node answers a caller
// whose message the participant's own side will not take. It is
// operation-specific and so sits above the reserved generic codes 0-4.
//
// why a reject code and not a missing route: a missing route is also the answer
// for an identity no node holds and for a node that could not be reached, so a
// caller reading one could not tell a door closed to it from a door that is not
// there — the first is permanent and the second is worth retrying.
//
// The code carries no reason. Which senders a participant takes is held by an
// authority the node asks and that answers one bit, so every ground for the
// refusal reaches the caller as this one code.
//
// note: the sender reads this code only when the recipient's mailbox is on the
// sender's own node. A sending node reaches a recipient on another node through
// the recipient's relay contract, and its relay path answers a relay's
// rejection as a missing route. Across nodes the recipient's node still answers
// this code over the link, and the sender reads a missing route.
const RejectNotAdmitted = 5

// The two boxes a stored message sits in. A message is in one of them for its
// whole life: inbox is what was written to the owner, outbox what the owner
// wrote. The archive is a state and not a third box — ArchivedAt carries it.
const (
	BoxInbox  = "inbox"
	BoxOutbox = "outbox"
)

// The lists MethodListMessages answers. ListInbox and ListOutbox hold what the
// owner has not archived, from either box as named; ListArchive holds what it
// has, from both.
const (
	ListInbox   = "inbox"
	ListOutbox  = "outbox"
	ListArchive = "archive"
)

// How much of each message's direct replies MethodReadMessages answers:
// ChildrenNone none, ChildrenEnvelopes each reply without its body, and
// ChildrenFull each reply with it. An empty value reads as ChildrenEnvelopes.
//
// The reply ids come back in every mode: they are the shape of the
// conversation, and the mode decides only how much of the replies' content
// one answer carries.
const (
	ChildrenNone      = "none"
	ChildrenEnvelopes = "envelopes"
	ChildrenFull      = "full"
)

// The module's bounds on one read. MaxReadRefs is the most distinct messages
// one read names. MaxChildren is the most replies one read answers per message
// and the default when a request names none.
//
// why the bounds are the module's: a body may be 64 KiB and a message may have
// any number of replies, so an unbounded read is the caller deciding how much
// of the reader's context a stranger fills.
const (
	MaxReadRefs = 20
	MaxChildren = 10
)
