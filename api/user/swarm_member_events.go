package user

import (
	"io"

	"github.com/astralp2p/astral-go/astral"
)

// SwarmMemberJoined is the events.Event data the user module emits locally when
// NodeID enters the local swarm member set.
type SwarmMemberJoined struct {
	NodeID *astral.Identity
}

var _ astral.Object = &SwarmMemberJoined{}

func (SwarmMemberJoined) ObjectType() string { return "mod.user.swarm_member_joined" }

func (e SwarmMemberJoined) WriteTo(w io.Writer) (int64, error) {
	return astral.Objectify(&e).WriteTo(w)
}

func (e *SwarmMemberJoined) ReadFrom(r io.Reader) (int64, error) {
	return astral.Objectify(e).ReadFrom(r)
}

func (e SwarmMemberJoined) MarshalJSON() ([]byte, error) {
	return astral.Objectify(&e).MarshalJSON()
}

func (e *SwarmMemberJoined) UnmarshalJSON(b []byte) error {
	return astral.Objectify(e).UnmarshalJSON(b)
}

// SwarmMemberLeft is the events.Event data the user module emits locally when
// NodeID leaves the local swarm member set.
type SwarmMemberLeft struct {
	NodeID *astral.Identity
}

var _ astral.Object = &SwarmMemberLeft{}

func (SwarmMemberLeft) ObjectType() string { return "mod.user.swarm_member_left" }

func (e SwarmMemberLeft) WriteTo(w io.Writer) (int64, error) {
	return astral.Objectify(&e).WriteTo(w)
}

func (e *SwarmMemberLeft) ReadFrom(r io.Reader) (int64, error) {
	return astral.Objectify(e).ReadFrom(r)
}

func (e SwarmMemberLeft) MarshalJSON() ([]byte, error) {
	return astral.Objectify(&e).MarshalJSON()
}

func (e *SwarmMemberLeft) UnmarshalJSON(b []byte) error {
	return astral.Objectify(e).UnmarshalJSON(b)
}

func init() {
	_ = astral.Add(&SwarmMemberJoined{})
	_ = astral.Add(&SwarmMemberLeft{})
}
