package inference

import (
	"io"

	"github.com/astralp2p/astral-go/astral"
)

// EffortLevel is one effort setting a model honours.
//
// why a named level and not a raw reasoning_effort string: the level is what a
// consumer chooses between, and it must be discoverable per model. A provider
// that maps it onto reasoning_effort, a token cap, or nothing at all does so on
// its far side; the consumer only needs Name, meaning and cost.
type EffortLevel struct {
	// Name is what a request names in Effort. EffortOff means the model does
	// not think.
	Name        astral.String8
	Description astral.String8
	// TokenBudget hints how much output the level is expected to spend. Zero
	// means the provider states no figure.
	TokenBudget astral.Uint32
}

func (EffortLevel) ObjectType() string { return "inference.effort_level" }

func (v EffortLevel) WriteTo(w io.Writer) (int64, error) { return astral.Objectify(&v).WriteTo(w) }
func (v *EffortLevel) ReadFrom(r io.Reader) (int64, error) {
	return astral.Objectify(v).ReadFrom(r)
}
func (v EffortLevel) MarshalJSON() ([]byte, error)  { return astral.Objectify(&v).MarshalJSON() }
func (v *EffortLevel) UnmarshalJSON(b []byte) error { return astral.Objectify(v).UnmarshalJSON(b) }

func init() { astral.MustAdd(&EffortLevel{}) }
