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
// why it is thin, and what the other type is: this is the frame that crosses a
// link, never the record. StoredMessage is what a node holds once a delivery
// lands — the parties the route authenticated, the box the row sits in, and the
// instants that node stamped, none of which a sender may state.
//
// ID is minted by the sender and names the message on both sides: the
// recipient reads by it, and a delivery that arrives twice is stored once.
//
// ParentID names the one message this message answers, and is the sender's
// claim as the content is. A message answering none carries the zero value. An
// exchange is the chain those links make: a query, never a record, and nothing
// is opened, owned or closed.
//
// why the parent is a claim and is still refused when unheld: the value is the
// sender's, while the sender and the recipient are the route's. The recipient's
// node refuses a parent it does not hold, and the sending agent's node refuses
// one that agent does not hold — a message has one of each, so a parent is a
// message between exactly these two parties, and no agent replies into an
// exchange it is not part of. That also makes an exchange a forest: every
// parent names a message stored earlier, so no chain of links returns to where
// it began, and a message naming itself is refused as the same rule's cheapest
// case.
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
