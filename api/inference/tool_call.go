package inference

import (
	"io"

	"github.com/astralp2p/astral-go/astral"
)

// ToolCall is one call the model made, as it appears in history.
type ToolCall struct {
	// ID is unique within the answer. The tool turn that answers the call
	// names it in ToolCallID.
	ID   astral.String8
	Name astral.String8
	// Arguments is a JSON object, as text.
	Arguments astral.String32
}

func (ToolCall) ObjectType() string { return "inference.tool_call" }

func (v ToolCall) WriteTo(w io.Writer) (int64, error) { return astral.Objectify(&v).WriteTo(w) }
func (v *ToolCall) ReadFrom(r io.Reader) (int64, error) {
	return astral.Objectify(v).ReadFrom(r)
}
func (v ToolCall) MarshalJSON() ([]byte, error)  { return astral.Objectify(&v).MarshalJSON() }
func (v *ToolCall) UnmarshalJSON(b []byte) error { return astral.Objectify(v).UnmarshalJSON(b) }

func init() { astral.MustAdd(&ToolCall{}) }
