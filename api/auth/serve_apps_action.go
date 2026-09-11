package auth

import (
	"io"

	"github.com/astralp2p/astral-go/astral"
)

// ServeAppsAction requests permission for Actor to host on the node: install an
// app handler that receives queries addressed to Actor, and publish Actor's
// service advertisement.
//
// The authority is a place on the node, not access to data. It grants no right
// to remove another identity's handlers or advertisements.
type ServeAppsAction struct {
	Action
}

func (ServeAppsAction) ObjectType() string { return "mod.auth.serve_apps_action" }

func (a ServeAppsAction) WriteTo(w io.Writer) (n int64, err error) {
	return astral.Objectify(&a).WriteTo(w)
}

func (a *ServeAppsAction) ReadFrom(r io.Reader) (n int64, err error) {
	return astral.Objectify(a).ReadFrom(r)
}

// ApplyConstraints refuses a permit that carries any constraint. This action
// does not evaluate constraints, and an action that does not evaluate them is
// permitted regardless of them — so a permit narrowed by its issuer would be
// honoured in full. Refusing is the bar that keeps the deferral safe until
// constraints are implemented.
func (a ServeAppsAction) ApplyConstraints(cs *astral.Bundle) bool {
	return cs == nil || len(cs.Objects()) == 0
}

func init() { astral.MustAdd(&ServeAppsAction{}) }
