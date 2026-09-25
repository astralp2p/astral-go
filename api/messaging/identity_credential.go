package messaging

import (
	"encoding/json"
	"io"

	"github.com/astralp2p/astral-go/astral"
)

// IdentityCredential is a participant as MethodCreateIdentity mints it: a
// node-minted identity, its optional alias, and the apphost access token that
// authenticates it until ExpiresAt.
//
// The token is a credential, and this is the only messaging answer that
// carries it; apphost.list_tokens lists the participant's tokens. IdentityInfo
// is the same participant without it.
type IdentityCredential struct {
	Identity  *astral.Identity
	Alias     astral.String8
	Token     astral.String8
	ExpiresAt astral.Time
}

// astral

var _ astral.Object = &IdentityCredential{}

func (c IdentityCredential) ObjectType() string { return "messaging.identity_credential" }

func (c IdentityCredential) WriteTo(w io.Writer) (n int64, err error) {
	return astral.Objectify(&c).WriteTo(w)
}

func (c *IdentityCredential) ReadFrom(r io.Reader) (n int64, err error) {
	return astral.Objectify(c).ReadFrom(r)
}

// json

func (c IdentityCredential) MarshalJSON() ([]byte, error) {
	type alias IdentityCredential
	return json.Marshal(alias(c))
}

func (c *IdentityCredential) UnmarshalJSON(bytes []byte) error {
	type alias IdentityCredential
	var v alias

	err := json.Unmarshal(bytes, &v)
	if err != nil {
		return err
	}

	*c = IdentityCredential(v)
	return nil
}

func init() {
	astral.MustAdd(&IdentityCredential{})
}
