package inference

import (
	"io"

	"github.com/astralp2p/astral-go/astral"
)

// Model describes one model a provider offers.
type Model struct {
	// ID is the name a request names in Model.
	ID          astral.String8
	DisplayName astral.String8
	// ContextWindow is the total token budget of one request, prompt and
	// output together. Zero means the provider states no figure.
	ContextWindow astral.Uint32
	// MaxOutput is the largest output one request may ask for.
	MaxOutput astral.Uint32
	// Modalities lists the input kinds the model accepts, from PartText,
	// PartImage, PartAudio, PartVideo. Empty means text alone.
	Modalities []astral.String8
	// EffortLevels lists the effort settings the model honours, EffortOff
	// among them when it can answer without thinking. Empty means effort is
	// not settable.
	EffortLevels []*EffortLevel
	Streams      astral.Bool
	// Description is free text, for a client to show beside the model.
	Description astral.String8
	// DefaultEffort is the effort level the model uses when a request names
	// none. Empty when the provider does not know.
	DefaultEffort astral.String8
	// Tools reports that the model accepts tool definitions and can answer
	// with tool calls.
	Tools astral.Bool
}

func (Model) ObjectType() string { return "inference.model" }

func (v Model) WriteTo(w io.Writer) (int64, error) { return astral.Objectify(&v).WriteTo(w) }
func (v *Model) ReadFrom(r io.Reader) (int64, error) {
	return astral.Objectify(v).ReadFrom(r)
}
func (v Model) MarshalJSON() ([]byte, error)  { return astral.Objectify(&v).MarshalJSON() }
func (v *Model) UnmarshalJSON(b []byte) error { return astral.Objectify(v).UnmarshalJSON(b) }

func init() { astral.MustAdd(&Model{}) }
