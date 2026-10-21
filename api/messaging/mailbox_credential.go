package messaging

import (
	"encoding/json"
	"io"

	"github.com/astralp2p/astral-go/astral"
)

// MailboxCredential is a participant as MethodCreateMailbox mints it: a
// node-minted identity, its optional alias, and the apphost access token that
// authenticates it until ExpiresAt.
//
// The token is a credential, and this is the only messaging answer that
// carries it; apphost.list_tokens lists the participant's tokens. MailboxInfo
// is the same participant without it.
type MailboxCredential struct {
	Identity  *astral.Identity
	Alias     astral.String8
	Token     astral.String8
	ExpiresAt astral.Time
}

// astral

var _ astral.Object = &MailboxCredential{}

func (c MailboxCredential) ObjectType() string { return "messaging.mailbox_credential" }

func (c MailboxCredential) WriteTo(w io.Writer) (n int64, err error) {
	return astral.Objectify(&c).WriteTo(w)
}

func (c *MailboxCredential) ReadFrom(r io.Reader) (n int64, err error) {
	return astral.Objectify(c).ReadFrom(r)
}

// json

func (c MailboxCredential) MarshalJSON() ([]byte, error) {
	type alias MailboxCredential
	return json.Marshal(alias(c))
}

func (c *MailboxCredential) UnmarshalJSON(bytes []byte) error {
	type alias MailboxCredential
	var v alias

	err := json.Unmarshal(bytes, &v)
	if err != nil {
		return err
	}

	*c = MailboxCredential(v)
	return nil
}

func init() {
	astral.MustAdd(&MailboxCredential{})
}
