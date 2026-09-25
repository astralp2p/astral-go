package messaging

import (
	"encoding/json"
	"io"

	"github.com/astralp2p/astral-go/astral"
)

// ReadMessagesRequest is the body of MethodReadMessages: whole messages from
// the caller's own mail, with their direct replies.
//
// Refs names between one and MaxReadRefs distinct rows; a repeat is read once.
// Children is ChildrenNone, ChildrenEnvelopes or ChildrenFull, and empty reads
// as ChildrenEnvelopes. MaxChildren bounds the replies answered per message: 0
// takes MaxChildren, and a larger value is clamped to it.
type ReadMessagesRequest struct {
	Refs        []*MessageRef
	Children    astral.String8
	MaxChildren astral.Uint16
}

// astral

var _ astral.Object = &ReadMessagesRequest{}

func (r ReadMessagesRequest) ObjectType() string { return "messaging.read_messages_request" }

func (r ReadMessagesRequest) WriteTo(w io.Writer) (n int64, err error) {
	return astral.Objectify(&r).WriteTo(w)
}

func (r *ReadMessagesRequest) ReadFrom(rd io.Reader) (n int64, err error) {
	return astral.Objectify(r).ReadFrom(rd)
}

// json

func (r ReadMessagesRequest) MarshalJSON() ([]byte, error) {
	type alias ReadMessagesRequest
	return json.Marshal(alias(r))
}

func (r *ReadMessagesRequest) UnmarshalJSON(bytes []byte) error {
	type alias ReadMessagesRequest
	var v alias

	err := json.Unmarshal(bytes, &v)
	if err != nil {
		return err
	}

	*r = ReadMessagesRequest(v)
	return nil
}

func init() {
	astral.MustAdd(&ReadMessagesRequest{})
}
