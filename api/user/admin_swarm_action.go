package user

import (
	"io"

	"github.com/astralp2p/astral-go/api/auth"
	"github.com/astralp2p/astral-go/astral"
)

// AdminSwarmAction requests permission for Actor to change what the swarm is:
// which nodes belong to it, and which objects it carries.
//
// Adopting, expelling, adding an asset and removing one are one action because
// an issuer granting control over the swarm's membership would not expect its
// asset set to be governed separately. Both decide what the swarm holds, and
// both propagate from this node to every sibling.
//
// Subject and ObjectID declare the nouns a call touches — Subject the node a
// call adopts or expels, ObjectID the asset it adds or removes. Nothing
// evaluates them yet; they are recorded so a constraint can bind to them later
// without changing the action type or any of its call sites. Each is left unset
// by an op that names neither.
//
// It replaces AdoptAction and ExpelAction, whose Subject it carries unchanged.
type AdminSwarmAction struct {
	auth.Action
	Subject  *astral.Identity
	ObjectID *astral.ObjectID
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
