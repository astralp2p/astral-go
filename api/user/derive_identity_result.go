package user

import (
	"io"

	"github.com/astralp2p/astral-go/astral"
)

// DeriveIdentityResult is the reply of user.derive_identity: the user that the
// words derive and, when the node generated them, the words.
//
// Mnemonic is empty when the caller sent the words.
type DeriveIdentityResult struct {
	Identity *astral.Identity
	Mnemonic astral.String16
}

func (DeriveIdentityResult) ObjectType() string { return "mod.user.derive_identity_result" }

func (r DeriveIdentityResult) WriteTo(w io.Writer) (n int64, err error) {
	return astral.Objectify(&r).WriteTo(w)
}

func (r *DeriveIdentityResult) ReadFrom(rd io.Reader) (n int64, err error) {
	return astral.Objectify(r).ReadFrom(rd)
}

func init() {
	astral.MustAdd(&DeriveIdentityResult{})
}
