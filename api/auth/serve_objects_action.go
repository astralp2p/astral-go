package auth

import (
	"io"

	"github.com/astralp2p/astral-go/astral"
)

// Roles a ServeObjectsAction can name.
const (
	RoleDescriber = astral.String8("describer")
	RoleFinder    = astral.String8("finder")
	RoleSearcher  = astral.String8("searcher")
)

// ServeObjectsAction requests permission for Actor to be consulted by the node
// when it answers object queries, as a describer, a finder, or a searcher.
//
// The authority is not access to data. It is a place in the node's answer path:
// afterwards the node calls out to Actor on every matching query, whoever
// asked, and relays what comes back. Registering is the mechanism.
//
// Role names which of the three a call asks for. Unlike the nouns the other
// actions declare, this one is evaluated — a permit's constraints narrow it.
type ServeObjectsAction struct {
	Action
	Role astral.String8
}

func (ServeObjectsAction) ObjectType() string { return "mod.auth.serve_objects_action" }

func (a ServeObjectsAction) WriteTo(w io.Writer) (n int64, err error) {
	return astral.Objectify(&a).WriteTo(w)
}

func (a *ServeObjectsAction) ReadFrom(r io.Reader) (n int64, err error) {
	return astral.Objectify(a).ReadFrom(r)
}

// ApplyConstraints narrows the permit to the roles its constraints name.
//
// An empty bundle is unconstrained and covers every role. Otherwise the bundle
// holds String8 roles and the permit covers exactly those.
//
// why: an object of any other type refuses the permit outright, even alongside a
// role that matches. Honouring a narrowing this action cannot read would grant
// in full what its issuer meant to limit, which is the failure the whole
// constraint check exists to prevent.
func (a ServeObjectsAction) ApplyConstraints(cs *astral.Bundle) bool {
	if cs == nil || len(cs.Objects()) == 0 {
		return true
	}

	var allowed bool

	for _, o := range cs.Objects() {
		role, ok := o.(*astral.String8)
		if !ok {
			return false
		}

		if *role == a.Role {
			allowed = true
		}
	}

	return allowed
}

func init() { astral.MustAdd(&ServeObjectsAction{}) }
