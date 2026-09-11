package auth

import (
	"bytes"
	"reflect"
	"testing"

	"github.com/astralp2p/astral-go/astral"
)

// nodeActions lists the node-wide actions that carry no nouns: each names a
// capability over a whole area of the node and nothing narrower.
func nodeActions() []ActionObject {
	return []ActionObject{
		&SeeNodeStateAction{},
		&ConfigureNodeStateAction{},
		&UseGatewayAction{},
		&ServeAppsAction{},
		&AdminNetworkAction{},
		&AdminManageAppsAction{},
	}
}

// blank returns a fresh zero value of the action's concrete type.
func blank(a ActionObject) ActionObject {
	return reflect.New(reflect.TypeOf(a).Elem()).Interface().(ActionObject)
}

// TestNodeActionsRoundTrip covers the two shapes each action reaches the wire
// in: one naming an actor, and the zero value, which is reachable over the wire
// for every registered type.
func TestNodeActionsRoundTrip(t *testing.T) {
	for _, proto := range nodeActions() {
		for name, sent := range map[string]ActionObject{
			"named actor": func() ActionObject {
				a := blank(proto)
				a.SetActor(astral.GenerateIdentity())
				return a
			}(),
			"zero value": blank(proto),
		} {
			t.Run(proto.ObjectType()+"/"+name, func(t *testing.T) {
				var buf bytes.Buffer
				if _, err := sent.WriteTo(&buf); err != nil {
					t.Fatalf("write: %v", err)
				}

				got := blank(proto)
				if _, err := got.ReadFrom(&buf); err != nil {
					t.Fatalf("read: %v", err)
				}

				if got.Id() != sent.Id() {
					t.Fatalf("nonce: got %v, want %v", got.Id(), sent.Id())
				}
				if !got.Actor().IsEqual(sent.Actor()) {
					t.Fatalf("actor: got %v, want %v", got.Actor(), sent.Actor())
				}
			})
		}
	}
}

// TestNodeActionsRefuseConstrainedPermits is the safety bar: constraints are
// not evaluated, so a permit that carries one must be refused rather than
// honoured in full.
func TestNodeActionsRefuseConstrainedPermits(t *testing.T) {
	for _, action := range nodeActions() {
		t.Run(action.ObjectType(), func(t *testing.T) {
			plain := &Permit{Action: astral.String8(action.ObjectType())}
			if !plain.Allows(action) {
				t.Fatal("an unconstrained permit must allow the action")
			}

			constraints := astral.NewBundle()
			if err := constraints.Append(astral.NewError("any constraint at all")); err != nil {
				t.Fatalf("append constraint: %v", err)
			}

			narrowed := &Permit{Action: astral.String8(action.ObjectType()), Constraints: constraints}
			if narrowed.Allows(action) {
				t.Fatal("a permit carrying constraints must be refused: nothing evaluates them")
			}
		})
	}
}

// TestNodeActionsAreDistinct is what makes each action its own decision: the
// auth registry dispatches on the object type, so a permit for one must not
// allow another — the read tier must not carry the write tier, and gateway use
// must not carry network administration.
func TestNodeActionsAreDistinct(t *testing.T) {
	all := append(nodeActions(),
		&SeeObjectsAction{}, &StoreObjectsAction{}, &ServeObjectsAction{}, &AdminObjectsAction{},
	)

	for _, held := range all {
		permit := &Permit{Action: astral.String8(held.ObjectType())}
		for _, asked := range all {
			if held.ObjectType() == asked.ObjectType() {
				continue
			}
			if permit.Allows(asked) {
				t.Fatalf("a %v permit allows %v", held.ObjectType(), asked.ObjectType())
			}
		}
	}
}
