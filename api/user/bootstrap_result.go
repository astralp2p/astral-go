package user

import (
	"io"

	"github.com/astralp2p/astral-go/astral"
)

// BootstrapResult is the reply of user.bootstrap: the user that now owns the
// node and, when the node generated them, the words that derive that user.
//
// Mnemonic is empty when the caller sent the words.
type BootstrapResult struct {
	Identity *astral.Identity
	Mnemonic astral.String16
}

func (BootstrapResult) ObjectType() string { return "mod.user.bootstrap_result" }

func (r BootstrapResult) WriteTo(w io.Writer) (n int64, err error) {
	return astral.Objectify(&r).WriteTo(w)
}

func (r *BootstrapResult) ReadFrom(rd io.Reader) (n int64, err error) {
	return astral.Objectify(r).ReadFrom(rd)
}

func init() {
	astral.MustAdd(&BootstrapResult{})
}
