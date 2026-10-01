package services

import (
	"testing"

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
