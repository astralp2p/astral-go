package services

import (
	"fmt"
	"io"

	"github.com/astralp2p/astral-go/astral"
)

// OfferingKey identifies one offering: a provider and a service name.
type OfferingKey struct {
	ProviderID *astral.Identity
	Name       astral.String8
}

func (OfferingKey) ObjectType() string { return "services.offering_key" }

func (k OfferingKey) WriteTo(w io.Writer) (n int64, err error) {
	return astral.Objectify(&k).WriteTo(w)
}

func (k *OfferingKey) ReadFrom(r io.Reader) (n int64, err error) {
	return astral.Objectify(k).ReadFrom(r)
}

func (k OfferingKey) MarshalJSON() ([]byte, error)  { return astral.Objectify(&k).MarshalJSON() }
func (k *OfferingKey) UnmarshalJSON(b []byte) error { return astral.Objectify(k).UnmarshalJSON(b) }

// RemovalCause is why a node retracted offerings from a discovery stream. It
// describes the hop the node itself saw.
type RemovalCause uint8

const (
	CauseMemberLost    RemovalCause = 1 // the carry to a swarm member ended; the node retries the member
	CauseMemberLeft    RemovalCause = 2 // the swarm member left the swarm set; no retry
	CauseMemberRemoved RemovalCause = 3 // the swarm member sent services.removed
	CauseBindingClosed RemovalCause = 4 // the provider's binding or native registration ended
	CauseHidden        RemovalCause = 5 // the caller may no longer see the offering
)

var removalCauseNames = map[RemovalCause]string{
	CauseMemberLost:    "member_lost",
	CauseMemberLeft:    "member_left",
	CauseMemberRemoved: "member_removed",
	CauseBindingClosed: "binding_closed",
	CauseHidden:        "hidden",
}

// String returns the cause's spec name, or "unknown(n)" for a value the spec
// does not define.
func (c RemovalCause) String() string {
	if name, ok := removalCauseNames[c]; ok {
		return name
	}
	return fmt.Sprintf("unknown(%d)", uint8(c))
}

// Removed tells a discovery stream that offerings it had shown left the
// caller's view, and why. Each Removed carries one cause. It is distinct from
// an Update with Available false, which is the provider's own answer.
type Removed struct {
	Offerings []*OfferingKey
	Cause     astral.Uint8 // a RemovalCause
}

func (Removed) ObjectType() string { return "services.removed" }

func (r Removed) WriteTo(w io.Writer) (n int64, err error) {
	return astral.Objectify(&r).WriteTo(w)
}

func (r *Removed) ReadFrom(rd io.Reader) (n int64, err error) {
	return astral.Objectify(r).ReadFrom(rd)
}

func (r Removed) MarshalJSON() ([]byte, error)  { return astral.Objectify(&r).MarshalJSON() }
func (r *Removed) UnmarshalJSON(b []byte) error { return astral.Objectify(r).UnmarshalJSON(b) }

func init() {
	astral.MustAdd(&OfferingKey{})
	astral.MustAdd(&Removed{})
}
