package messaging

import (
	"encoding/json"
	"io"

	"github.com/astralp2p/astral-go/astral"
)

// ReadMessagesResult is what MethodReadMessages answers. Messages are the rows
// the request named and the caller holds, in the request's order. Replies are
// their direct replies the answer carries, as the request's Children mode
// asked. NotFound names every requested row the caller does not hold; the rest
// are read regardless.
//
// why the replies are a flat set beside the messages: the edge is on the reply,
// which names its parent in ParentID, and a nested answer refers to its own
// type — a shape the SDK's schema generator refuses.
type ReadMessagesResult struct {
	Messages []*ReadMessage
	Replies  []*ReadMessage
	NotFound []*MessageRef
}

// astral

var _ astral.Object = &ReadMessagesResult{}

func (r ReadMessagesResult) ObjectType() string { return "messaging.read_messages_result" }

func (r ReadMessagesResult) WriteTo(w io.Writer) (n int64, err error) {
	return astral.Objectify(&r).WriteTo(w)
}

func (r *ReadMessagesResult) ReadFrom(rd io.Reader) (n int64, err error) {
	return astral.Objectify(r).ReadFrom(rd)
}

// json

func (r ReadMessagesResult) MarshalJSON() ([]byte, error) {
	type alias ReadMessagesResult
	return json.Marshal(alias(r))
}

func (r *ReadMessagesResult) UnmarshalJSON(bytes []byte) error {
	type alias ReadMessagesResult
	var v alias

	err := json.Unmarshal(bytes, &v)
	if err != nil {
		return err
	}

	*r = ReadMessagesResult(v)
	return nil
}

func init() {
	astral.MustAdd(&ReadMessagesResult{})
}
