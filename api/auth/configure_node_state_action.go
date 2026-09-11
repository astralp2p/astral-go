package auth

import (
	"io"

	"github.com/astralp2p/astral-go/astral"
)

// ConfigureNodeStateAction requests permission for Actor to change the node's
// state: set and delete tree values, mount and unmount remote trees, and set or
// clear identity aliases.
//
// Reading the same state is SeeNodeStateAction. Neither action implies the
// other.
type ConfigureNodeStateAction struct {
	Action
}

func (ConfigureNodeStateAction) ObjectType() string { return "mod.auth.configure_node_state_action" }

func (a ConfigureNodeStateAction) WriteTo(w io.Writer) (n int64, err error) {
	return astral.Objectify(&a).WriteTo(w)
}

func (a *ConfigureNodeStateAction) ReadFrom(r io.Reader) (n int64, err error) {
	return astral.Objectify(a).ReadFrom(r)
}

// ApplyConstraints refuses a permit that carries any constraint. This action
// does not evaluate constraints, and an action that does not evaluate them is
// permitted regardless of them — so a permit narrowed by its issuer would be
// honoured in full. Refusing is the bar that keeps the deferral safe until
// constraints are implemented.
func (a ConfigureNodeStateAction) ApplyConstraints(cs *astral.Bundle) bool {
	return cs == nil || len(cs.Objects()) == 0
}

func init() { astral.MustAdd(&ConfigureNodeStateAction{}) }
