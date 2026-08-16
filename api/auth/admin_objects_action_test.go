package auth

import (
	"bytes"
	"testing"

	"github.com/astralp2p/astral-go/astral"
)

// TestAdminObjectsActionRoundTrip covers the noun shapes the action is used in:
// an op that names a path and the repository it attaches, and one that names
// nothing because it addresses the whole node.
func TestAdminObjectsActionRoundTrip(t *testing.T) {
	for _, tc := range []struct {
		name string
		a    *AdminObjectsAction
	}{
		{"repo and path", &AdminObjectsAction{Repo: "photos", Path: "/srv/photos"}},
		{"no nouns", &AdminObjectsAction{}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var buf bytes.Buffer
			if _, err := tc.a.WriteTo(&buf); err != nil {
				t.Fatalf("write: %v", err)
			}

			var got AdminObjectsAction
			if _, err := got.ReadFrom(&buf); err != nil {
				t.Fatalf("read: %v", err)
			}

			if got.Repo != tc.a.Repo {
				t.Fatalf("repo: got %q, want %q", got.Repo, tc.a.Repo)
			}
			if got.Path != tc.a.Path {
				t.Fatalf("path: got %q, want %q", got.Path, tc.a.Path)
			}
		})
	}
}

// TestAdminObjectsActionRefusesConstrainedPermit is the safety bar: constraints
// are not implemented, so a permit that carries one must be refused rather than
// honoured in full.
func TestAdminObjectsActionRefusesConstrainedPermit(t *testing.T) {
	action := &AdminObjectsAction{}

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
