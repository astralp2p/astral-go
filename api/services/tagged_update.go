package services

import (
	"io"

	"github.com/astralp2p/astral-go/astral"
)

// TaggedUpdate is an offering a swarm member sends to the node carrying a
// discovery that asked for tags. Open is the provider's effective visibility
// on the member; the carrying node filters by it and forwards only Update.
type TaggedUpdate struct {
	Open   astral.Bool
	Update *Update
}

func (TaggedUpdate) ObjectType() string { return "services.tagged_update" }

func (t TaggedUpdate) WriteTo(w io.Writer) (int64, error) {
	return astral.Objectify(&t).WriteTo(w)
}

func (t *TaggedUpdate) ReadFrom(r io.Reader) (int64, error) {
	return astral.Objectify(t).ReadFrom(r)
}

func (t TaggedUpdate) MarshalJSON() ([]byte, error)  { return astral.Objectify(&t).MarshalJSON() }
func (t *TaggedUpdate) UnmarshalJSON(b []byte) error { return astral.Objectify(t).UnmarshalJSON(b) }

func init() { astral.MustAdd(&TaggedUpdate{}) }
