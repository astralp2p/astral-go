package inference

import (
	"io"

	"github.com/astralp2p/astral-go/astral"
)

// Provider says who is offering. A provider's services offering carries it in
// its Info bundle, so a consumer shows a name instead of an identity.
type Provider struct {
	// Name is chosen by the operator, e.g. "ai0".
	Name astral.String8
	// Description is free text.
	Description astral.String16
}

func (Provider) ObjectType() string { return "inference.provider" }

func (v Provider) WriteTo(w io.Writer) (int64, error) { return astral.Objectify(&v).WriteTo(w) }
func (v *Provider) ReadFrom(r io.Reader) (int64, error) {
	return astral.Objectify(v).ReadFrom(r)
}
func (v Provider) MarshalJSON() ([]byte, error)  { return astral.Objectify(&v).MarshalJSON() }
func (v *Provider) UnmarshalJSON(b []byte) error { return astral.Objectify(v).UnmarshalJSON(b) }

func init() { astral.MustAdd(&Provider{}) }
