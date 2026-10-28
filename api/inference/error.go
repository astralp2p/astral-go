package inference

import (
	"fmt"
	"io"

	"github.com/astralp2p/astral-go/astral"
)

var _ astral.Error = &Error{}

// Error is a refusal or failure, the answer of either op in place of an
// error_message. Before the first chunk it is the whole answer; after some
// chunks those chunks stand. Either way eos follows it.
//
// why not error_message: a consumer must tell a request it may retry from one
// it must not, and a message string carries no kind, status or delay.
type Error struct {
	// Kind is ErrorRefused, ErrorBadRequest, ErrorContextOverflow,
	// ErrorRateLimited, ErrorUnavailable or ErrorFarSide.
	Kind astral.String8
	// Status is the server's HTTP status; 0 when there is none.
	Status astral.Uint32
	// Message is the server's message, word for word when there is one.
	Message astral.String16
	// RetryAfter is the number of seconds to wait before retrying; 0 when the
	// server gave none.
	RetryAfter astral.Uint32
}

func (Error) ObjectType() string { return "inference.error" }

func (v Error) WriteTo(w io.Writer) (int64, error) { return astral.Objectify(&v).WriteTo(w) }
func (v *Error) ReadFrom(r io.Reader) (int64, error) {
	return astral.Objectify(v).ReadFrom(r)
}
func (v Error) MarshalJSON() ([]byte, error)  { return astral.Objectify(&v).MarshalJSON() }
func (v *Error) UnmarshalJSON(b []byte) error { return astral.Objectify(v).UnmarshalJSON(b) }

// Error renders the kind, the status when there is one, and the message.
func (v *Error) Error() string {
	s := "inference " + string(v.Kind)
	if v.Status != 0 {
		s += fmt.Sprintf(" (status %d)", v.Status)
	}
	if v.Message != "" {
		s += ": " + string(v.Message)
	}
	return s
}

func init() { astral.MustAdd(&Error{}) }
