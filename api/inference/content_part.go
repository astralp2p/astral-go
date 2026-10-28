package inference

import (
	"io"

	"github.com/astralp2p/astral-go/astral"
)

// ContentPart is one piece of a message: text, or media of one kind.
type ContentPart struct {
	// Kind is PartText, PartImage, PartAudio or PartVideo.
	Kind astral.String8
	// Text is set when Kind is PartText.
	Text astral.String32
	// Media is set when Kind is not PartText.
	Media *MediaRef
}

func (ContentPart) ObjectType() string { return "inference.content_part" }

func (v ContentPart) WriteTo(w io.Writer) (int64, error) { return astral.Objectify(&v).WriteTo(w) }
func (v *ContentPart) ReadFrom(r io.Reader) (int64, error) {
	return astral.Objectify(v).ReadFrom(r)
}
func (v ContentPart) MarshalJSON() ([]byte, error)  { return astral.Objectify(&v).MarshalJSON() }
func (v *ContentPart) UnmarshalJSON(b []byte) error { return astral.Objectify(v).UnmarshalJSON(b) }

func init() { astral.MustAdd(&ContentPart{}) }
