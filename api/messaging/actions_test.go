package messaging

import (
	"bytes"
	"encoding/json"
	"testing"

	"github.com/astralp2p/astral-go/api/auth"
	"github.com/astralp2p/astral-go/astral"
)

// TestSendActionRoundTrip covers the two shapes the action reaches the wire in:
// one naming a recipient, and the zero value, which is reachable over the wire
// for every registered type.
func TestSendActionRoundTrip(t *testing.T) {
	sender := astral.GenerateIdentity()
	recipient := astral.GenerateIdentity()

	for _, tc := range []struct {
		name string
		a    *SendAction
	}{
		{"named recipient", &SendAction{Action: auth.NewAction(sender), ToID: recipient}},
		{"zero value", &SendAction{}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var buf bytes.Buffer
			if _, err := tc.a.WriteTo(&buf); err != nil {
				t.Fatalf("write: %v", err)
			}

			var got SendAction
			if _, err := got.ReadFrom(&buf); err != nil {
				t.Fatalf("read: %v", err)
			}

			if !got.ToID.IsEqual(tc.a.ToID) {
				t.Fatalf("to: got %v, want %v", got.ToID, tc.a.ToID)
			}
			if !got.Actor().IsEqual(tc.a.Actor()) {
				t.Fatalf("actor: got %v, want %v", got.Actor(), tc.a.Actor())
			}
		})
	}
}

// TestReceiveActionRoundTrip is the same coverage for the inbound direction.
// Its actor is the recipient and FromID is the sender.
func TestReceiveActionRoundTrip(t *testing.T) {
	recipient := astral.GenerateIdentity()
	sender := astral.GenerateIdentity()

	for _, tc := range []struct {
		name string
		a    *ReceiveAction
	}{
		{"named sender", &ReceiveAction{Action: auth.NewAction(recipient), FromID: sender}},
		{"zero value", &ReceiveAction{}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var buf bytes.Buffer
			if _, err := tc.a.WriteTo(&buf); err != nil {
				t.Fatalf("write: %v", err)
			}

			var got ReceiveAction
			if _, err := got.ReadFrom(&buf); err != nil {
				t.Fatalf("read: %v", err)
			}

			if !got.FromID.IsEqual(tc.a.FromID) {
				t.Fatalf("from: got %v, want %v", got.FromID, tc.a.FromID)
			}
			if !got.Actor().IsEqual(tc.a.Actor()) {
				t.Fatalf("actor: got %v, want %v", got.Actor(), tc.a.Actor())
			}
		})
	}
}

// TestHostMailboxActionRoundTrip is the same coverage for hosting. Its actor is
// the hosting node and MailboxID is the identity whose mailbox it hosts.
func TestHostMailboxActionRoundTrip(t *testing.T) {
	node := astral.GenerateIdentity()
	mailbox := astral.GenerateIdentity()

	for _, tc := range []struct {
		name string
		a    *HostMailboxAction
	}{
		{"named mailbox", &HostMailboxAction{Action: auth.NewAction(node), MailboxID: mailbox}},
		{"zero value", &HostMailboxAction{}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var buf bytes.Buffer
			if _, err := tc.a.WriteTo(&buf); err != nil {
				t.Fatalf("write: %v", err)
			}

			var got HostMailboxAction
			if _, err := got.ReadFrom(&buf); err != nil {
				t.Fatalf("read: %v", err)
			}

			if !got.MailboxID.IsEqual(tc.a.MailboxID) {
				t.Fatalf("mailbox: got %v, want %v", got.MailboxID, tc.a.MailboxID)
			}
			if !got.Actor().IsEqual(tc.a.Actor()) {
				t.Fatalf("actor: got %v, want %v", got.Actor(), tc.a.Actor())
			}
		})
	}
}

// TestMessagingActionsAreDistinctTypes is what makes the actions separate
// decisions: the auth registry dispatches on the object type, and a permit
// matches by it, so a shared one would put both directions, mail and generic
// queries, or hosting and relaying under one authority. The mcp and nodes
// actions are spelled out because this package imports neither.
func TestMessagingActionsAreDistinctTypes(t *testing.T) {
	names := map[string]string{
		"send":         SendAction{}.ObjectType(),
		"receive":      ReceiveAction{}.ObjectType(),
		"host_mailbox": HostMailboxAction{}.ObjectType(),
		"mcp call":     "mod.mcp.call_agent_action",
		"nodes relay":  "mod.nodes.relay_for_action",
	}

	seen := map[string]string{}
	for name, objectType := range names {
		if other, ok := seen[objectType]; ok {
			t.Fatalf("%v and %v both report %q; the registry cannot tell them apart", name, other, objectType)
		}
		seen[objectType] = name
	}
}

// TestMessagingActionsRefuseConstrainedPermits is the safety bar: constraints
// are not evaluated, so a permit that carries one must be refused rather than
// honoured in full.
func TestMessagingActionsRefuseConstrainedPermits(t *testing.T) {
	for _, action := range []auth.ActionObject{
		&SendAction{Action: auth.NewAction(astral.GenerateIdentity())},
		&ReceiveAction{Action: auth.NewAction(astral.GenerateIdentity())},
		&HostMailboxAction{Action: auth.NewAction(astral.GenerateIdentity()), MailboxID: astral.GenerateIdentity()},
	} {
		t.Run(action.ObjectType(), func(t *testing.T) {
			plain := &auth.Permit{Action: astral.String8(action.ObjectType())}
			if !plain.Allows(action) {
				t.Fatal("an unconstrained permit must allow the action")
			}

			constraints := astral.NewBundle()
			if err := constraints.Append(astral.NewError("any constraint at all")); err != nil {
				t.Fatalf("append constraint: %v", err)
			}

			narrowed := &auth.Permit{
				Action:      astral.String8(action.ObjectType()),
				Constraints: constraints,
			}
			if narrowed.Allows(action) {
				t.Fatal("a permit carrying constraints must be refused: nothing evaluates them")
			}
		})
	}
}

// TestMessagingActionsCarryTheDocumentedJSON pins the shape astral-docs states
// and a json peer decodes: the embedded base action named rather than inlined.
func TestMessagingActionsCarryTheDocumentedJSON(t *testing.T) {
	actor, other := astral.GenerateIdentity(), astral.GenerateIdentity()

	for name, tc := range map[string]struct {
		action astral.Object
		field  string
	}{
		"send":         {&SendAction{Action: auth.NewAction(actor), ToID: other}, "ToID"},
		"receive":      {&ReceiveAction{Action: auth.NewAction(actor), FromID: other}, "FromID"},
		"host_mailbox": {&HostMailboxAction{Action: auth.NewAction(actor), MailboxID: other}, "MailboxID"},
	} {
		t.Run(name, func(t *testing.T) {
			got := jsonFields(t, tc.action)

			if _, ok := got["Action"]; !ok {
				t.Fatalf("no Action member: the base action was inlined (%v)", got)
			}
			if _, ok := got[tc.field]; !ok {
				t.Fatalf("no %v member (%v)", tc.field, got)
			}
			if _, ok := got["ActorID"]; ok {
				t.Fatalf("ActorID sits at the top level: the base action was inlined (%v)", got)
			}
		})
	}
}

// TestMessagingActionsRoundTripThroughJSON: what a json peer sends decodes
// back, actor included.
func TestMessagingActionsRoundTripThroughJSON(t *testing.T) {
	actor, other := astral.GenerateIdentity(), astral.GenerateIdentity()

	b, err := json.Marshal(&SendAction{Action: auth.NewAction(actor), ToID: other})
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var send SendAction
	if err = json.Unmarshal(b, &send); err != nil {
		t.Fatalf("unmarshal send: %v", err)
	}
	if !send.Actor().IsEqual(actor) || !send.ToID.IsEqual(other) {
		t.Fatalf("send: got actor %v to %v", send.Actor(), send.ToID)
	}

	b, err = json.Marshal(&ReceiveAction{Action: auth.NewAction(actor), FromID: other})
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var receive ReceiveAction
	if err = json.Unmarshal(b, &receive); err != nil {
		t.Fatalf("unmarshal receive: %v", err)
	}
	if !receive.Actor().IsEqual(actor) || !receive.FromID.IsEqual(other) {
		t.Fatalf("receive: got actor %v from %v", receive.Actor(), receive.FromID)
	}

	b, err = json.Marshal(&HostMailboxAction{Action: auth.NewAction(actor), MailboxID: other})
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var host HostMailboxAction
	if err = json.Unmarshal(b, &host); err != nil {
		t.Fatalf("unmarshal host_mailbox: %v", err)
	}
	if !host.Actor().IsEqual(actor) || !host.MailboxID.IsEqual(other) {
		t.Fatalf("host_mailbox: got actor %v mailbox %v", host.Actor(), host.MailboxID)
	}
}
