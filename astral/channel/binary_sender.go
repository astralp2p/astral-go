package channel

import (
	"bytes"
	"io"

	"github.com/astralp2p/astral-go/astral"
)

// BinarySender writes a stream of astral.Objects to the underlying io.Writer.
type BinarySender struct {
	w io.Writer
}

var _ Sender = &BinarySender{}

func NewBinarySender(w io.Writer) *BinarySender {
	return &BinarySender{w: w}
}

// Send writes object as one frame: its type, then its payload with a 32-bit
// length prefix.
//
// why the frame is built before anything is written: a payload that fails to
// encode would otherwise follow a type already on the stream, and every later
// object on the channel would decode from the wrong offset.
func (w BinarySender) Send(object astral.Object) (err error) {
	var payload = bytes.NewBuffer(nil)
	_, err = object.WriteTo(payload)
	if err != nil {
		return
	}

	var frame = bytes.NewBuffer(nil)
	_, err = astral.String8(object.ObjectType()).WriteTo(frame)
	if err != nil {
		return
	}
	_, err = astral.Bytes32(payload.Bytes()).WriteTo(frame)
	if err != nil {
		return
	}

	_, err = w.w.Write(frame.Bytes())

	return
}
