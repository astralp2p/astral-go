package astral

import (
	"encoding/json"
	"io"
	"strings"
)

// Error is an error that is also a wire-serializable Object.
type Error interface {
	Object
	error
}

var _ Error = &ErrorMessage{}

type ErrorMessage struct {
	err string
}

func NewError(s string) Error {
	return &ErrorMessage{
		err: s,
	}
}

// Err wraps any error into a transmittable Error, preserving only its message.
func Err(err error) Error {
	return NewError(err.Error())
}

// astral

func (msg ErrorMessage) ObjectType() string {
	return "error_message"
}

func (msg ErrorMessage) WriteTo(w io.Writer) (n int64, err error) {
	return String16(msg.err).WriteTo(w)
}

func (msg *ErrorMessage) ReadFrom(r io.Reader) (n int64, err error) {
	return (*String16)(&msg.err).ReadFrom(r)
}

// json

func (msg ErrorMessage) MarshalJSON() ([]byte, error) {
	return json.Marshal(msg.err)
}

func (msg *ErrorMessage) UnmarshalJSON(bytes []byte) error {
	return json.Unmarshal(bytes, &msg.err)
}

// text

func (msg ErrorMessage) MarshalText() (text []byte, err error) {
	return []byte(msg.err), nil
}

func (msg *ErrorMessage) UnmarshalText(text []byte) error {
	msg.err = string(text)
	return nil
}

// other

func (msg ErrorMessage) Error() string {
	return msg.err
}

func (msg ErrorMessage) String() string {
	return msg.err
}

// Is reports whether target is an ErrorMessage with the same text, or with the
// text this message starts with before ": " and detail.
//
// why: a decoded message is a new value, so errors.Is would never match a
// sentinel by identity across the wire.
func (msg ErrorMessage) Is(target error) bool {
	t, ok := target.(*ErrorMessage)
	if !ok {
		return false
	}
	return msg.err == t.err || strings.HasPrefix(msg.err, t.err+": ")
}

func init() {
	MustAdd(&ErrorMessage{})
}
