package inference

import (
	"io"

	"github.com/astralp2p/astral-go/astral"
)

// Request is the single object a caller sends on inference.complete.
type Request struct {
	Model astral.String8
	// Effort names one of the model's EffortLevels. Empty takes the model's
	// DefaultEffort.
	Effort astral.String8
	// MaxOutput caps this request's output tokens. Zero takes the model's
	// MaxOutput.
	MaxOutput astral.Uint32
	// Temperature is a pointer so "unset" and "0" differ.
	Temperature *astral.Float32
	Messages    []*Message
	// System is a convenience for the common single system prompt; a leading
	// Message with RoleSystem is equivalent. Empty carries none.
	System astral.String32
	// Tools lists the functions the model may call. Empty offers none.
	Tools []*Tool
	// ToolChoice is empty or ToolChoiceAuto to let the model decide,
	// ToolChoiceNone, ToolChoiceRequired, or the Name of the one tool the
	// model must call.
	ToolChoice astral.String8
	// Stream asks for deltas. False asks for the whole answer in one chunk.
	Stream astral.Bool
}

func (Request) ObjectType() string { return "inference.request" }

func (v Request) WriteTo(w io.Writer) (int64, error) { return astral.Objectify(&v).WriteTo(w) }
func (v *Request) ReadFrom(r io.Reader) (int64, error) {
	return astral.Objectify(v).ReadFrom(r)
}
func (v Request) MarshalJSON() ([]byte, error)  { return astral.Objectify(&v).MarshalJSON() }
func (v *Request) UnmarshalJSON(b []byte) error { return astral.Objectify(v).UnmarshalJSON(b) }

func init() { astral.MustAdd(&Request{}) }
