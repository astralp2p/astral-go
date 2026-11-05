package nearby

import (
	"io"

	"github.com/astralp2p/astral-go/astral"
)

// PublicProfile is a Visible status's identity: the sender's node, its user, and its
// alias.
//
// A decoded NodeID or UserID is nil when its presence byte is 0, and a non-nil zero
// identity when presence byte 1 precedes 33 zero bytes. IsZero reports both.
type PublicProfile struct {
	// NodeID is the sender's node. A zero NodeID has no public key, so no StatusProof
	// verifies under it.
	NodeID *astral.Identity
	// UserID is nil (presence byte 0) when the node has no owner. A receiver reads a
	// zero UserID as no owner.
	UserID *astral.Identity
	// why: string8, as the spec states (protocols/nearby/types/mod.nearby.public_profile.md).
	NodeAlias astral.String8
}

var _ astral.Object = &PublicProfile{}

func (PublicProfile) ObjectType() string {
	return "mod.nearby.public_profile"
}

// binary

func (s PublicProfile) WriteTo(w io.Writer) (int64, error) {
	return astral.Objectify(&s).WriteTo(w)
}

func (s *PublicProfile) ReadFrom(r io.Reader) (int64, error) {
	return astral.Objectify(s).ReadFrom(r)
}

// json

func (s PublicProfile) MarshalJSON() ([]byte, error) {
	return astral.Objectify(&s).MarshalJSON()
}

func (s *PublicProfile) UnmarshalJSON(bytes []byte) error {
	return astral.Objectify(s).UnmarshalJSON(bytes)
}

// ...

func init() {
	_ = astral.Add(&PublicProfile{})
}
