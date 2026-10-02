package services

import (
	"io"

	"github.com/astralp2p/astral-go/astral"
)

// Ask asks a provider, on its advertisement binding, to evaluate one
// registered service for one caller. The provider replies with an Answer
// echoing RequestID.
type Ask struct {
	RequestID astral.Nonce
	CallerID  *astral.Identity
	Service   astral.String8
}

func (Ask) ObjectType() string { return "services.ask" }

func (a Ask) WriteTo(w io.Writer) (n int64, err error) {
	return astral.Objectify(&a).WriteTo(w)
}

func (a *Ask) ReadFrom(r io.Reader) (n int64, err error) {
	return astral.Objectify(a).ReadFrom(r)
}

func init() { astral.MustAdd(&Ask{}) }
