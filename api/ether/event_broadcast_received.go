package ether

import (
	"io"

	"github.com/astralp2p/astral-go/api/ip"
	"github.com/astralp2p/astral-go/astral"
)

var _ astral.Object = &EventBroadcastReceived{}

type EventBroadcastReceived struct {
	SourceIP ip.IP
	Object   astral.Object
}

// astral

func (EventBroadcastReceived) ObjectType() string {
	return "mod.ether.events.broadcast_received"
}

func (e EventBroadcastReceived) WriteTo(w io.Writer) (n int64, err error) {
	return astral.Objectify(&e).WriteTo(w)
}

func (e *EventBroadcastReceived) ReadFrom(r io.Reader) (n int64, err error) {
	return astral.Objectify(e).ReadFrom(r)
}

// json

// MarshalJSON writes Object as a nested Type/Object container, the form of an
// interface-typed slot (.ai/system/topics/json-encoding.md).
func (e EventBroadcastReceived) MarshalJSON() ([]byte, error) {
	return astral.Objectify(&e).MarshalJSON()
}

func (e *EventBroadcastReceived) UnmarshalJSON(data []byte) error {
	return astral.Objectify(e).UnmarshalJSON(data)
}

func init() {
	_ = astral.Add(&EventBroadcastReceived{})
}
