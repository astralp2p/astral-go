package apphost

import (
	"io"

	"github.com/astralp2p/astral-go/api/auth"
	"github.com/astralp2p/astral-go/astral"
)

// GrantPolicyRequest is what the grant request policy decides on, and what a
// delegated policy sends to its delegate, once for each apphost.request_grant
// call.
//
// Requester is the authenticated identity of the asking session and receives
// the grants. Origin is its web origin, empty for local IPC callers. Permits
// are the node-local grants it asks for, with their constraints as sent.
// Duration is the lifetime it asks for each grant; zero asks for grants that
// last until revoked.
type GrantPolicyRequest struct {
	Requester *astral.Identity
	Origin    astral.String8
	Permits   []*auth.Permit
	Duration  astral.Duration
}

func (GrantPolicyRequest) ObjectType() string { return "mod.apphost.grant_policy_request" }

func (r GrantPolicyRequest) WriteTo(w io.Writer) (n int64, err error) {
	return astral.Objectify(&r).WriteTo(w)
}

func (r *GrantPolicyRequest) ReadFrom(rd io.Reader) (n int64, err error) {
	return astral.Objectify(r).ReadFrom(rd)
}

func (r GrantPolicyRequest) MarshalJSON() ([]byte, error) {
	return astral.Objectify(&r).MarshalJSON()
}

func (r *GrantPolicyRequest) UnmarshalJSON(b []byte) error {
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
	astral.MustAdd(&GrantPolicyRequest{})
	astral.MustAdd(&GrantDecision{})
}
