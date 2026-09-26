package messaging

import (
	"bytes"
	"encoding/json"
	"reflect"
	"testing"

	"github.com/astralp2p/astral-go/api/auth"
	"github.com/astralp2p/astral-go/astral"
	"github.com/astralp2p/astral-go/astral/channel"
)

// binaryRoundTrip encodes src with its type tag and decodes it by that tag, as a
// binary channel peer reads it.
func binaryRoundTrip(t *testing.T, src astral.Object) astral.Object {
	t.Helper()

	data, err := astral.EncodeBytes(src)
	if err != nil {
		t.Fatalf("encode: %v", err)
	}

	obj, _, err := astral.Decode(bytes.NewReader(data))
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	return obj
}

// jsonRoundTrip sends src on a JSON channel and receives it back, which is the
// path an op under out=json takes.
func jsonRoundTrip(t *testing.T, src astral.Object) astral.Object {
	t.Helper()

	var buf bytes.Buffer
	if err := channel.NewJSONSender(&buf).Send(src); err != nil {
		t.Fatalf("send: %v", err)
	}

	obj, err := channel.NewJSONReceiver(bytes.NewReader(buf.Bytes())).Receive()
	if err != nil {
		t.Fatalf("receive %q: %v", buf.String(), err)
	}
	return obj
}

// assertSameWire fails unless got is want's Go type and encodes to want's
// bytes. The binary form writes every field, a nil flag for every pointer and
// nanoseconds for every instant, so equal bytes are equal values.
func assertSameWire(t *testing.T, want, got astral.Object) {
	t.Helper()

	if reflect.TypeOf(got) != reflect.TypeOf(want) {
		t.Fatalf("want %T, got %T", want, got)
	}

	w, err := astral.EncodeBytes(want)
	if err != nil {
		t.Fatalf("encode want: %v", err)
	}
	g, err := astral.EncodeBytes(got)
	if err != nil {
		t.Fatalf("encode got: %v", err)
	}
	if !bytes.Equal(w, g) {
		t.Fatalf("the value changed on the way:\nwant %x\n got %x", w, g)
	}
}

// jsonFields answers the top-level members v marshals to.
func jsonFields(t *testing.T, v any) map[string]json.RawMessage {
	t.Helper()

	data, err := json.Marshal(v)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var fields map[string]json.RawMessage
	if err = json.Unmarshal(data, &fields); err != nil {
		t.Fatalf("unmarshal %s: %v", data, err)
	}
	return fields
}

func ptrString32(s string) *astral.String32 {
	v := astral.String32(s)
	return &v
}

// wireCases is every object this package adds beside the moved mail types,
// filled and empty. Empty covers both a nil slice and a zero-length one: a
// decoder that tells them apart, or chokes on either, fails here.
func wireCases() map[string]astral.Object {
	full := sampleStoredMessage()
	ref := &MessageRef{Box: BoxInbox, ID: NewMessageID()}
	read := &ReadMessage{
		Envelope:  full.Envelope(),
		Content:   ptrString32("the index is rebuilt"),
		ChildIDs:  []MessageID{NewMessageID(), NewMessageID()},
		Truncated: true,
	}
	actor, other := astral.GenerateIdentity(), astral.GenerateIdentity()

	return map[string]astral.Object{
		"envelope":                         full.Envelope(),
		"envelope zero":                    &Envelope{},
		"message_ref":                      ref,
		"message_ref zero":                 &MessageRef{},
		"identity_credential":              sampleCredential(),
		"identity_credential zero":         &IdentityCredential{},
		"identity_info":                    sampleIdentityInfo(),
		"identity_info zero":               &IdentityInfo{},
		"send_message_request":             &SendMessageRequest{To: "scout", Content: "hello", ParentID: NewMessageID()},
		"send_message_request zero":        &SendMessageRequest{},
		"read_messages_request":            &ReadMessagesRequest{Refs: []*MessageRef{ref, {Box: BoxOutbox, ID: NewMessageID()}}, Children: ChildrenFull, MaxChildren: 3},
		"read_messages_request mailbox":    &ReadMessagesRequest{Refs: []*MessageRef{ref}, Children: ChildrenNone, Mailbox: other},
		"read_messages_request nil refs":   &ReadMessagesRequest{},
		"read_messages_request empty refs": &ReadMessagesRequest{Refs: []*MessageRef{}},
		"read_message":                     read,
		"read_message withheld body":       &ReadMessage{Envelope: full.Envelope(), ChildIDs: []MessageID{}},
		"read_message empty body":          &ReadMessage{Envelope: full.Envelope(), Content: ptrString32("")},
		"read_message zero":                &ReadMessage{},
		"read_messages_result":             &ReadMessagesResult{Messages: []*ReadMessage{read}, Replies: []*ReadMessage{read, read}, NotFound: []*MessageRef{ref}},
		"read_messages_result nil":         &ReadMessagesResult{},
		"read_messages_result empty":       &ReadMessagesResult{Messages: []*ReadMessage{}, Replies: []*ReadMessage{}, NotFound: []*MessageRef{}},
		"wait_result":                      &WaitResult{Messages: []*Envelope{full.Envelope()}, NextSince: 42, TimedOut: false, Granted: astral.Duration(2e9), Waited: astral.Duration(1500)},
		"wait_result timed out":            &WaitResult{Messages: []*Envelope{}, NextSince: 7, TimedOut: true, Granted: astral.Duration(2e9), Waited: astral.Duration(2e9)},
		"wait_result zero":                 &WaitResult{},
		"archive_result":                   &ArchiveResult{Changed: true},
		"archive_result zero":              &ArchiveResult{},
		"send_action":                      &SendAction{Action: auth.NewAction(actor), ToID: other},
		"send_action zero":                 &SendAction{},
		"receive_action":                   &ReceiveAction{Action: auth.NewAction(actor), FromID: other},
		"receive_action zero":              &ReceiveAction{},
		"host_mailbox_action":              &HostMailboxAction{Action: auth.NewAction(actor), MailboxID: other},
		"host_mailbox_action zero":         &HostMailboxAction{},
		"read_mailbox_action":              &ReadMailboxAction{Action: auth.NewAction(actor), MailboxID: other},
		"read_mailbox_action zero":         &ReadMailboxAction{},
	}
}

func TestEveryObjectRoundTripsThroughBinary(t *testing.T) {
	for name, src := range wireCases() {
		t.Run(name, func(t *testing.T) {
			assertSameWire(t, src, binaryRoundTrip(t, src))
		})
	}
}

func TestEveryObjectRoundTripsThroughJSON(t *testing.T) {
	for name, src := range wireCases() {
		t.Run(name, func(t *testing.T) {
			assertSameWire(t, src, jsonRoundTrip(t, src))
		})
	}
}

// The names are spelled out rather than read back from the types, because a
// test that reads the same method as the code cannot catch the method being
// wrong. Every one is registered: a receiver materializes it by this name.
func TestObjectTypesAreTheMessagingNames(t *testing.T) {
	for want, proto := range map[string]astral.Object{
		"messaging.message_id":              &MessageID{},
		"messaging.message":                 &Message{},
		"messaging.receipt":                 &Receipt{},
		"messaging.stored_message":          &StoredMessage{},
		"messaging.envelope":                &Envelope{},
		"messaging.message_ref":             &MessageRef{},
		"messaging.identity_credential":     &IdentityCredential{},
		"messaging.identity_info":           &IdentityInfo{},
		"messaging.send_message_request":    &SendMessageRequest{},
		"messaging.read_messages_request":   &ReadMessagesRequest{},
		"messaging.read_message":            &ReadMessage{},
		"messaging.read_messages_result":    &ReadMessagesResult{},
		"messaging.wait_result":             &WaitResult{},
		"messaging.archive_result":          &ArchiveResult{},
		"mod.messaging.send_action":         &SendAction{},
		"mod.messaging.receive_action":      &ReceiveAction{},
		"mod.messaging.host_mailbox_action": &HostMailboxAction{},
		"mod.messaging.read_mailbox_action": &ReadMailboxAction{},
	} {
		if got := proto.ObjectType(); got != want {
			t.Errorf("%T: object type %q, want %q", proto, got, want)
		}
		if got := astral.New(want); reflect.TypeOf(got) != reflect.TypeOf(proto) {
			t.Errorf("%q materializes as %T, want %T", want, got, proto)
		}
	}
}
