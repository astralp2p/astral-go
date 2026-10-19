package channel

import (
	"bytes"
	"fmt"
	"io"

	"github.com/astralp2p/astral-go/astral"
)

type CanonicalSender struct {
	w io.Writer
}

var _ Sender = &CanonicalSender{}

func NewCanonicalSender(w io.Writer) *CanonicalSender {
	return &CanonicalSender{w: w}
}

// Send writes Stamp, type, then payload with no length framing.
//
// why the frame is built before anything is written: a payload that fails to
// encode would otherwise follow a stamp and type already on the stream, and
// every later object on the channel would decode from the wrong offset.
func (c CanonicalSender) Send(object astral.Object) (err error) {
	var frame = bytes.NewBuffer(nil)
	_, err = astral.Stamp{}.WriteTo(frame)
	if err != nil {
		return fmt.Errorf("error writing stamp: %w", err)
	}

	_, err = astral.ObjectType(object.ObjectType()).WriteTo(frame)
	if err != nil {
		return fmt.Errorf("error writing object type: %w", err)
	}

	_, err = object.WriteTo(frame)
	if err != nil {
		return fmt.Errorf("error writing object: %w", err)
	}

	_, err = c.w.Write(frame.Bytes())

	return
}
