package inference

import (
	"io"

	"github.com/astralp2p/astral-go/astral"
)

// MediaRef names where one media part's bytes come from.
//
// why three sources: inline bytes suit a small picture a client already holds;
// an object ID suits media that is already addressable on the network, so a
// provider reads it once through objects.read instead of the client re-sending
// it on every call; a URL suits a provider allowed to fetch from the open web.
type MediaRef struct {
	// Kind is SourceInline, SourceObject or SourceURL.
	Kind astral.String8
	// Data carries the bytes when Kind is SourceInline.
	//
	// why astral.Bytes32 and not []byte: a blueprint is derived from the Go
	// type, and a bare byte slice derives nothing, so the node could not
	// decode this type. Bytes16 capped a picture at 64 KiB.
	Data astral.Bytes32
	// ObjectID names the object when Kind is SourceObject.
	ObjectID astral.ObjectID
	// URL is the address when Kind is SourceURL.
	URL astral.String16
	// MIME is the media type, e.g. "image/png". Required for inline data,
	// optional otherwise.
	MIME astral.String8
}

func (MediaRef) ObjectType() string { return "inference.media_ref" }

func (v MediaRef) WriteTo(w io.Writer) (int64, error) { return astral.Objectify(&v).WriteTo(w) }
func (v *MediaRef) ReadFrom(r io.Reader) (int64, error) {
	return astral.Objectify(v).ReadFrom(r)
}
func (v MediaRef) MarshalJSON() ([]byte, error)  { return astral.Objectify(&v).MarshalJSON() }
func (v *MediaRef) UnmarshalJSON(b []byte) error { return astral.Objectify(v).UnmarshalJSON(b) }

func init() { astral.MustAdd(&MediaRef{}) }
