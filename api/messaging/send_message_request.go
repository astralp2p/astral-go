package messaging

import (
	"encoding/json"
	"io"

	"github.com/astralp2p/astral-go/astral"
)

// SendMessageRequest is the body of MethodSendMessage: one message from the
// calling participant.
//
// To names the recipient by hex identity or by alias. ParentID names the
// message this one answers; the zero value answers none. The sender is the
// caller and is never a field, so no value here can speak for another
// participant.
type SendMessageRequest struct {
	To       astral.String8
	Content  astral.String32
	ParentID MessageID
}

// astral

var _ astral.Object = &SendMessageRequest{}

func (r SendMessageRequest) ObjectType() string { return "messaging.send_message_request" }

func (r SendMessageRequest) WriteTo(w io.Writer) (n int64, err error) {
	return astral.Objectify(&r).WriteTo(w)
}

func (r *SendMessageRequest) ReadFrom(rd io.Reader) (n int64, err error) {
	return astral.Objectify(r).ReadFrom(rd)
}

// json

func (r SendMessageRequest) MarshalJSON() ([]byte, error) {
	type alias SendMessageRequest
	return json.Marshal(alias(r))
}

func (r *SendMessageRequest) UnmarshalJSON(bytes []byte) error {
	type alias SendMessageRequest
	var v alias

	err := json.Unmarshal(bytes, &v)
	if err != nil {
		return err
	}

	*r = SendMessageRequest(v)
	return nil
}

func init() {
	astral.MustAdd(&SendMessageRequest{})
}
