package apphost

import (
	"io"

	"github.com/astralp2p/astral-go/api/auth"
	"github.com/astralp2p/astral-go/astral"
)

// GrantRequest is what the grant request policy decides on, and what a
// delegated policy sends to its delegate.
//
// Requester is the authenticated identity of the asking session and receives
// the grants. Origin is its web origin, empty for local IPC callers. Permits
// are the node-local grants it asks for.
type GrantRequest struct {
	Requester *astral.Identity
	Origin    astral.String8
	Permits   []*auth.Permit
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

// GrantDecision is the grant request policy's answer. Allow true grants every
// permit the request lists; false refuses them all.
type GrantDecision struct {
	Allow astral.Bool
}

func (GrantDecision) ObjectType() string { return "mod.apphost.grant_decision" }

func (d GrantDecision) WriteTo(w io.Writer) (n int64, err error) {
	return astral.Objectify(&d).WriteTo(w)
}

func (d *GrantDecision) ReadFrom(r io.Reader) (n int64, err error) {
	return astral.Objectify(d).ReadFrom(r)
}

func (d GrantDecision) MarshalJSON() ([]byte, error) {
	return astral.Objectify(&d).MarshalJSON()
}

func (d *GrantDecision) UnmarshalJSON(b []byte) error {
	return astral.Objectify(d).UnmarshalJSON(b)
}

func init() {
	astral.MustAdd(&GrantRequest{})
	astral.MustAdd(&GrantDecision{})
}
