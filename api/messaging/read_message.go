package messaging

import (
	"encoding/json"
	"io"

	"github.com/astralp2p/astral-go/astral"
)

// ReadMessage is one message a read answers, with what the read decided about
// it.
//
// Content is the body, or nil when this answer does not hand it out: a reply
// read under ChildrenEnvelopes, or any message the answer had no room for.
// Truncated says the reason was room, not the mode. ChildIDs are the ids of the
// message's direct replies, oldest first — the whole set, whatever the answer
// carried of them.
//
// why Content is a pointer: an empty body is a message whose words were empty,
// which is not a body withheld.
type ReadMessage struct {
	Envelope  *Envelope
	Content   *astral.String32
	ChildIDs  []MessageID
	Truncated astral.Bool
}

// astral

var _ astral.Object = &ReadMessage{}

func (m ReadMessage) ObjectType() string { return "messaging.read_message" }

func (m ReadMessage) WriteTo(w io.Writer) (n int64, err error) {
	return astral.Objectify(&m).WriteTo(w)
}

func (m *ReadMessage) ReadFrom(r io.Reader) (n int64, err error) {
	return astral.Objectify(m).ReadFrom(r)
}

// json

func (m ReadMessage) MarshalJSON() ([]byte, error) {
	type alias ReadMessage
	return json.Marshal(alias(m))
}

func (m *ReadMessage) UnmarshalJSON(bytes []byte) error {
	type alias ReadMessage
	var v alias

	err := json.Unmarshal(bytes, &v)
	if err != nil {
		return err
	}

	*m = ReadMessage(v)
	return nil
}

func init() {
	astral.MustAdd(&ReadMessage{})
}
