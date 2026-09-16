package objects

import (
	"io"

	"github.com/astralp2p/astral-go/astral"
)

// RegistrationLease is the lease a node grants an external provider that
// registers as a searcher, describer or finder. The registration is valid only
// until it expires; the registrant renews it by repeating the registration op
// before then.
//
// The node chooses the lease: it clamps the requested duration to its own
// configured maximum, so Duration is what was granted rather than what was
// asked for, and is never longer.
//
// Duration and ExpiresAt describe the same lease from two vantage points.
// Duration is relative and needs no agreement about the current time, so a
// registrant schedules renewal from it. ExpiresAt is the node's own record of
// when the lease ends.
type RegistrationLease struct {
	Duration  astral.Duration // Granted lease lifetime, after clamping
	ExpiresAt astral.Time     // When the registration expires unless renewed
}

var _ astral.Object = &RegistrationLease{}

func (RegistrationLease) ObjectType() string {
	return "mod.objects.registration_lease"
}

// binary

func (l RegistrationLease) WriteTo(w io.Writer) (int64, error) {
	return astral.Objectify(&l).WriteTo(w)
}

func (l *RegistrationLease) ReadFrom(r io.Reader) (int64, error) {
	return astral.Objectify(l).ReadFrom(r)
}

// json

func (l RegistrationLease) MarshalJSON() ([]byte, error) {
	return astral.Objectify(&l).MarshalJSON()
}

func (l *RegistrationLease) UnmarshalJSON(bytes []byte) error {
	return astral.Objectify(l).UnmarshalJSON(bytes)
}

// ...

func init() {
	astral.MustAdd(&RegistrationLease{})
}
