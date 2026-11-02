package apphost

import (
	"io"

	"github.com/astralp2p/astral-go/api/auth"
	"github.com/astralp2p/astral-go/astral"
)

// GrantRequest is the grants a caller asks apphost.grant to write for one
// identity. Each permit becomes one node-local grant.
//
// Identity receives the grants. Duration is the lifetime of each grant; zero
// writes grants that last until revoked. Permits carry their constraints.
type GrantRequest struct {
	Identity *astral.Identity
	Duration astral.Duration
	Permits  []*auth.Permit
}

func (GrantRequest) ObjectType() string { return "mod.apphost.grant_request" }

func (r GrantRequest) WriteTo(w io.Writer) (n int64, err error) {
	return astral.Objectify(&r).WriteTo(w)
}

func (r *GrantRequest) ReadFrom(rd io.Reader) (n int64, err error) {
	return astral.Objectify(r).ReadFrom(rd)
}

func (r GrantRequest) MarshalJSON() ([]byte, error) {
	return astral.Objectify(&r).MarshalJSON()
}

func (r *GrantRequest) UnmarshalJSON(b []byte) error {
	return astral.Objectify(r).UnmarshalJSON(b)
}

func init() {
	astral.MustAdd(&GrantRequest{})
}
