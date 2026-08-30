package mcp

import (
	"encoding/json"
	"io"

	"github.com/astralp2p/astral-go/astral"
)

// Message is one message an agent sends to another agent. The recipient's node
// stores it and answers an Ack, and the recipient reads it on its own schedule,
// so neither agent has to be present while the other is. Carried by the
// MethodMessage query.
//
// why it names neither party: the sender is the query's caller and the
// recipient its target. A field would be a second claim about a fact the route
// already holds, and a sender could make the two disagree.
//
// ID is minted by the sender and names the message on both sides: the
// recipient reads by it, and a delivery that arrives twice is stored once.
//
// Thread names the exchange the message belongs to. A first message carries
// its own id, so every message is in a thread and a thread is the set of
// messages sharing the label — a query, never a record. A reply copies the
// value unchanged, so a reply to a reply carries the root's: the label is flat
// and never a tree.
//
// why Thread is last: the binary channel frames a payload with a length
// prefix and decodes from that bounded buffer, so a reader that predates this
// field reads ID and Content and leaves the rest. That holds only for a field
// appended after the ones already there. Never insert one above.
//
// why a sender may name any thread: the value is the sender's claim, as the
// content is, while the sender and recipient are the route's. Joining an
// exchange means naming a 128-bit identifier nobody published, and a recipient
// sees on every row who wrote it.
type Message struct {
	ID      MessageID
	Content astral.String32
	Thread  MessageID
}

// astral

var _ astral.Object = &Message{}

func (m Message) ObjectType() string { return "mcp.message" }

func (m Message) WriteTo(w io.Writer) (n int64, err error) {
	return astral.Objectify(&m).WriteTo(w)
}

func (m *Message) ReadFrom(r io.Reader) (n int64, err error) {
	return astral.Objectify(m).ReadFrom(r)
}

// json

func (m Message) MarshalJSON() ([]byte, error) {
	type alias Message
	return json.Marshal(alias(m))
}

func (m *Message) UnmarshalJSON(bytes []byte) error {
	type alias Message
	var v alias

	err := json.Unmarshal(bytes, &v)
	if err != nil {
		return err
	}

	*m = Message(v)
	return nil
}

func init() {
	astral.MustAdd(&Message{})
}
