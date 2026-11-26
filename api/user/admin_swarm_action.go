package user

import (
	"io"

	"github.com/astralp2p/astral-go/api/auth"
	"github.com/astralp2p/astral-go/astral"
)

// AdminSwarmAction requests permission for Actor to change what the swarm is:
// which nodes belong to it. Expelling is its own ExpelAction, because a
// PrivilegedMembership does not cover it (D7).
//
// Subject declares the noun a call touches, the node a call introduces.
// Nothing evaluates it yet; it is recorded so a constraint can bind to it
// later without changing the action type or any of its call sites. It is left
// unset by an op that names no node.
type AdminSwarmAction struct {
	auth.Action
	Subject *astral.Identity
}

func (AdminSwarmAction) ObjectType() string { return "mod.user.admin_swarm_action" }

func (a AdminSwarmAction) WriteTo(w io.Writer) (n int64, err error) {
	return astral.Objectify(&a).WriteTo(w)
}

func (a *AdminSwarmAction) ReadFrom(r io.Reader) (n int64, err error) {
	return astral.Objectify(a).ReadFrom(r)
}

// ApplyConstraints refuses a permit that carries any constraint. This action
// does not evaluate constraints, and an action that does not evaluate them is
// permitted regardless of them — so a permit narrowed by its issuer would be
// honoured in full. Refusing is the bar that keeps the deferral safe until
// constraints are implemented.
func (a AdminSwarmAction) ApplyConstraints(cs *astral.Bundle) bool {
	return cs == nil || len(cs.Objects()) == 0
}

func init() { astral.MustAdd(&AdminSwarmAction{}) }
