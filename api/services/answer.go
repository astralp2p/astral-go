package services

import (
	"io"

	"github.com/astralp2p/astral-go/astral"
)

// Answer is a provider's reply to one Ask: the complete offering of the asked
// service for the asked caller. The provider leaves Update.ProviderID empty;
// the node sets it from the binding.
type Answer struct {
	RequestID astral.Nonce
	Update    *Update
}

func (Answer) ObjectType() string { return "services.answer" }

func (a Answer) WriteTo(w io.Writer) (n int64, err error) {
	return astral.Objectify(&a).WriteTo(w)
}

func (a *Answer) ReadFrom(r io.Reader) (n int64, err error) {
	return astral.Objectify(a).ReadFrom(r)
}

func init() { astral.MustAdd(&Answer{}) }
