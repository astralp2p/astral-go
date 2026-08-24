package mcp

import (
	"io"

	"github.com/astralp2p/astral-go/api/auth"
	"github.com/astralp2p/astral-go/astral"
)

// AnswerAgentAction requests permission for Actor to answer a query from
// FromID.
//
// The actor is the agent being called, not the caller. Every action type names
// what its actor does, and answering is the called agent's act.
//
// why it matters beyond the name: auth resolves an action no handler grants by
// walking the contracts the actor is subject to, re-entering as each issuer. An
// action naming the caller as actor would search the caller's delegations for a
// permission the called agent's side holds, and a stranger's contracts would
// decide whether this agent answers.
//
// What the caller is permitted to reach is CallAgentAction. A call proceeds
// only when both are granted, and neither party's permission decides the
// other's.
type AnswerAgentAction struct {
	auth.Action
	FromID *astral.Identity
}

func (AnswerAgentAction) ObjectType() string { return "mod.mcp.answer_agent_action" }

func (a AnswerAgentAction) WriteTo(w io.Writer) (n int64, err error) {
	return astral.Objectify(&a).WriteTo(w)
}

func (a *AnswerAgentAction) ReadFrom(r io.Reader) (n int64, err error) {
	return astral.Objectify(a).ReadFrom(r)
}

// ApplyConstraints refuses a permit that carries any constraint, as
// CallAgentAction does and for the same reason: nothing evaluates them, so a
// permit its issuer narrowed would otherwise be honoured in full.
func (a AnswerAgentAction) ApplyConstraints(cs *astral.Bundle) bool {
	return cs == nil || len(cs.Objects()) == 0
}

func init() { astral.MustAdd(&AnswerAgentAction{}) }
