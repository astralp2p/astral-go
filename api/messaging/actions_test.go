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

// TestReadMailboxActionRoundTrip is the same coverage for a delegated read. Its
// actor is the reading identity and MailboxID is the identity whose mailbox it
// reads.
func TestReadMailboxActionRoundTrip(t *testing.T) {
	reader := astral.GenerateIdentity()
	mailbox := astral.GenerateIdentity()

	for _, tc := range []struct {
		name string
		a    *ReadMailboxAction
	}{
		{"named mailbox", &ReadMailboxAction{Action: auth.NewAction(reader), MailboxID: mailbox}},
		{"zero value", &ReadMailboxAction{}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var buf bytes.Buffer
			if _, err := tc.a.WriteTo(&buf); err != nil {
				t.Fatalf("write: %v", err)
			}

			var got ReadMailboxAction
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
// matches by it, so a shared one would put both directions, hosting and
// relaying, or hosting and reading, under one authority. The nodes action is
// spelled out because this package does not import nodes.
func TestMessagingActionsAreDistinctTypes(t *testing.T) {
	names := map[string]string{
		"send":         SendAction{}.ObjectType(),
		"receive":      ReceiveAction{}.ObjectType(),
		"host_mailbox": HostMailboxAction{}.ObjectType(),
		"read_mailbox": ReadMailboxAction{}.ObjectType(),
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
// honoured in full. An unconstrained permit allows each action but
// ReadMailboxAction, which refuses every permit
// (TestReadMailboxActionRefusesAnUnconstrainedPermit).
func TestMessagingActionsRefuseConstrainedPermits(t *testing.T) {
	for _, tc := range []struct {
		action auth.ActionObject
		plain  bool
	}{
		{&SendAction{Action: auth.NewAction(astral.GenerateIdentity())}, true},
		{&ReceiveAction{Action: auth.NewAction(astral.GenerateIdentity())}, true},
		{&HostMailboxAction{Action: auth.NewAction(astral.GenerateIdentity()), MailboxID: astral.GenerateIdentity()}, true},
		{&ReadMailboxAction{Action: auth.NewAction(astral.GenerateIdentity()), MailboxID: astral.GenerateIdentity()}, false},
	} {
		t.Run(tc.action.ObjectType(), func(t *testing.T) {
			plain := &auth.Permit{Action: astral.String8(tc.action.ObjectType())}
			if plain.Allows(tc.action) != tc.plain {
				t.Fatalf("an unconstrained permit: allows %v, want %v", !tc.plain, tc.plain)
			}

			constraints := astral.NewBundle()
			if err := constraints.Append(astral.NewError("any constraint at all")); err != nil {
				t.Fatalf("append constraint: %v", err)
			}

			narrowed := &auth.Permit{
				Action:      astral.String8(tc.action.ObjectType()),
				Constraints: constraints,
			}
			if narrowed.Allows(tc.action) {
				t.Fatal("a permit carrying constraints must be refused: nothing evaluates them")
			}
		})
	}
}

// TestReadMailboxActionRefusesAnUnconstrainedPermit: no permit carries a
// delegated read. A permit naming the action with no constraint is refused
// whatever its Delegation, and whether its bundle is absent or empty, so no
// contract link passes the read to a reader.
func TestReadMailboxActionRefusesAnUnconstrainedPermit(t *testing.T) {
	read := &ReadMailboxAction{Action: auth.NewAction(astral.GenerateIdentity()), MailboxID: astral.GenerateIdentity()}
	action := astral.String8(read.ObjectType())

	for name, p := range map[string]*auth.Permit{
		"no bundle":    {Action: action},
		"empty bundle": {Action: action, Constraints: astral.NewBundle()},
		"delegable":    {Action: action, Delegation: 255},
	} {
		t.Run(name, func(t *testing.T) {
			if p.Allows(read) {
				t.Fatal("an unconstrained permit allows a delegated read")
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
		"read_mailbox": {&ReadMailboxAction{Action: auth.NewAction(actor), MailboxID: other}, "MailboxID"},
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

	b, err = json.Marshal(&ReadMailboxAction{Action: auth.NewAction(actor), MailboxID: other})
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var read ReadMailboxAction
	if err = json.Unmarshal(b, &read); err != nil {
		t.Fatalf("unmarshal read_mailbox: %v", err)
	}
	if !read.Actor().IsEqual(actor) || !read.MailboxID.IsEqual(other) {
		t.Fatalf("read_mailbox: got actor %v mailbox %v", read.Actor(), read.MailboxID)
	}
}
