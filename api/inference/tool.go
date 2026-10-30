package inference

import (
	"io"

	"github.com/astralp2p/astral-go/astral"
)

// Tool is a function the model may call.
type Tool struct {
	Name        astral.String8
	Description astral.String16
	// Parameters is a JSON Schema object, as text.
	Parameters astral.String32
}

func (Tool) ObjectType() string { return "inference.tool" }

func (v Tool) WriteTo(w io.Writer) (int64, error) { return astral.Objectify(&v).WriteTo(w) }
func (v *Tool) ReadFrom(r io.Reader) (int64, error) {
	return astral.Objectify(v).ReadFrom(r)
}
func (v Tool) MarshalJSON() ([]byte, error)  { return astral.Objectify(&v).MarshalJSON() }
func (v *Tool) UnmarshalJSON(b []byte) error { return astral.Objectify(v).UnmarshalJSON(b) }

func init() { astral.MustAdd(&Tool{}) }
