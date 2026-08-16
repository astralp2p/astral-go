package auth

import (
	"io"

	"github.com/astralp2p/astral-go/astral"
)

// SeeObjectsAction requests permission for Actor to read objects, object
// metadata, or the repository namespace itself.
//
// One action covers every read in the objects module: reading bytes and
// enumerating ids are not separated, because a caller who can enumerate a
// repository already holds what guarding a single object would protect.
//
// ObjectID and Repo declare the nouns a call touches. Nothing evaluates them
// yet — they are recorded so a constraint can bind to them later without
// changing the action type or any of its call sites. Either is left unset by
// an op that names no single object or no single repository.
type SeeObjectsAction struct {
	Action
	ObjectID *astral.ObjectID
	Repo     astral.String8
}

func (SeeObjectsAction) ObjectType() string { return "mod.auth.see_objects_action" }

func (a SeeObjectsAction) WriteTo(w io.Writer) (n int64, err error) {
	return astral.Objectify(&a).WriteTo(w)
}

func (a *SeeObjectsAction) ReadFrom(r io.Reader) (n int64, err error) {
	return astral.Objectify(a).ReadFrom(r)
}

// ApplyConstraints refuses a permit that carries any constraint. This action
// does not evaluate constraints, and an action that does not evaluate them is
// permitted regardless of them — so a permit narrowed by its issuer would be
// honoured in full. Refusing is the bar that keeps the deferral safe until
// constraints are implemented.
func (a SeeObjectsAction) ApplyConstraints(cs *astral.Bundle) bool {
	return cs == nil || len(cs.Objects()) == 0
}

func init() { astral.MustAdd(&SeeObjectsAction{}) }
