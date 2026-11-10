package services

import (
	"io"

	"github.com/astralp2p/astral-go/astral"
)

// NotGranted opens a discovery stream that names services when the caller
// holds no discovery permit for some of them on at least one contributing
// node. Private offerings of those services there are filtered; open ones
// still arrive.
type NotGranted struct {
	Services []astral.String8
}

func (NotGranted) ObjectType() string { return "services.not_granted" }

func (n NotGranted) WriteTo(w io.Writer) (int64, error) {
	return astral.Objectify(&n).WriteTo(w)
}

func (n *NotGranted) ReadFrom(r io.Reader) (int64, error) {
	return astral.Objectify(n).ReadFrom(r)
}

func (n NotGranted) MarshalJSON() ([]byte, error)  { return astral.Objectify(&n).MarshalJSON() }
func (n *NotGranted) UnmarshalJSON(b []byte) error { return astral.Objectify(n).UnmarshalJSON(b) }

func init() { astral.MustAdd(&NotGranted{}) }
