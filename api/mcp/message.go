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
// why ParentID sits where Thread sat: Thread named a flat exchange label before
// a reply named the message it answers, and retiring it takes its slot rather
// than leaving a dead sixteen bytes on every message. The frame is positional
// and carries no version marker, so this is not a compatible change: a peer at
// the revision before it writes a thread where this reads a parent, and the
// substitution is type-correct and silent. It is safe here because one node
// delivers to itself — mcp.message is carried by a query that loops back
// through the router — so both ends of every frame are the same binary. A
// second node at a different revision is what makes it unsafe, and the answer
// then is a version marker or a new object type, never a reader guessing which
// field a peer meant.
//
// A field is otherwise only ever appended: the channel frames a payload with a
// length prefix and decodes from that bounded buffer, so a reader that predates
// a field reads the ones before it and leaves the rest. That holds only for a
// field added after the ones already there.
type Message struct {
	ID       MessageID
	Content  astral.String32
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
