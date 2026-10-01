package services

import (
	"io"

	"github.com/astralp2p/astral-go/astral"
)

// Incomplete precedes the eos that ends the initial attempt of a discovery when
// some initial work did not resolve. Services names requested services, never
// providers or nodes. A bare eos means the initial attempt completed.
type Incomplete struct {
	Services []astral.String8
}

func (Incomplete) ObjectType() string { return "services.incomplete" }

func (i Incomplete) WriteTo(w io.Writer) (n int64, err error) {
	return astral.Objectify(&i).WriteTo(w)
}

func (i *Incomplete) ReadFrom(r io.Reader) (n int64, err error) {
	return astral.Objectify(i).ReadFrom(r)
}

func init() { astral.MustAdd(&Incomplete{}) }
