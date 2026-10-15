package apphost

import (
	"errors"
	"fmt"
	"io"

	"github.com/astralp2p/astral-go/api/auth"
	"github.com/astralp2p/astral-go/astral"
)

// AppRegisterRequest is what the app registration policy decides on, and what
// a delegated policy sends to its delegate.
//
// Origin is the caller's web origin, empty for local IPC callers. Name is the
// app's own claim, not verified.
//
// GrantPermits are the permits the app asked to hold as node-local grants;
// ContractPermits are the permits it asked to hold in a signed node→app
// contract, with the origin's trusted-source entitlement already joined.
type AppRegisterRequest struct {
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
// writes exactly the permits listed, on the rail each list names, and records
// the authorizer assignments for the new identity.
type AppRegisterDecision struct {
	Allow           astral.Bool
	GrantPermits    []*auth.Permit
	ContractPermits []*auth.Permit
	// Authorizers are recorded for the new identity. The delegate names the
	// authorizer; the node names the actor, so an entry naming one fails
	// Validate.
	Authorizers []*auth.AuthorizerAssignment
}

// Validate reports a decision the node cannot apply: an authorizer assignment
// that is nil, invalid, or names an actor.
func (d *AppRegisterDecision) Validate() error {
	for _, a := range d.Authorizers {
		switch {
		case a == nil:
			return errors.New("app register decision: nil authorizer assignment")
		case a.Actor != nil && !a.Actor.IsZero():
			return errors.New("app register decision: an assignment names an actor")
		}
		if err := a.Validate(); err != nil {
			return fmt.Errorf("app register decision: %w", err)
		}
	}
	return nil
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
