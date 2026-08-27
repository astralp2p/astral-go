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
type Message struct {
	ID      MessageID
	Content astral.String32
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
