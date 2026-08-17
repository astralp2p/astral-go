package auth

import (
	"io"

	"github.com/astralp2p/astral-go/astral"
)

// AdminObjectsAction requests permission for Actor to modify the node's
// repositories: which repositories it has, and what they hold.
//
// ObjectID, Repo and Path declare the nouns a call touches. Nothing evaluates
// them yet — they are recorded so a constraint can bind to them later without
// changing the action type or any of its call sites. Each is left unset by an
// op that names no single object, repository, or path.
type AdminObjectsAction struct {
	Action
	ObjectID *astral.ObjectID
	Repo     astral.String8
	Path     astral.String8
}

func (AdminObjectsAction) ObjectType() string { return "mod.auth.admin_objects_action" }

func (a AdminObjectsAction) WriteTo(w io.Writer) (n int64, err error) {
	return astral.Objectify(&a).WriteTo(w)
}

func (a *AdminObjectsAction) ReadFrom(r io.Reader) (n int64, err error) {
	return astral.Objectify(a).ReadFrom(r)
}

// ApplyConstraints refuses a permit that carries any constraint. This action
// does not evaluate constraints, and an action that does not evaluate them is
// permitted regardless of them — so a permit narrowed by its issuer would be
// honoured in full. Refusing is the bar that keeps the deferral safe until
// constraints are implemented.
func (a AdminObjectsAction) ApplyConstraints(cs *astral.Bundle) bool {
	return cs == nil || len(cs.Objects()) == 0
}

func init() { astral.MustAdd(&AdminObjectsAction{}) }
