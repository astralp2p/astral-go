package apphost

import (
	"io"

	"github.com/astralp2p/astral-go/api/auth"
	"github.com/astralp2p/astral-go/astral"
)

// AppRegisterRequest is what the app registration policy decides on, and what
// a delegated policy sends to its delegate.
//
// Caller is the identity that called apphost.register. Anonymous is true when
// the caller's apphost session presented no token; the router then substitutes
// the node's identity for the missing caller. Origin is the caller's web origin,
// empty for local IPC callers. Name is the app's own claim, not verified.
//
// GrantPermits are the permits the app asked to hold as node-local grants;
// ContractPermits are the permits it asked to hold in a signed node→app
// contract, with the origin's trusted-source entitlement already joined.
type AppRegisterRequest struct {
	Caller          *astral.Identity
	Anonymous       astral.Bool
	Origin          astral.String8
	Name            astral.String8
	GrantPermits    []*auth.Permit
	ContractPermits []*auth.Permit
}

func (AppRegisterRequest) ObjectType() string { return "mod.apphost.app_register_request" }

func (r AppRegisterRequest) WriteTo(w io.Writer) (n int64, err error) {
	return astral.Objectify(&r).WriteTo(w)
}

func (r *AppRegisterRequest) ReadFrom(rd io.Reader) (n int64, err error) {
	return astral.Objectify(r).ReadFrom(rd)
}

// AppRegisterDecision is the app registration policy's answer. Allow false
// refuses the registration and the other fields are ignored. Otherwise the node
// writes exactly the permits listed, on the rail each list names, and assigns
// each authorizer to the new app.
//
// Authorizers leave Actor nil: the app's identity is generated after the
// decision, and registration fills it in.
type AppRegisterDecision struct {
	Allow           astral.Bool
	GrantPermits    []*auth.Permit
	ContractPermits []*auth.Permit
	Authorizers     []*auth.AuthorizerAssignment
}

func (AppRegisterDecision) ObjectType() string { return "mod.apphost.app_register_decision" }

func (d AppRegisterDecision) WriteTo(w io.Writer) (n int64, err error) {
	return astral.Objectify(&d).WriteTo(w)
}

func (d *AppRegisterDecision) ReadFrom(r io.Reader) (n int64, err error) {
	return astral.Objectify(d).ReadFrom(r)
}

func init() {
	astral.MustAdd(&AppRegisterRequest{})
	astral.MustAdd(&AppRegisterDecision{})
}
