package apphost

import (
	"io"

	"github.com/astralp2p/astral-go/api/auth"
	"github.com/astralp2p/astral-go/astral"
)

// Grant is one node-local grant, as apphost.list_grants reports it. A grant
// authorizes its permit on one node and travels nowhere.
//
// ID names the grant to apphost.revoke; the node never assigns the ID of a
// deleted grant to another grant. Permit is the recorded permit, with its
// constraints as written. ExpiresAt is the instant the grant lapses; nil means
// the grant lasts until revoked.
type Grant struct {
	ID        astral.Uint64
	Permit    *auth.Permit
	ExpiresAt *astral.Time
}

func (Grant) ObjectType() string { return "mod.apphost.grant" }

func (g Grant) WriteTo(w io.Writer) (n int64, err error) {
	return astral.Objectify(&g).WriteTo(w)
}

func (g *Grant) ReadFrom(r io.Reader) (n int64, err error) {
	return astral.Objectify(g).ReadFrom(r)
}

func (g Grant) MarshalJSON() ([]byte, error) {
	return astral.Objectify(&g).MarshalJSON()
}

func (g *Grant) UnmarshalJSON(b []byte) error {
	return astral.Objectify(g).UnmarshalJSON(b)
}

func init() {
	astral.MustAdd(&Grant{})
}
