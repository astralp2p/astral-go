package auth

import (
	"bytes"
	"testing"

	"github.com/astralp2p/astral-go/astral"
)

func roles(names ...astral.String8) *astral.Bundle {
	bundle := astral.NewBundle()
	for _, name := range names {
		if err := bundle.Append(astral.NewString8(string(name))); err != nil {
			panic(err)
		}
	}
	return bundle
}

func TestServeObjectsActionRoundTrip(t *testing.T) {
	for _, tc := range []struct {
		name string
		a    *ServeObjectsAction
	}{
		{"with role", &ServeObjectsAction{Role: RoleDescriber}},
		{"no role", &ServeObjectsAction{}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var buf bytes.Buffer
			if _, err := tc.a.WriteTo(&buf); err != nil {
				t.Fatalf("write: %v", err)
			}

			var got ServeObjectsAction
			if _, err := got.ReadFrom(&buf); err != nil {
				t.Fatalf("read: %v", err)
			}

			if got.Role != tc.a.Role {
				t.Fatalf("role: got %q, want %q", got.Role, tc.a.Role)
			}
		})
	}
}

// TestServeObjectsPermitNarrowsByRole is what the constraints are for: one
// permit that lets an app describe without letting it search.
func TestServeObjectsPermitNarrowsByRole(t *testing.T) {
	for _, tc := range []struct {
		name        string
		constraints *astral.Bundle
		role        astral.String8
		want        bool
	}{
		{"unconstrained covers describer", nil, RoleDescriber, true},
		{"unconstrained covers searcher", astral.NewBundle(), RoleSearcher, true},

		{"one role, matching", roles(RoleDescriber), RoleDescriber, true},
		{"one role, not matching", roles(RoleDescriber), RoleSearcher, false},

		{"two roles, first", roles(RoleDescriber, RoleFinder), RoleDescriber, true},
		{"two roles, second", roles(RoleDescriber, RoleFinder), RoleFinder, true},
		{"two roles, neither", roles(RoleDescriber, RoleFinder), RoleSearcher, false},

		{"empty role against a constrained permit", roles(RoleDescriber), "", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			action := &ServeObjectsAction{Role: tc.role}
			permit := &Permit{
				Action:      astral.String8(action.ObjectType()),
				Constraints: tc.constraints,
			}

			if got := permit.Allows(action); got != tc.want {
				t.Fatalf("Allows(role=%q) = %v, want %v", tc.role, got, tc.want)
			}
		})
	}
}

// TestServeObjectsRefusesUnreadableConstraint is the safety property: a permit
// carrying a constraint this action cannot interpret is refused, even when a
// role beside it matches. Honouring it would grant in full what its issuer
// meant to limit.
func TestServeObjectsRefusesUnreadableConstraint(t *testing.T) {
	constraints := roles(RoleDescriber)
	if err := constraints.Append(astral.NewError("a constraint from some later version")); err != nil {
		t.Fatalf("append: %v", err)
	}

	action := &ServeObjectsAction{Role: RoleDescriber}
	permit := &Permit{
		Action:      astral.String8(action.ObjectType()),
		Constraints: constraints,
	}

	if permit.Allows(action) {
		t.Fatal("a permit carrying an unreadable constraint must be refused, matching role or not")
	}
}

// TestServeObjectsPermitMatchesActionType guards the other half of Allows: the
// constraints never get consulted for a different action.
func TestServeObjectsPermitMatchesActionType(t *testing.T) {
	permit := &Permit{Action: astral.String8("mod.auth.some_other_action")}

	if permit.Allows(&ServeObjectsAction{Role: RoleDescriber}) {
		t.Fatal("a permit for another action type must not allow ServeObjects")
	}
}
