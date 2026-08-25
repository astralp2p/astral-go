package mcp

import (
	"io"

	"github.com/astralp2p/astral-go/api/auth"
	"github.com/astralp2p/astral-go/astral"
)

// CallAgentAction requests permission for Actor to start a query to ToID.
//
// The actor is the calling agent, and the action asks what that agent is
// permitted to reach. What the agent on the other side is permitted to answer
// is AnswerAgentAction. A call proceeds only when both are granted, and neither
// party's permission decides the other's.
type CallAgentAction struct {
	auth.Action
	ToID *astral.Identity
}

func (CallAgentAction) ObjectType() string { return "mod.mcp.call_agent_action" }

func (a CallAgentAction) WriteTo(w io.Writer) (n int64, err error) {
	return astral.Objectify(&a).WriteTo(w)
}

func (a *CallAgentAction) ReadFrom(r io.Reader) (n int64, err error) {
	return astral.Objectify(a).ReadFrom(r)
}

// json
//
// why declared and not left to the default: an embedded struct marshals inline
// unless something names it, so without these the action would reach a json
// peer as a flat object while the spec documents it as an Action beside
// ToID. Contract and Permit declare the same pair for the same reason.

func (a CallAgentAction) MarshalJSON() ([]byte, error) {
	return astral.Objectify(&a).MarshalJSON()
}

func (a *CallAgentAction) UnmarshalJSON(b []byte) error {
	return astral.Objectify(a).UnmarshalJSON(b)
}

// ApplyConstraints refuses a permit that carries any constraint. This action
// does not evaluate constraints, and an action that does not evaluate them is
// permitted regardless of them — so a permit narrowed by its issuer would be
// honoured in full. Refusing is the bar that keeps the deferral safe until
// constraints are implemented.
func (a CallAgentAction) ApplyConstraints(cs *astral.Bundle) bool {
	return cs == nil || len(cs.Objects()) == 0
}

func init() { astral.MustAdd(&CallAgentAction{}) }
