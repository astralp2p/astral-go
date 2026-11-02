package apphost

import (
	"io"

	"github.com/astralp2p/astral-go/api/auth"
	"github.com/astralp2p/astral-go/astral"
)

// RequestGrantRequest is the grants a session asks apphost.request_grant for.
// The grantee is the identity of the session's access token; the request names
// no identity.
//
// Permits carry their constraints. Duration is the lifetime the session asks
// for each grant; zero asks for grants that last until revoked.
type RequestGrantRequest struct {
	Permits  []*auth.Permit
	Duration astral.Duration
}

func (RequestGrantRequest) ObjectType() string { return "mod.apphost.request_grant_request" }

func (r RequestGrantRequest) WriteTo(w io.Writer) (n int64, err error) {
	return astral.Objectify(&r).WriteTo(w)
}

func (r *RequestGrantRequest) ReadFrom(rd io.Reader) (n int64, err error) {
	return astral.Objectify(r).ReadFrom(rd)
}

func (r RequestGrantRequest) MarshalJSON() ([]byte, error) {
	return astral.Objectify(&r).MarshalJSON()
}

func (r *RequestGrantRequest) UnmarshalJSON(b []byte) error {
	return astral.Objectify(r).UnmarshalJSON(b)
}

func init() {
	astral.MustAdd(&RequestGrantRequest{})
}
