package inference

import (
	"io"

	"github.com/astralp2p/astral-go/astral"
)

// Models is the answer of inference.models, and the object a provider's
// services offering carries in its Info bundle, so a discoverer learns the
// inventory in the same round trip as the offering.
type Models struct {
	Models []*Model
}

func (Models) ObjectType() string { return "inference.models" }

func (v Models) WriteTo(w io.Writer) (int64, error) { return astral.Objectify(&v).WriteTo(w) }
func (v *Models) ReadFrom(r io.Reader) (int64, error) {
	return astral.Objectify(v).ReadFrom(r)
}
func (v Models) MarshalJSON() ([]byte, error)  { return astral.Objectify(&v).MarshalJSON() }
func (v *Models) UnmarshalJSON(b []byte) error { return astral.Objectify(v).UnmarshalJSON(b) }

func init() { astral.MustAdd(&Models{}) }
