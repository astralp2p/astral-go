package services

import (
	"io"

	"github.com/astralp2p/astral-go/astral"
)

// OperationsList lists the operations a provider exposes to one caller. It is
// carried in Update.Info. Listing an operation grants nothing.
type OperationsList struct {
	Operations []astral.String8
}

func (OperationsList) ObjectType() string { return "services.operations_list" }

func (l OperationsList) WriteTo(w io.Writer) (n int64, err error) {
	return astral.Objectify(&l).WriteTo(w)
}

func (l *OperationsList) ReadFrom(r io.Reader) (n int64, err error) {
	return astral.Objectify(l).ReadFrom(r)
}

func init() { astral.MustAdd(&OperationsList{}) }
