package coldcard

import (
	"bytes"
	"testing"

	"github.com/astralp2p/astral-go/api/auth"
	"github.com/astralp2p/astral-go/astral"
)

// TestScanActionRoundTrip covers the two shapes the action reaches the wire in:
// one naming an actor, and the zero value, which is reachable over the wire for
// every registered type.
func TestScanActionRoundTrip(t *testing.T) {
	for _, tc := range []struct {
		name string
		a    *ScanAction
	}{
		{"named actor", &ScanAction{Action: auth.NewAction(astral.GenerateIdentity())}},
		{"zero value", &ScanAction{}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var buf bytes.Buffer
			if _, err := tc.a.WriteTo(&buf); err != nil {
				t.Fatalf("write: %v", err)
			}

			var got ScanAction
			if _, err := got.ReadFrom(&buf); err != nil {
				t.Fatalf("read: %v", err)
			}

			if got.Nonce != tc.a.Nonce {
				t.Fatalf("nonce: got %v, want %v", got.Nonce, tc.a.Nonce)
			}
			if !got.Actor().IsEqual(tc.a.Actor()) {
				t.Fatalf("actor: got %v, want %v", got.Actor(), tc.a.Actor())
			}
		})
	}
}

// TestScanActionRefusesConstrainedPermit is the safety bar: constraints are not
// evaluated, so a permit that carries one must be refused rather than honoured
// in full.
func TestScanActionRefusesConstrainedPermit(t *testing.T) {
	action := &ScanAction{Action: auth.NewAction(astral.GenerateIdentity())}

	if !(&auth.Permit{Action: astral.String8(action.ObjectType())}).Allows(action) {
		t.Fatal("an unconstrained permit must allow the action")
	}

	constraints := astral.NewBundle()
	if err := constraints.Append(astral.NewError("any constraint at all")); err != nil {
		t.Fatalf("append constraint: %v", err)
	}

	permit := &auth.Permit{Action: astral.String8(action.ObjectType()), Constraints: constraints}
	if permit.Allows(action) {
		t.Fatal("a permit carrying constraints must be refused: nothing evaluates them")
	}
}
