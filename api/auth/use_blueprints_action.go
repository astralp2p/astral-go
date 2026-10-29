package auth

import (
	"io"

	"github.com/astralp2p/astral-go/astral"
)

// UseBlueprintsAction requests permission for Actor to read and register the
// node's blueprints: objects.blueprints, objects.get_blueprint and
// objects.register_blueprint.
//
// The action reads no object and writes no object, so a permit for blueprint
// sync grants nothing SeeObjects or StoreObjects governs. A registered
// blueprint is node-wide: every caller decodes the type through it.
type UseBlueprintsAction struct {
	Action
}

func (UseBlueprintsAction) ObjectType() string { return "mod.auth.use_blueprints_action" }

func (a UseBlueprintsAction) WriteTo(w io.Writer) (n int64, err error) {
	return astral.Objectify(&a).WriteTo(w)
}

func (a *UseBlueprintsAction) ReadFrom(r io.Reader) (n int64, err error) {
	return astral.Objectify(a).ReadFrom(r)
}

// ApplyConstraints refuses a permit that carries any constraint. This action
// does not evaluate constraints, and an action that does not evaluate them is
// permitted regardless of them — so a permit narrowed by its issuer would be
// honoured in full.
func (a UseBlueprintsAction) ApplyConstraints(cs *astral.Bundle) bool {
	return cs == nil || len(cs.Objects()) == 0
}

func init() { astral.MustAdd(&UseBlueprintsAction{}) }
