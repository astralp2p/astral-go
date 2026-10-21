package events

import (
	"io"

	"github.com/astralp2p/astral-go/astral"
)

type Event struct {
	ID        astral.Nonce
	SourceID  *astral.Identity
	Timestamp astral.Time
	Data      astral.Object
}

func (Event) ObjectType() string {
	return "mod.events.event"
}

func (e Event) WriteTo(w io.Writer) (n int64, err error) {
	return astral.Objectify(&e).WriteTo(w)
}

func (e *Event) ReadFrom(r io.Reader) (n int64, err error) {
	return astral.Objectify(e).ReadFrom(r)
}

// MarshalJSON writes Data as a nested Type/Object container, the form of an
// interface-typed slot (.ai/system/topics/json-encoding.md).
func (e Event) MarshalJSON() ([]byte, error)     { return astral.Objectify(&e).MarshalJSON() }
func (e *Event) UnmarshalJSON(data []byte) error { return astral.Objectify(e).UnmarshalJSON(data) }

func init() {
	_ = astral.Add(&Event{})
}
