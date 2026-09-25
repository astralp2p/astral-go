package mcp

import (
	"bytes"
	"encoding/json"
	"testing"

	"github.com/astralp2p/astral-go/api/auth"
	"github.com/astralp2p/astral-go/astral"
)

// TestCallAgentActionRoundTrip covers the two shapes the action reaches the
// wire in: one naming a target, and the zero value, which is reachable over the
// wire for every registered type.
func TestCallAgentActionRoundTrip(t *testing.T) {
	actor := astral.GenerateIdentity()
	target := astral.GenerateIdentity()

	for _, tc := range []struct {
		name string
		a    *CallAgentAction
	}{
		{"named target", &CallAgentAction{Action: auth.NewAction(actor), ToID: target}},
		{"zero value", &CallAgentAction{}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var buf bytes.Buffer
			if _, err := tc.a.WriteTo(&buf); err != nil {
				t.Fatalf("write: %v", err)
			}

			var got CallAgentAction
			if _, err := got.ReadFrom(&buf); err != nil {
				t.Fatalf("read: %v", err)
			}

			if !got.ToID.IsEqual(tc.a.ToID) {
				t.Fatalf("to: got %v, want %v", got.ToID, tc.a.ToID)
			}
			if !got.Actor().IsEqual(tc.a.Actor()) {
				t.Fatalf("actor: got %v, want %v", got.Actor(), tc.a.Actor())
			}
		})
	}
}

// TestCallAgentActionRefusesConstrainedPermits is the safety bar: constraints
// are not evaluated, so a permit that carries one must be refused rather than
// honoured in full.
func TestCallAgentActionRefusesConstrainedPermits(t *testing.T) {
	action := &CallAgentAction{Action: auth.NewAction(astral.GenerateIdentity())}

	plain := &auth.Permit{Action: astral.String8(action.ObjectType())}
	if !plain.Allows(action) {
		t.Fatal("an unconstrained permit must allow the action")
	}

	constraints := astral.NewBundle()
	if err := constraints.Append(astral.NewError("any constraint at all")); err != nil {
		t.Fatalf("append constraint: %v", err)
	}

	narrowed := &auth.Permit{
		Action:      astral.String8(action.ObjectType()),
		Constraints: constraints,
	}
	if narrowed.Allows(action) {
		t.Fatal("a permit carrying constraints must be refused: nothing evaluates them")
	}
}

// TestCallAgentActionCarriesTheDocumentedJSON pins the shape astral-docs states
// and a json peer decodes: the embedded base action named rather than inlined.
func TestCallAgentActionCarriesTheDocumentedJSON(t *testing.T) {
	action := &CallAgentAction{Action: auth.NewAction(astral.GenerateIdentity()), ToID: astral.GenerateIdentity()}

	b, err := json.Marshal(action)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var got map[string]json.RawMessage
	if err = json.Unmarshal(b, &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if _, ok := got["Action"]; !ok {
		t.Fatalf("no Action member: the base action was inlined (%s)", b)
	}
	if _, ok := got["ToID"]; !ok {
		t.Fatalf("no ToID member (%s)", b)
	}
	if _, ok := got["ActorID"]; ok {
		t.Fatalf("ActorID sits at the top level: the base action was inlined (%s)", b)
	}
}

// TestCallAgentActionRoundTripsThroughJSON: what a json peer sends decodes back.
func TestCallAgentActionRoundTripsThroughJSON(t *testing.T) {
	actor, other := astral.GenerateIdentity(), astral.GenerateIdentity()

	b, err := json.Marshal(&CallAgentAction{Action: auth.NewAction(actor), ToID: other})
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var got CallAgentAction
	if err = json.Unmarshal(b, &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if !got.Actor().IsEqual(actor) {
		t.Fatalf("actor: got %v, want %v", got.Actor(), actor)
	}
	if !got.ToID.IsEqual(other) {
		t.Fatalf("to: got %v, want %v", got.ToID, other)
	}
}
