package services

import (
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

// Removed tells a discovery stream that offerings it had shown are no longer
// valid because their provider was lost. It is distinct from an Update with
// Available false, which is the provider's own answer.
type Removed struct {
	Offerings []*OfferingKey
}

func (Removed) ObjectType() string { return "services.removed" }

func (r Removed) WriteTo(w io.Writer) (n int64, err error) {
	return astral.Objectify(&r).WriteTo(w)
}

func (r *Removed) ReadFrom(rd io.Reader) (n int64, err error) {
	return astral.Objectify(r).ReadFrom(rd)
}

func init() {
	astral.MustAdd(&OfferingKey{})
	astral.MustAdd(&Removed{})
}
