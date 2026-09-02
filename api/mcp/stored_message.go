package mcp

import (
	"encoding/json"
	"io"

	"github.com/astralp2p/astral-go/astral"
)

// The two boxes a stored message sits in. A message is in one of them for its
// whole life: inbox is what was written to the owner, outbox what the owner
// wrote. The archive is a state and not a third box — ArchivedAt carries it.
const (
	BoxInbox  = "inbox"
	BoxOutbox = "outbox"
)

// StoredMessage is one message as a node holds it, in one agent's box. Every
// message a node carries is two of these — the sender's and the recipient's,
// differing in Box and owner and in nothing else — and across nodes only one of
// them is on any given machine.
//
// why this and Message are two types: Message is the frame that crosses a link,
// and it names neither party because the route already does. This is what a
// node holds afterwards: the parties the route authenticated, the box the row
// sits in, and the instants that node stamped. Neither is derivable from the
// other, and a single type would either put a spoofable claim on the wire or
// leave a reader unable to say who wrote what. AgentInfo sits beside Agent for
// the same reason.
//
// why the optional instants are pointers: an unset instant is the absence of
// the fact, never a value somebody chose, and astral.Time has no spare value to
// say so — the zero time's UnixNano overflows, so a zero encodes and decodes as
// an instant in 1754. A nil-flag says it plainly. A row carrying CreatedAt
// alone is a send whose fate is unknown, which is the honest answer after a
// crash: an acknowledgement that never arrived proves nothing about the write.
type StoredMessage struct {
	// Cursor is the position the node wrote this row at, in its own order. It
	// is opaque: only its order is a fact.
	//
	// why a position and not an instant: CreatedAt is chosen before the row
	// commits, so a row can carry an earlier instant and appear later, and a
	// cursor over it steps past a message permanently. Only the inbox pages by
	// this; the other lists are histories read newest first.
	Cursor astral.Uint64

	// ID names the message on both sides, minted by the sender. Box is BoxInbox
	// or BoxOutbox, and the two together with the owner name one row: an id is
	// the peer's to choose, so one owner may hold two rows under it.
	ID  MessageID
	Box astral.String8

	// Sender and Recipient are the parties the route authenticated when the
	// message was delivered, never a claim the message carried.
	Sender    *astral.Identity
	Recipient *astral.Identity

	Content  astral.String32
	ParentID MessageID

	// CreatedAt is when this node wrote this row: the recipient's arrival on an
	// inbox row, the sender's attempt on an outbox row.
	CreatedAt astral.Time

	// ArchivedAt is the owner putting the message away. It is the owner's own
	// bookkeeping, it crosses no link, and it is the one stamp with an inverse.
	ArchivedAt *astral.Time

	// Inbox only. ReadAt is when the body was handed to the owner,
	// ReceiptDueAt when a receipt became owed to the sender, and
	// ReceiptStoredAt when the sender's node acknowledged that receipt.
	ReadAt          *astral.Time
	ReceiptDueAt    *astral.Time
	ReceiptStoredAt *astral.Time

	// Outbox only. LandedAt is the recipient's node acknowledging the write,
	// FailedAt a delivery known not to have been stored, and FetchedAt the body
	// handed out — which reports a collection and never that a model read it.
	LandedAt  *astral.Time
	FailedAt  *astral.Time
	FetchedAt *astral.Time

	// Err is the recipient's node's own words for a refusal, bounded by the
	// storing node and marked where it was cut. It is quoted material: another
	// operator wrote it, and nothing acts on it.
	//
	// why a pointer here too: an empty string is a refusal whose words were
	// empty, which is not the absence of a refusal.
	Err *astral.String16
}

// astral

var _ astral.Object = &StoredMessage{}

func (m StoredMessage) ObjectType() string { return "mcp.stored_message" }

func (m StoredMessage) WriteTo(w io.Writer) (n int64, err error) {
	return astral.Objectify(&m).WriteTo(w)
}

func (m *StoredMessage) ReadFrom(r io.Reader) (n int64, err error) {
	return astral.Objectify(m).ReadFrom(r)
}

// json

func (m StoredMessage) MarshalJSON() ([]byte, error) {
	type alias StoredMessage
	return json.Marshal(alias(m))
}

func (m *StoredMessage) UnmarshalJSON(bytes []byte) error {
	type alias StoredMessage
	var v alias

	err := json.Unmarshal(bytes, &v)
	if err != nil {
		return err
	}

	*m = StoredMessage(v)
	return nil
}

func init() {
	astral.MustAdd(&StoredMessage{})
}
