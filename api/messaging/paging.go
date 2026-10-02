package messaging

import (
	"encoding/json"
	"io"

	"github.com/astralp2p/astral-go/astral"
)

// Paging reads a mailbox a bounded page at a time, in two directions.
//
// A page reads backwards by Cursor, the position a row was written at: the
// newest rows first, then each older page from the NextBefore the last one
// answered. A change read reads forwards by Rev, the position of a row's last
// change: every row written or changed after the Since a caller holds, in the
// order the changes happened. Both positions are node-wide orders, so a
// position is meaningful under any scope; a caller that changes scope starts a
// new traversal rather than reusing a position from another.
//
// Generation names the mailbox's rows as one set. A deleted mailbox takes a new
// one, and a request carrying an older generation is answered ErrGeneration
// rather than a page that silently names nothing.

// The ceiling and the default of every page and change read.
const (
	PageLimitDefault = 50
	PageLimitMax     = 100
)

// ListedMessage is one row of a page or a change read: its envelope, and the
// revision it carries.
//
// why a wrapper and not a field on Envelope: the envelope's wire encoding is the
// legacy listing's and the wait's, and stays as it is.
type ListedMessage struct {
	Rev      astral.Uint64
	Envelope *Envelope
}

// MessagePage is one page of a mailbox, newest first.
//
// NextBefore is the Before of the next older page, and 0 at the end. Rev is the
// revision the node had reached when it read the page: a caller following
// changes from the first page starts from it. Generation is the mailbox's.
type MessagePage struct {
	Messages   []*ListedMessage
	NextBefore astral.Uint64
	Rev        astral.Uint64
	Generation astral.Uint64
}

// MessageChanges is one read of what changed, oldest change first. Each row is
// answered at its latest state, archived rows included.
//
// NextRev is the Since of the next read: the last row's Rev, or the Since the
// read was given when nothing changed. More says rows are left after NextRev.
type MessageChanges struct {
	Messages   []*ListedMessage
	NextRev    astral.Uint64
	More       astral.Bool
	Generation astral.Uint64
}

// Conversation is one correspondent of a mailbox: its latest unarchived message
// and how many inbox messages from it wait unread and unarchived.
//
// Latest is nil for a tombstone: a conversation whose last unarchived message
// was put away. A page leaves tombstones out; a change read answers them, so a
// follower drops the row.
type Conversation struct {
	Peer   *astral.Identity
	Latest *Envelope
	Unread astral.Uint64
	Rev    astral.Uint64
}

// ConversationPage is one page of conversations, by latest message, newest
// first. NextBefore is the Before of the next page, and 0 at the end.
type ConversationPage struct {
	Conversations []*Conversation
	NextBefore    astral.Uint64
	Rev           astral.Uint64
	Generation    astral.Uint64
}

// ConversationChanges is one read of the conversations that changed, oldest
// change first, tombstones included.
type ConversationChanges struct {
	Conversations []*Conversation
	NextRev       astral.Uint64
	More          astral.Bool
	Generation    astral.Uint64
}

// astral

var (
	_ astral.Object = &ListedMessage{}
	_ astral.Object = &MessagePage{}
	_ astral.Object = &MessageChanges{}
	_ astral.Object = &Conversation{}
	_ astral.Object = &ConversationPage{}
	_ astral.Object = &ConversationChanges{}
)

func (ListedMessage) ObjectType() string       { return "messaging.listed_message" }
func (MessagePage) ObjectType() string         { return "messaging.message_page" }
func (MessageChanges) ObjectType() string      { return "messaging.message_changes" }
func (Conversation) ObjectType() string        { return "messaging.conversation" }
func (ConversationPage) ObjectType() string    { return "messaging.conversation_page" }
func (ConversationChanges) ObjectType() string { return "messaging.conversation_changes" }

func (m ListedMessage) WriteTo(w io.Writer) (int64, error) { return astral.Objectify(&m).WriteTo(w) }
func (m *ListedMessage) ReadFrom(r io.Reader) (int64, error) {
	return astral.Objectify(m).ReadFrom(r)
}

func (p MessagePage) WriteTo(w io.Writer) (int64, error)    { return astral.Objectify(&p).WriteTo(w) }
func (p *MessagePage) ReadFrom(r io.Reader) (int64, error)  { return astral.Objectify(p).ReadFrom(r) }
func (c MessageChanges) WriteTo(w io.Writer) (int64, error) { return astral.Objectify(&c).WriteTo(w) }
func (c *MessageChanges) ReadFrom(r io.Reader) (int64, error) {
	return astral.Objectify(c).ReadFrom(r)
}

func (c Conversation) WriteTo(w io.Writer) (int64, error)   { return astral.Objectify(&c).WriteTo(w) }
func (c *Conversation) ReadFrom(r io.Reader) (int64, error) { return astral.Objectify(c).ReadFrom(r) }
func (p ConversationPage) WriteTo(w io.Writer) (int64, error) {
	return astral.Objectify(&p).WriteTo(w)
}
func (p *ConversationPage) ReadFrom(r io.Reader) (int64, error) {
	return astral.Objectify(p).ReadFrom(r)
}
func (c ConversationChanges) WriteTo(w io.Writer) (int64, error) {
	return astral.Objectify(&c).WriteTo(w)
}
func (c *ConversationChanges) ReadFrom(r io.Reader) (int64, error) {
	return astral.Objectify(c).ReadFrom(r)
}

// json
//
// why through astral's codec: an empty slice then marshals as [] and never null.

func (m ListedMessage) MarshalJSON() ([]byte, error)       { return astral.Objectify(&m).MarshalJSON() }
func (p MessagePage) MarshalJSON() ([]byte, error)         { return astral.Objectify(&p).MarshalJSON() }
func (c MessageChanges) MarshalJSON() ([]byte, error)      { return astral.Objectify(&c).MarshalJSON() }
func (c Conversation) MarshalJSON() ([]byte, error)        { return astral.Objectify(&c).MarshalJSON() }
func (p ConversationPage) MarshalJSON() ([]byte, error)    { return astral.Objectify(&p).MarshalJSON() }
func (c ConversationChanges) MarshalJSON() ([]byte, error) { return astral.Objectify(&c).MarshalJSON() }

func (m *ListedMessage) UnmarshalJSON(b []byte) error {
	type alias ListedMessage
	return json.Unmarshal(b, (*alias)(m))
}

func (p *MessagePage) UnmarshalJSON(b []byte) error {
	type alias MessagePage
	return json.Unmarshal(b, (*alias)(p))
}

func (c *MessageChanges) UnmarshalJSON(b []byte) error {
	type alias MessageChanges
	return json.Unmarshal(b, (*alias)(c))
}

func (c *Conversation) UnmarshalJSON(b []byte) error {
	type alias Conversation
	return json.Unmarshal(b, (*alias)(c))
}

func (p *ConversationPage) UnmarshalJSON(b []byte) error {
	type alias ConversationPage
	return json.Unmarshal(b, (*alias)(p))
}

func (c *ConversationChanges) UnmarshalJSON(b []byte) error {
	type alias ConversationChanges
	return json.Unmarshal(b, (*alias)(c))
}

func init() {
	astral.MustAdd(&ListedMessage{})
	astral.MustAdd(&MessagePage{})
	astral.MustAdd(&MessageChanges{})
	astral.MustAdd(&Conversation{})
	astral.MustAdd(&ConversationPage{})
	astral.MustAdd(&ConversationChanges{})
}
