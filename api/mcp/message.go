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
// ParentID names the one message this message answers, and is the sender's
// claim as the content is. A message answering none carries the zero value. An
// exchange is the chain those links make: a query, never a record, and nothing
// is opened, owned or closed.
//
// why a sender may name any parent: the value is the sender's, while the sender
// and the recipient are the route's. Naming a message means naming a 128-bit
// identifier nobody published, and a recipient sees on every row who wrote it.
// A parent the recipient does not hold is stored as it stands — the link is a
// claim about another message, and a claim about a message nobody has is
// simply one nothing answers.
//
// Thread is dead. It named a flat exchange label before replies named their
// parent, and it is carried for compatibility alone: no node reads it, a sender
// may set it, and a recipient ignores it. The slot is never reused, because a
// MessageID in it and a MessageID in ParentID are the same sixteen bytes and
// nothing in the frame tells them apart.
//
// why ParentID is last: the binary channel frames a payload with a length
// prefix and decodes from that bounded buffer, so a reader that predates a
// field reads the ones before it and leaves the rest. That holds only for a
// field appended after the ones already there. Never insert one above.
//
// The other direction is not covered and never has been: a payload written
// before a field exists is short, and the read of the missing field answers
// EOF rather than a zero value. Thread carries the same asymmetry. It costs
// nothing while both ends of a frame are the same binary — mcp.message is
// delivered by a query that loops back through the router — and the answer
// when that stops being true is a version marker or a new object type, not a
// reader that guesses which fields a peer meant to write.
type Message struct {
	ID       MessageID
	Content  astral.String32
	Thread   MessageID
	ParentID MessageID
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
