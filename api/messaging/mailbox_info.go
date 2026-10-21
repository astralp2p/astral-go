package messaging

import (
	"encoding/json"
	"io"

	"github.com/astralp2p/astral-go/astral"
)

// MailboxInfo is a participant without its credential: what
// MethodMailbox answers to a caller that does not hold it.
//
// why a second type and not MailboxCredential with the token left empty: a
// caller cannot tell "this participant has no token" from "the token was
// withheld", and a type that sometimes carries a secret is one refactor away
// from carrying it always.
//
// It carries no expiry: a participant may hold several tokens, and their
// lifetimes are apphost's.
type MailboxInfo struct {
	Identity *astral.Identity
	Alias    astral.String8
}

// astral

var _ astral.Object = &MailboxInfo{}

func (i MailboxInfo) ObjectType() string { return "messaging.mailbox_info" }

func (i MailboxInfo) WriteTo(w io.Writer) (n int64, err error) {
	return astral.Objectify(&i).WriteTo(w)
}

func (i *MailboxInfo) ReadFrom(r io.Reader) (n int64, err error) {
	return astral.Objectify(i).ReadFrom(r)
}

// json

func (i MailboxInfo) MarshalJSON() ([]byte, error) {
	type alias MailboxInfo
	return json.Marshal(alias(i))
}

func (i *MailboxInfo) UnmarshalJSON(bytes []byte) error {
	type alias MailboxInfo
	var v alias

	err := json.Unmarshal(bytes, &v)
	if err != nil {
		return err
	}

	*i = MailboxInfo(v)
	return nil
}

func init() {
	astral.MustAdd(&MailboxInfo{})
}
