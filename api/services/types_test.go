package services

import (
	"bytes"
	"testing"

	"github.com/astralp2p/astral-go/api/auth"
	"github.com/astralp2p/astral-go/astral"
)

type wireObject interface {
	astral.Object
}

// roundTrip encodes obj, decodes it into fresh, and requires the second
// encoding to equal the first, in binary and in JSON.
func roundTrip(t *testing.T, obj, fresh wireObject, freshJSON wireObject) {
	t.Helper()

	var first bytes.Buffer
	if _, err := obj.WriteTo(&first); err != nil {
		t.Fatalf("write: %v", err)
	}
	want := append([]byte(nil), first.Bytes()...)
	if _, err := fresh.ReadFrom(&first); err != nil {
		t.Fatalf("read: %v", err)
	}
	var second bytes.Buffer
	if _, err := fresh.WriteTo(&second); err != nil {
		t.Fatalf("rewrite: %v", err)
	}
	if !bytes.Equal(want, second.Bytes()) {
		t.Fatalf("binary round trip changed %s", obj.ObjectType())
	}

	j, err := astral.Objectify(obj).MarshalJSON()
	if err != nil {
		t.Fatalf("marshal json: %v", err)
	}
	if err := astral.Objectify(freshJSON).UnmarshalJSON(j); err != nil {
		t.Fatalf("unmarshal json: %v (%s)", err, j)
	}
	j2, err := astral.Objectify(freshJSON).MarshalJSON()
	if err != nil {
		t.Fatalf("remarshal json: %v", err)
	}
	if !bytes.Equal(j, j2) {
		t.Fatalf("json round trip changed %s:\n%s\n%s", obj.ObjectType(), j, j2)
	}
}

func TestWireTypesRoundTrip(t *testing.T) {
	caller := astral.GenerateIdentity()
	provider := astral.GenerateIdentity()
	node := astral.GenerateIdentity()

	ops := astral.NewBundle()
	if err := ops.Append(&OperationsList{Operations: []astral.String8{"player.play", "player.pause"}}); err != nil {
		t.Fatal(err)
	}

	for _, tc := range []struct {
		name      string
		obj, a, b wireObject
	}{
		{"ask", &Ask{RequestID: astral.NewNonce(), CallerID: caller, Service: "player"}, &Ask{}, &Ask{}},
		{"answer", &Answer{RequestID: astral.NewNonce(), Update: &Update{Available: true, Name: "player", Info: ops}}, &Answer{}, &Answer{}},
		{"change callers", &Change{Callers: []*astral.Identity{caller, provider}}, &Change{}, &Change{}},
		{"change all", &Change{All: true}, &Change{}, &Change{}},
		{"operations list", &OperationsList{Operations: []astral.String8{"a.b"}}, &OperationsList{}, &OperationsList{}},
		{"incomplete", &Incomplete{Services: []astral.String8{"player"}}, &Incomplete{}, &Incomplete{}},
		{"offering key", &OfferingKey{ProviderID: provider, Name: "player"}, &OfferingKey{}, &OfferingKey{}},
		{"removed", &Removed{Offerings: []*OfferingKey{{ProviderID: provider, Name: "player"}, {ProviderID: provider, Name: "bitcoin-wallet"}}}, &Removed{}, &Removed{}},
		{"discovery rule", &DiscoveryRule{Services: []astral.String8{"player"}, Nodes: []*astral.Identity{node}}, &DiscoveryRule{}, &DiscoveryRule{}},
		{"discovery scope", scope(rule([]astral.String8{"player"})), &DiscoveryScope{}, &DiscoveryScope{}},
		{"action", &ServiceDiscoveryAction{Action: auth.NewAction(caller), Service: "player", NodeID: node}, &ServiceDiscoveryAction{}, &ServiceDiscoveryAction{}},
	} {
		t.Run(tc.name, func(t *testing.T) { roundTrip(t, tc.obj, tc.a, tc.b) })
	}
}

func TestConstraintsBundleJSONRoundTrip(t *testing.T) {
	cs := constraints(scope(rule([]astral.String8{"player"})))

	j, err := cs.MarshalJSON()
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	got := astral.NewBundle()
	if err := got.UnmarshalJSON(j); err != nil {
		t.Fatalf("unmarshal: %v (%s)", err, j)
	}

	action := ServiceDiscoveryAction{Service: "player", NodeID: astral.GenerateIdentity()}
	if !action.ApplyConstraints(got) {
		t.Fatal("decoded scope does not allow the service it names")
	}
}

func TestChangeValid(t *testing.T) {
	if (Change{}).Valid() {
		t.Fatal("a Change selecting no caller is valid")
	}
	if !(Change{All: true}).Valid() {
		t.Fatal("Change{All} is invalid")
	}
	if !(Change{Callers: []*astral.Identity{astral.GenerateIdentity()}}).Valid() {
		t.Fatal("Change with a caller is invalid")
	}
}
