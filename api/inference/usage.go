package inference

import (
	"io"

	"github.com/astralp2p/astral-go/astral"
)

// Usage reports what one completed request cost. A field is 0 when the server
// did not report it.
type Usage struct {
	PromptTokens     astral.Uint32
	CompletionTokens astral.Uint32
	TotalTokens      astral.Uint32
	// CachedPromptTokens counts the prompt tokens the server read from its
	// cache.
	CachedPromptTokens astral.Uint32
	// ReasoningTokens counts the completion tokens the model spent thinking.
	ReasoningTokens astral.Uint32
}

func (Usage) ObjectType() string { return "inference.usage" }

func (v Usage) WriteTo(w io.Writer) (int64, error) { return astral.Objectify(&v).WriteTo(w) }
func (v *Usage) ReadFrom(r io.Reader) (int64, error) {
	return astral.Objectify(v).ReadFrom(r)
}
func (v Usage) MarshalJSON() ([]byte, error)  { return astral.Objectify(&v).MarshalJSON() }
func (v *Usage) UnmarshalJSON(b []byte) error { return astral.Objectify(v).UnmarshalJSON(b) }

func init() { astral.MustAdd(&Usage{}) }
