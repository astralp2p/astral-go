package user

import (
	"io"

	"github.com/astralp2p/astral-go/api/auth"
	"github.com/astralp2p/astral-go/astral"
)

// ExpelAction requests permission for Actor to expel Subject from the swarm.
// It is the one action a PrivilegedMembership does not cover (D7): only the
// user key expels, so the root handler never allows it.
type ExpelAction struct {
	auth.Action
	Subject *astral.Identity
}

func (ExpelAction) ObjectType() string { return "mod.user.expel_action" }

func (a ExpelAction) WriteTo(w io.Writer) (int64, error)   { return astral.Objectify(&a).WriteTo(w) }
func (a *ExpelAction) ReadFrom(r io.Reader) (int64, error) { return astral.Objectify(a).ReadFrom(r) }

// ApplyConstraints refuses a permit that carries any constraint; this action
// evaluates none (see AdminSwarmAction).
func (a ExpelAction) ApplyConstraints(cs *astral.Bundle) bool { return noConstraints(cs) }

func init() { astral.MustAdd(&ExpelAction{}) }
