package user

import (
	"io"

	"github.com/astralp2p/astral-go/astral"
)

// Mnemonic holds the BIP-39 words that derive the node's user, kept beside the
// key so the user can read them back later for a backup.
//
// why: the passphrase is never part of it; it stays with the user alone.
type Mnemonic struct {
	Words astral.String16
}

func (Mnemonic) ObjectType() string { return "mod.user.mnemonic" }

func (m Mnemonic) WriteTo(w io.Writer) (n int64, err error) {
	return astral.Objectify(&m).WriteTo(w)
}

func (m *Mnemonic) ReadFrom(r io.Reader) (n int64, err error) {
	return astral.Objectify(m).ReadFrom(r)
}

func init() {
	astral.MustAdd(&Mnemonic{})
}
