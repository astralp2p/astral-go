package auth

import (
	"io"

	"github.com/astralp2p/astral-go/astral"
)

// AdminManageAppsAction requests permission for Actor to administer app and
// agent credentials on the node: create, list and delete apphost access tokens
// and MCP agents.
//
// Listing is administration here, not a separate read tier: a list hands out
// the bearer credentials it enumerates.
type AdminManageAppsAction struct {
	Action
}

func (AdminManageAppsAction) ObjectType() string { return "mod.auth.admin_manage_apps_action" }

func (a AdminManageAppsAction) WriteTo(w io.Writer) (n int64, err error) {
	return astral.Objectify(&a).WriteTo(w)
}

func (a *AdminManageAppsAction) ReadFrom(r io.Reader) (n int64, err error) {
	return astral.Objectify(a).ReadFrom(r)
}

// ApplyConstraints refuses a permit that carries any constraint. This action
// does not evaluate constraints, and an action that does not evaluate them is
// permitted regardless of them — so a permit narrowed by its issuer would be
// honoured in full. Refusing is the bar that keeps the deferral safe until
// constraints are implemented.
func (a AdminManageAppsAction) ApplyConstraints(cs *astral.Bundle) bool {
	return cs == nil || len(cs.Objects()) == 0
}

func init() { astral.MustAdd(&AdminManageAppsAction{}) }
