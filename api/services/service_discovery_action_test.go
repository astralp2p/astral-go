package services

import (
	"errors"
	"strings"
	"testing"

	"github.com/astralp2p/astral-go/api/auth"
	"github.com/astralp2p/astral-go/astral"
)

func rule(services []astral.String8, nodes ...*astral.Identity) *DiscoveryRule {
	return &DiscoveryRule{Services: services, Nodes: nodes}
}

func scope(rules ...*DiscoveryRule) *DiscoveryScope {
	return &DiscoveryScope{Rules: rules}
}

func constraints(objects ...astral.Object) *astral.Bundle {
	b := astral.NewBundle()
	if err := b.Append(objects...); err != nil {
		panic(err)
	}
	return b
}

// TestDiscoveryPermitNarrowsByServiceAndNode is what the scope is for: player
// anywhere, the wallet on one node only, and nothing unconstrained.
func TestDiscoveryPermitNarrowsByServiceAndNode(t *testing.T) {
	phone := astral.GenerateIdentity()
	laptop := astral.GenerateIdentity()

	paired := scope(
		rule([]astral.String8{"player"}),
		rule([]astral.String8{"bitcoin-wallet"}, laptop),
	)

	for _, tc := range []struct {
		name    string
		cs      *astral.Bundle
		service astral.String8
		node    *astral.Identity
		want    bool
	}{
		{"nil bundle covers nothing", nil, "player", phone, false},
		{"empty bundle covers nothing", astral.NewBundle(), "player", phone, false},
		{"foreign constraint refuses", constraints(astral.NewString8("player")), "player", phone, false},
		{"two scopes refuse", constraints(paired, scope(rule([]astral.String8{"player"}))), "player", phone, false},
		{"scope beside foreign object refuses", constraints(paired, astral.NewString8("x")), "player", phone, false},
		{"empty scope covers nothing", constraints(scope()), "player", phone, false},
		{"rule with no services covers nothing", constraints(scope(rule(nil))), "player", phone, false},

		{"any-node rule, phone", constraints(paired), "player", phone, true},
		{"any-node rule, laptop", constraints(paired), "player", laptop, true},
		{"node rule, its node", constraints(paired), "bitcoin-wallet", laptop, true},
		{"node rule, other node", constraints(paired), "bitcoin-wallet", phone, false},
		{"service not named", constraints(paired), "storage", phone, false},
		{"node rule, nil node", constraints(paired), "bitcoin-wallet", nil, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			a := ServiceDiscoveryAction{Service: tc.service, NodeID: tc.node}
			if got := a.ApplyConstraints(tc.cs); got != tc.want {
				t.Fatalf("ApplyConstraints: got %v, want %v", got, tc.want)
			}
		})
	}
}

func TestServiceDiscoveryActionValidateConstraints(t *testing.T) {
	node := astral.GenerateIdentity()

	for _, tc := range []struct {
		name    string
		cs      *astral.Bundle
		wantErr error
		allows  astral.String8 // for an accepted bundle: a service it allows
	}{
		{"nil bundle", nil, errOneDiscoveryScope, ""},
		{"empty bundle", astral.NewBundle(), errOneDiscoveryScope, ""},
		{"foreign object", constraints(astral.NewString8("x")), errOneDiscoveryScope, ""},
		{"two scopes", constraints(scope(rule([]astral.String8{"player"})), scope(rule([]astral.String8{"storage"}))), errOneDiscoveryScope, ""},
		{"scope beside foreign object", constraints(scope(rule([]astral.String8{"player"})), astral.NewString8("x")), errOneDiscoveryScope, ""},
		{"empty scope", constraints(scope()), errScopeNamesNoService, ""},
		{"rule with nil services", constraints(scope(rule(nil))), errScopeNamesNoService, ""},
		{"nil rule", constraints(scope(nil)), errScopeNamesNoService, ""},

		{"one scope naming inference", constraints(scope(rule([]astral.String8{"inference"}))), nil, "inference"},
		{"empty rule beside a rule naming player", constraints(scope(rule(nil), rule([]astral.String8{"player"}))), nil, "player"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			err := ServiceDiscoveryAction{}.ValidateConstraints(tc.cs)
			if !errors.Is(err, tc.wantErr) {
				t.Fatalf("ValidateConstraints: got %v, want %v", err, tc.wantErr)
			}
			if tc.wantErr != nil {
				return
			}
			a := ServiceDiscoveryAction{Service: tc.allows, NodeID: node}
			if !a.ApplyConstraints(tc.cs) {
				t.Fatalf("accepted bundle does not allow %v", tc.allows)
			}
		})
	}
}

func TestValidatePermitDiscovery(t *testing.T) {
	const action = "mod.services.service_discovery_action"
	one := scope(rule([]astral.String8{"inference"}))

	for _, tc := range []struct {
		name    string
		cs      *astral.Bundle
		refused string // empty: passes
	}{
		{"nil constraints", nil, "requires constraints"},
		{"two scopes", constraints(one, scope(rule([]astral.String8{"player"}))), errOneDiscoveryScope.Error()},
		{"one valid scope", constraints(one), ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			err := auth.ValidatePermit(&auth.Permit{Action: action, Constraints: tc.cs})
			if tc.refused == "" {
				if err != nil {
					t.Fatalf("ValidatePermit: %v, want nil", err)
				}
				return
			}
			if !errors.Is(err, auth.ErrInvalidPermit) {
				t.Fatalf("errors.Is(%v, auth.ErrInvalidPermit) = false", err)
			}
			if !strings.Contains(err.Error(), action) || !strings.Contains(err.Error(), tc.refused) {
				t.Fatalf("error %q does not name %q and %q", err, action, tc.refused)
			}
		})
	}
}
