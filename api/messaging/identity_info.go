package messaging

import (
	"encoding/json"
	"io"

	"github.com/astralp2p/astral-go/astral"
)

// IdentityInfo is a participant without its credential: what
// MethodIdentity answers to a caller that does not hold it.
//
// why a second type and not IdentityCredential with the token left empty: a
// caller cannot tell "this participant has no token" from "the token was
// withheld", and a type that sometimes carries a secret is one refactor away
// from carrying it always.
//
// It carries no expiry: a participant may hold several tokens, and their
// lifetimes are apphost's.
type IdentityInfo struct {
	Identity *astral.Identity
	Alias    astral.String8
}

// astral

var _ astral.Object = &IdentityInfo{}

func (i IdentityInfo) ObjectType() string { return "messaging.identity_info" }

func (i IdentityInfo) WriteTo(w io.Writer) (n int64, err error) {
	return astral.Objectify(&i).WriteTo(w)
}

func (i *IdentityInfo) ReadFrom(r io.Reader) (n int64, err error) {
	return astral.Objectify(i).ReadFrom(r)
}

// json

func (i IdentityInfo) MarshalJSON() ([]byte, error) {
	type alias IdentityInfo
	return json.Marshal(alias(i))
}

func (i *IdentityInfo) UnmarshalJSON(bytes []byte) error {
	type alias IdentityInfo
	var v alias

	err := json.Unmarshal(bytes, &v)
	if err != nil {
		return err
	}

	*i = IdentityInfo(v)
	return nil
}

func init() {
	astral.MustAdd(&IdentityInfo{})
}
