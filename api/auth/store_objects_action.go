package auth

import (
	"io"

	"github.com/astralp2p/astral-go/astral"
)

// StoreObjectsAction requests permission for Actor to write objects into the
// node: storing bytes, pushing objects at it, adding a repository, registering
// a type, and turning indexing of a repository on or off.
//
// One action covers every write that adds to node state. Deleting, purging and
// removing a repository are not here — an op that destroys is AdminObjects.
//
// Repo and Type declare the nouns a call touches. Nothing evaluates them yet —
// they are recorded so a constraint can bind to them later without changing the
// action type or any of its call sites. Either is left empty by an op that
// names neither, and by an op that reads its subject off the channel after the
// authorization decision is made.
type StoreObjectsAction struct {
	Action
	Repo astral.String8
	Type astral.String8
}

func (StoreObjectsAction) ObjectType() string { return "mod.auth.store_objects_action" }

func (a StoreObjectsAction) WriteTo(w io.Writer) (n int64, err error) {
	return astral.Objectify(&a).WriteTo(w)
}

func (a *StoreObjectsAction) ReadFrom(r io.Reader) (n int64, err error) {
	return astral.Objectify(a).ReadFrom(r)
}

// ApplyConstraints refuses a permit that carries any constraint. This action
// does not evaluate constraints, and an action that does not evaluate them is
// permitted regardless of them — so a permit narrowed by its issuer would be
// honoured in full. Refusing is the bar that keeps the deferral safe until
// constraints are implemented.
func (a StoreObjectsAction) ApplyConstraints(cs *astral.Bundle) bool {
	return cs == nil || len(cs.Objects()) == 0
}

func init() { astral.MustAdd(&StoreObjectsAction{}) }
