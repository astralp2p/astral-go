package user

import (
	"bytes"
	"testing"

	"github.com/astralp2p/astral-go/api/auth"
	"github.com/astralp2p/astral-go/astral"
)

func constrained(objects ...astral.Object) *astral.Bundle {
	bundle := astral.NewBundle()
	for _, o := range objects {
		if err := bundle.Append(o); err != nil {
			panic(err)
		}
	}
	return bundle
}

func TestSeeSwarmActionRoundTrip(t *testing.T) {
	var buf bytes.Buffer
	sent := &SeeSwarmAction{Action: auth.NewAction(nil)}

	if _, err := sent.WriteTo(&buf); err != nil {
		t.Fatalf("write: %v", err)
	}

	var got SeeSwarmAction
	if _, err := got.ReadFrom(&buf); err != nil {
		t.Fatalf("read: %v", err)
	}

	if got.Nonce != sent.Nonce {
		t.Fatalf("nonce: got %v, want %v", got.Nonce, sent.Nonce)
	}
}

func TestAdminSwarmActionRoundTrip(t *testing.T) {
	for _, tc := range []struct {
		name string
		a    *AdminSwarmAction
	}{
		{"membership call", &AdminSwarmAction{Action: auth.NewAction(nil), Subject: &astral.Identity{}}},
		{"asset call", &AdminSwarmAction{Action: auth.NewAction(nil), ObjectID: &astral.ObjectID{}}},
		{"neither noun", &AdminSwarmAction{Action: auth.NewAction(nil)}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var buf bytes.Buffer
			if _, err := tc.a.WriteTo(&buf); err != nil {
				t.Fatalf("write: %v", err)
			}

			var got AdminSwarmAction
			if _, err := got.ReadFrom(&buf); err != nil {
				t.Fatalf("read: %v", err)
			}

			if got.Nonce != tc.a.Nonce {
				t.Fatalf("nonce: got %v, want %v", got.Nonce, tc.a.Nonce)
			}
		})
	}
}

// TestSwarmActionsRefuseConstrainedPermits is the deferral bar: neither action
// evaluates a constraint, and an action that does not evaluate one is permitted
// regardless of it — so a permit its issuer narrowed must be refused outright
// rather than honoured in full.
func TestSwarmActionsRefuseConstrainedPermits(t *testing.T) {
	for _, tc := range []struct {
		name   string
		action auth.ActionObject
	}{
		{"see_swarm", &SeeSwarmAction{}},
		{"admin_swarm", &AdminSwarmAction{}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			unconstrained := &auth.Permit{Action: astral.String8(tc.action.ObjectType())}
			if !unconstrained.Allows(tc.action) {
				t.Fatal("an unconstrained permit must allow its own action")
			}

			narrowed := &auth.Permit{
				Action:      astral.String8(tc.action.ObjectType()),
				Constraints: constrained(astral.NewString8("some later narrowing")),
			}
			if narrowed.Allows(tc.action) {
				t.Fatal("a permit carrying a constraint this action cannot evaluate must be refused")
			}
		})
	}
}

// TestSwarmActionsAreDistinct guards the consolidation: a read permit must not
// carry the authority to change the swarm.
func TestSwarmActionsAreDistinct(t *testing.T) {
	see := &auth.Permit{Action: astral.String8(SeeSwarmAction{}.ObjectType())}

	if see.Allows(&AdminSwarmAction{}) {
		t.Fatal("a see_swarm permit must not allow admin_swarm")
	}

	admin := &auth.Permit{Action: astral.String8(AdminSwarmAction{}.ObjectType())}

	if admin.Allows(&SeeSwarmAction{}) {
		t.Fatal("an admin_swarm permit must not allow see_swarm")
	}
}

// TestNodeContractCarriesBothSwarmPermits pins what a management node receives:
// the two swarm actions, delegable one hop, and nothing else beside membership.
func TestNodeContractCarriesBothSwarmPermits(t *testing.T) {
	contract, err := NewNodeContract(&astral.Identity{}, &astral.Identity{}, true, 0)
	if err != nil {
		t.Fatalf("new node contract: %v", err)
	}

	for _, want := range []string{
		SwarmMembershipAction{}.ObjectType(),
		SeeSwarmAction{}.ObjectType(),
		AdminSwarmAction{}.ObjectType(),
	} {
		if len(contract.HasPermit(want)) == 0 {
			t.Fatalf("management node contract is missing a %v permit", want)
		}
	}

	if len(contract.Permits) != 3 {
		t.Fatalf("permits: got %d, want 3", len(contract.Permits))
	}

	plain, err := NewNodeContract(&astral.Identity{}, &astral.Identity{}, false, 0)
	if err != nil {
		t.Fatalf("new node contract: %v", err)
	}

	if len(plain.Permits) != 1 {
		t.Fatalf("a non-management contract carries membership alone: got %d permits", len(plain.Permits))
	}
}
