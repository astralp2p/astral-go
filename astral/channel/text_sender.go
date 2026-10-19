package channel

import (
	"bytes"
	"encoding"
	"encoding/base64"
	"fmt"
	"io"

	"github.com/astralp2p/astral-go/astral"
)

type TextSender struct {
	Base64 bool // force base64
	w      io.Writer
}

var _ Sender = &TextSender{}

func NewTextSender(w io.Writer) *TextSender {
	return &TextSender{w: w}
}

// Send writes the object as one line: a #[type] header followed by a text-marshaled body,
// or a base64-encoded body when Base64 is set or the object is not a TextMarshaler.
//
// why the line is built before anything is written: a body that fails to encode would
// otherwise follow a header already on the stream, and the next object's line would merge
// into it.
func (sender TextSender) Send(obj astral.Object) (err error) {
	var line = bytes.NewBuffer(nil)

	// write the type
	_, err = fmt.Fprintf(line, "#[%s]", obj.ObjectType())
	if err != nil {
		return
	}

	// check if the object is a TextMarshaler
	m, ok := obj.(encoding.TextMarshaler)
	if ok && !sender.Base64 {
		var text []byte

		// marshal the object into text
		text, err = m.MarshalText()
		if err != nil {
			return err
		}

		_, err = line.Write([]byte(" " + string(text) + "\n"))
		if err != nil {
			return
		}

		_, err = sender.w.Write(line.Bytes())
		return
	}

	_, err = line.Write([]byte(":"))
	if err != nil {
		return
	}

	enc := base64.NewEncoder(base64.StdEncoding, line)
	_, err = obj.WriteTo(enc)
	enc.Close()
	if err != nil {
		return
	}

	_, err = line.Write([]byte("\n"))
	if err != nil {
		return
	}

	_, err = sender.w.Write(line.Bytes())
	return err
}
