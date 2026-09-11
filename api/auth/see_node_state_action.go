package auth

import (
	"io"

	"github.com/astralp2p/astral-go/astral"
)

// SeeNodeStateAction requests permission for Actor to read the node's state:
// tree values and listings, the directory's alias map and filters, agent
// metadata, and the node's log stream.
//
// One action covers configuration reads and the log. A holder therefore reads
// other callers' logged activity as well as node metadata.
//
// Changing the same state is ConfigureNodeStateAction. Neither action implies
// the other.
type SeeNodeStateAction struct {
	Action
}

func (SeeNodeStateAction) ObjectType() string { return "mod.auth.see_node_state_action" }

func (a SeeNodeStateAction) WriteTo(w io.Writer) (n int64, err error) {
	return astral.Objectify(&a).WriteTo(w)
}

func (a *SeeNodeStateAction) ReadFrom(r io.Reader) (n int64, err error) {
	return astral.Objectify(a).ReadFrom(r)
}

// ApplyConstraints refuses a permit that carries any constraint. This action
// does not evaluate constraints, and an action that does not evaluate them is
// permitted regardless of them — so a permit narrowed by its issuer would be
// honoured in full. Refusing is the bar that keeps the deferral safe until
// constraints are implemented.
func (a SeeNodeStateAction) ApplyConstraints(cs *astral.Bundle) bool {
	return cs == nil || len(cs.Objects()) == 0
}

func init() { astral.MustAdd(&SeeNodeStateAction{}) }
