package nodes

import (
	"io"

	"github.com/astralp2p/astral-go/astral"
)

// EndpointsChangedMessage is pushed by a node to its linked local swarm members
// after its own set of local addresses changed. It asks the receiver to pull the
// endpoints of Subject from the sender; it carries no endpoints.
type EndpointsChangedMessage struct {
	Subject *astral.Identity
}

var _ astral.Object = &EndpointsChangedMessage{}

func (EndpointsChangedMessage) ObjectType() string { return "mod.nodes.endpoints_changed_message" }

func (m EndpointsChangedMessage) WriteTo(w io.Writer) (int64, error) {
	return astral.Objectify(&m).WriteTo(w)
}

func (m *EndpointsChangedMessage) ReadFrom(r io.Reader) (int64, error) {
	return astral.Objectify(m).ReadFrom(r)
}

func (m EndpointsChangedMessage) MarshalJSON() ([]byte, error) {
	return astral.Objectify(&m).MarshalJSON()
}

func (m *EndpointsChangedMessage) UnmarshalJSON(b []byte) error {
	return astral.Objectify(m).UnmarshalJSON(b)
}

// EndpointsUpdatedEvent is the events.Event data the nodes module emits locally
// when a pull that an EndpointsChangedMessage started stored at least one new
// endpoint of Identity.
type EndpointsUpdatedEvent struct {
	Identity *astral.Identity
}

var _ astral.Object = &EndpointsUpdatedEvent{}

func (EndpointsUpdatedEvent) ObjectType() string { return "mod.nodes.endpoints_updated_event" }

func (e EndpointsUpdatedEvent) WriteTo(w io.Writer) (int64, error) {
	return astral.Objectify(&e).WriteTo(w)
}

func (e *EndpointsUpdatedEvent) ReadFrom(r io.Reader) (int64, error) {
	return astral.Objectify(e).ReadFrom(r)
}

func (e EndpointsUpdatedEvent) MarshalJSON() ([]byte, error) {
	return astral.Objectify(&e).MarshalJSON()
}

func (e *EndpointsUpdatedEvent) UnmarshalJSON(b []byte) error {
	return astral.Objectify(e).UnmarshalJSON(b)
}

func init() {
	_ = astral.Add(&EndpointsChangedMessage{})
	_ = astral.Add(&EndpointsUpdatedEvent{})
}
