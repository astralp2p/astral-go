package inference

import (
	"io"

	"github.com/astralp2p/astral-go/astral"
)

// ToolCallDelta is one piece of a tool call while it streams.
//
// All deltas of one call carry the same Index. The call's ID and Name arrive
// on its first delta, and its Arguments pieces concatenate in order. A
// non-streaming answer carries every call as a single delta.
type ToolCallDelta struct {
	// Index says which call of this answer the delta belongs to, from 0.
	Index astral.Uint32
	// ID and Name are set on the call's first delta.
	ID   astral.String8
	Name astral.String8
	// Arguments is the next piece of the arguments text.
	Arguments astral.String32
}

func (ToolCallDelta) ObjectType() string { return "inference.tool_call_delta" }

func (v ToolCallDelta) WriteTo(w io.Writer) (int64, error) { return astral.Objectify(&v).WriteTo(w) }
func (v *ToolCallDelta) ReadFrom(r io.Reader) (int64, error) {
	return astral.Objectify(v).ReadFrom(r)
}
func (v ToolCallDelta) MarshalJSON() ([]byte, error) { return astral.Objectify(&v).MarshalJSON() }
func (v *ToolCallDelta) UnmarshalJSON(b []byte) error {
	return astral.Objectify(v).UnmarshalJSON(b)
}

func init() { astral.MustAdd(&ToolCallDelta{}) }
