package messaging

import (
	"encoding/json"
	"io"

	"github.com/astralp2p/astral-go/astral"
)

// MessageRef names one row of the caller's own mail.
//
// why the box is part of the name: an id is the peer's to choose, so one owner
// may hold a row under it in each box, and the archive spans both. The box is
// never inferred from the id.
type MessageRef struct {
	Box astral.String8
	ID  MessageID
}

// astral

var _ astral.Object = &MessageRef{}

func (r MessageRef) ObjectType() string { return "messaging.message_ref" }

func (r MessageRef) WriteTo(w io.Writer) (n int64, err error) {
	return astral.Objectify(&r).WriteTo(w)
}

func (r *MessageRef) ReadFrom(rd io.Reader) (n int64, err error) {
	return astral.Objectify(r).ReadFrom(rd)
}

// json

func (r MessageRef) MarshalJSON() ([]byte, error) {
	type alias MessageRef
	return json.Marshal(alias(r))
}

func (r *MessageRef) UnmarshalJSON(bytes []byte) error {
	type alias MessageRef
	var v alias

	err := json.Unmarshal(bytes, &v)
	if err != nil {
		return err
	}

	*r = MessageRef(v)
	return nil
}

func init() {
	astral.MustAdd(&MessageRef{})
}
