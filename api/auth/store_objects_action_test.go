package auth

import (
	"bytes"
	"testing"

	"github.com/astralp2p/astral-go/astral"
)

// TestStoreObjectsActionRoundTrip covers both noun shapes the action is used
// in: an op that names the repository it writes to, and one that names nothing
// because its subject arrives on the channel after the decision is made.
func TestStoreObjectsActionRoundTrip(t *testing.T) {
	for _, tc := range []struct {
		name string
		a    *StoreObjectsAction
	}{
		{"both nouns", &StoreObjectsAction{Repo: "local", Type: "mod.auth.contract"}},
		{"no nouns", &StoreObjectsAction{}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var buf bytes.Buffer
			if _, err := tc.a.WriteTo(&buf); err != nil {
				t.Fatalf("write: %v", err)
			}

			var got StoreObjectsAction
			if _, err := got.ReadFrom(&buf); err != nil {
				t.Fatalf("read: %v", err)
			}

			if got.Repo != tc.a.Repo {
				t.Fatalf("repo: got %q, want %q", got.Repo, tc.a.Repo)
			}
			if got.Type != tc.a.Type {
				t.Fatalf("type: got %q, want %q", got.Type, tc.a.Type)
			}
		})
	}
}

// TestStoreObjectsActionRefusesConstrainedPermit is the safety bar: constraints
// are not implemented, so a permit that carries one must be refused rather than
// honoured in full.
func TestStoreObjectsActionRefusesConstrainedPermit(t *testing.T) {
	action := &StoreObjectsAction{}

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
