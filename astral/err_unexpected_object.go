package astral

import (
	"fmt"
	"io"
)

type ErrUnexpectedObject struct {
	Object Object
}

var _ JSONObject = &ErrUnexpectedObject{}
var _ error = &ErrUnexpectedObject{}

func NewErrUnexpectedObject(object Object) *ErrUnexpectedObject {
	return &ErrUnexpectedObject{Object: object}
}

func (ErrUnexpectedObject) ObjectType() string {
	return "err_unexpected_object"
}

func (e ErrUnexpectedObject) WriteTo(writer io.Writer) (n int64, err error) {
	return Objectify(&e).WriteTo(writer)
}

func (e *ErrUnexpectedObject) ReadFrom(reader io.Reader) (n int64, err error) {
	return Objectify(e).ReadFrom(reader)
}

func (e ErrUnexpectedObject) MarshalJSON() ([]byte, error) {
	return Objectify(&e).MarshalJSON()
}

func (e *ErrUnexpectedObject) UnmarshalJSON(bytes []byte) error {
	return Objectify(e).UnmarshalJSON(bytes)
}

// Error names the unexpected object's type. An absent Object names the Nil marker.
func (e *ErrUnexpectedObject) Error() string {
	// why: an absent Object is reachable — decoding a wire or JSON form whose
	// Object slot is empty leaves it nil — and a typed nil pointer panics on a
	// value-receiver ObjectType. isPtrNil covers both, plus the *Nil marker,
	// the three forms the codec spells as one absent slot.
	if isPtrNil(e.Object) {
		return fmt.Sprintf("unexpected object: %s", nilTypeName)
	}
	return fmt.Sprintf("unexpected object: %s", e.Object.ObjectType())
}

func (e *ErrUnexpectedObject) Is(other error) bool {
	_, ok := other.(*ErrUnexpectedObject)
	return ok
}

func init() {
	MustAdd(&ErrUnexpectedObject{})
}
