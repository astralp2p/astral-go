package auth

import (
	"bytes"
	"testing"

	"github.com/astralp2p/astral-go/api/crypto"
	"github.com/astralp2p/astral-go/astral"
)

// TestRegisterSignerActionRoundTrip covers the two shapes the action reaches the
// wire in: one naming an actor and a key, and the zero value, which is reachable
// over the wire for every registered type.
func TestRegisterSignerActionRoundTrip(t *testing.T) {
	key := &crypto.PublicKey{Type: "secp256k1", Key: bytes.Repeat([]byte{2}, 33)}

	for _, tc := range []struct {
		name string
		a    *RegisterSignerAction
	}{
		{"named actor and key", &RegisterSignerAction{Action: NewAction(astral.GenerateIdentity()), Key: key}},
		{"zero value", &RegisterSignerAction{}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var buf bytes.Buffer
			if _, err := tc.a.WriteTo(&buf); err != nil {
				t.Fatalf("write: %v", err)
			}

			var got RegisterSignerAction
			if _, err := got.ReadFrom(&buf); err != nil {
				t.Fatalf("read: %v", err)
			}

			if got.Nonce != tc.a.Nonce {
				t.Fatalf("nonce: got %v, want %v", got.Nonce, tc.a.Nonce)
			}
			if !got.Actor().IsEqual(tc.a.Actor()) {
				t.Fatalf("actor: got %v, want %v", got.Actor(), tc.a.Actor())
			}
			if tc.a.Key != nil && (got.Key == nil || !bytes.Equal(got.Key.Key, tc.a.Key.Key)) {
				t.Fatalf("key: got %v, want %v", got.Key, tc.a.Key)
			}
		})
	}
}

// TestRegisterSignerActionRefusesConstrainedPermit is the safety bar: constraints
// are not evaluated, so a permit that carries one must be refused rather than
// honoured in full.
func TestRegisterSignerActionRefusesConstrainedPermit(t *testing.T) {
	action := &RegisterSignerAction{Action: NewAction(astral.GenerateIdentity())}

	if !(&Permit{Action: astral.String8(action.ObjectType())}).Allows(action) {
		t.Fatal("an unconstrained permit must allow the action")
	}

	constraints := astral.NewBundle()
	if err := constraints.Append(astral.NewError("any constraint at all")); err != nil {
		t.Fatalf("append constraint: %v", err)
	}

	permit := &Permit{Action: astral.String8(action.ObjectType()), Constraints: constraints}
	if permit.Allows(action) {
		t.Fatal("a permit carrying constraints must be refused: nothing evaluates them")
	}
}
