package user

import (
	"io"

	"github.com/astralp2p/astral-go/api/auth"
	"github.com/astralp2p/astral-go/astral"
)

// SeeSwarmAction requests permission for Actor to read what the swarm is: the
// active contract's metadata, the sibling nodes and their link state, the bans
// the user has issued, and the asset set together with its delta ledger.
//
// One action covers every read in the user module, because between them they
// answer a single question — who and what belongs to this swarm. Splitting them
// protects nothing: a caller that can list siblings and stream the asset ledger
// already holds what guarding the contract metadata alone would keep from it.
//
// It replaces InfoAction, which named user.info and nothing else while the five
// reads beside it authorized nothing at all.
type SeeSwarmAction struct {
	auth.Action
}

func (SeeSwarmAction) ObjectType() string { return "mod.user.see_swarm_action" }

func (a SeeSwarmAction) WriteTo(w io.Writer) (n int64, err error) {
	return astral.Objectify(&a).WriteTo(w)
}

func (a *SeeSwarmAction) ReadFrom(r io.Reader) (n int64, err error) {
	return astral.Objectify(a).ReadFrom(r)
}

// ApplyConstraints refuses a permit that carries any constraint. This action
// does not evaluate constraints, and an action that does not evaluate them is
// permitted regardless of them — so a permit narrowed by its issuer would be
// honoured in full. Refusing is the bar that keeps the deferral safe until
// constraints are implemented.
func (a SeeSwarmAction) ApplyConstraints(cs *astral.Bundle) bool {
	return cs == nil || len(cs.Objects()) == 0
}

func init() { astral.MustAdd(&SeeSwarmAction{}) }
