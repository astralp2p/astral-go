package auth

import (
	"bytes"
	"testing"

	"github.com/astralp2p/astral-go/astral"
)

// TestSeeObjectsActionRoundTrip covers both noun shapes the action is used in:
// an op that names a single object in a named repository, and one that names
// neither. The second case is the one that matters — a nil ObjectID reaches the
// wire from every enumerating op (scan, repositories, blueprints).
func TestSeeObjectsActionRoundTrip(t *testing.T) {
	var objectID astral.ObjectID
	objectID.Size = 42

	for _, tc := range []struct {
		name string
		a    *SeeObjectsAction
	}{
		{"both nouns", &SeeObjectsAction{ObjectID: &objectID, Repo: "cache"}},
		{"no nouns", &SeeObjectsAction{}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var buf bytes.Buffer
			if _, err := tc.a.WriteTo(&buf); err != nil {
				t.Fatalf("write: %v", err)
			}

			var got SeeObjectsAction
			if _, err := got.ReadFrom(&buf); err != nil {
				t.Fatalf("read: %v", err)
			}

			if got.Repo != tc.a.Repo {
				t.Fatalf("repo: got %q, want %q", got.Repo, tc.a.Repo)
			}
			switch {
			case (got.ObjectID == nil) != (tc.a.ObjectID == nil):
				t.Fatalf("object id: got %v, want %v", got.ObjectID, tc.a.ObjectID)
			case got.ObjectID != nil && !got.ObjectID.IsEqual(tc.a.ObjectID):
				t.Fatalf("object id: got %v, want %v", got.ObjectID, tc.a.ObjectID)
			}
		})
	}
}

// TestSeeObjectsActionRefusesConstrainedPermit is the safety bar: constraints
// are not implemented, so a permit that carries one must be refused rather than
// honoured in full.
func TestSeeObjectsActionRefusesConstrainedPermit(t *testing.T) {
	action := &SeeObjectsAction{}

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
