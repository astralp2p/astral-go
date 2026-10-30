package inference

import (
	"io"

	"github.com/astralp2p/astral-go/astral"
)

// Chunk is one delta of an answer.
//
// Index orders the chunks of one answer, starting at 0 and increasing by one.
// Reasoning, Text and ToolCalls may share a chunk. A non-empty FinishReason
// marks the last chunk; the usage object, when the provider reports one,
// follows it, and eos ends the answer.
type Chunk struct {
	Index astral.Uint32
	// Text is the next piece of the answer.
	Text astral.String32
	// Reasoning is the model's thinking when the provider exposes it. It is
	// not the answer: a consumer that renders only answers ignores it, and one
	// that shows the thinking renders it before the text it precedes.
	Reasoning astral.String32
	// ToolCalls carries the next pieces of the answer's tool calls.
	ToolCalls []*ToolCallDelta
	// FinishReason is empty while the answer continues. On the last chunk it
	// is the server's own reason: FinishStop, FinishLength, FinishToolCalls,
	// FinishContentFilter, or another.
	FinishReason astral.String8
	// Model echoes the model that answered, so a client that offered a choice
	// learns which one ran. Set on the first chunk.
	Model astral.String8
}

func (Chunk) ObjectType() string { return "inference.chunk" }

func (v Chunk) WriteTo(w io.Writer) (int64, error) { return astral.Objectify(&v).WriteTo(w) }
func (v *Chunk) ReadFrom(r io.Reader) (int64, error) {
	return astral.Objectify(v).ReadFrom(r)
}
func (v Chunk) MarshalJSON() ([]byte, error)  { return astral.Objectify(&v).MarshalJSON() }
func (v *Chunk) UnmarshalJSON(b []byte) error { return astral.Objectify(v).UnmarshalJSON(b) }

func init() { astral.MustAdd(&Chunk{}) }
