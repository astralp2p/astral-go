package pub_test

import (
	"bytes"
	"encoding/json"
	"reflect"
	"testing"
	"time"

	_ "github.com/astralp2p/astral-go"
	"github.com/astralp2p/astral-go/api/crypto"
	"github.com/astralp2p/astral-go/api/ether"
	"github.com/astralp2p/astral-go/api/events"
	"github.com/astralp2p/astral-go/api/ip"
	"github.com/astralp2p/astral-go/api/nearby"
	"github.com/astralp2p/astral-go/api/nodes"
	"github.com/astralp2p/astral-go/astral"
	"github.com/astralp2p/astral-go/astral/channel"
)

// A value in an interface-typed slot is JSON-encoded as a nested "Type"/"Object"
// container, and a null in the slot is the absent value
// (.ai/system/topics/json-encoding.md). The four types below carry such a slot.
//
// The documents are the JSON examples of their astral-docs type pages
// (.ai/system/protocols/ether/types/*.md, events/types/mod.events.event.md). One
// change: the pages give the inner mod.nearby.status "Attachments" as {}, and a
// Bundle encodes as a list (astral/bundle.go), so here it carries a one-element
// list. The test lives at the repo root because the examples' payload types come
// from other api packages.

var objectSlotSpecExamples = map[string]string{
	"mod.ether.broadcast": `{
  "Type": "mod.ether.broadcast",
  "Object": {
    "Timestamp": "2026-06-19T10:00:00Z",
    "Source": "02bef8840eb35ef2ae3c83c07cb5779278904f99cb4103f71e37cc69931ae5e15f",
    "Object": {
      "Type": "mod.nearby.status",
      "Object": {
        "Identity": "02bef8840eb35ef2ae3c83c07cb5779278904f99cb4103f71e37cc69931ae5e15f",
        "Attachments": [{"Type": "string8", "Object": "hi"}]
      }
    }
  }
}`,
	"mod.ether.signed_broadcast": `{
  "Type": "mod.ether.signed_broadcast",
  "Object": {
    "Timestamp": "2026-06-19T10:00:00Z",
    "Source": "02bef8840eb35ef2ae3c83c07cb5779278904f99cb4103f71e37cc69931ae5e15f",
    "Object": {
      "Type": "mod.nearby.status",
      "Object": {
        "Identity": "02bef8840eb35ef2ae3c83c07cb5779278904f99cb4103f71e37cc69931ae5e15f",
        "Attachments": [{"Type": "string8", "Object": "hi"}]
      }
    },
    "Signature": {
      "Scheme": "asn1",
      "Data": "MEUCIQDg+5p9aT4q2QzVKbS9N5wXYrFh3vUaQ7E2lY0bX9lSAgIgex8="
    }
  }
}`,
	"mod.ether.events.broadcast_received": `{
  "Type": "mod.ether.events.broadcast_received",
  "Object": {
    "SourceIP": "192.168.1.42",
    "Object": {
      "Type": "mod.nearby.status",
      "Object": {
        "Identity": "02bef8840eb35ef2ae3c83c07cb5779278904f99cb4103f71e37cc69931ae5e15f",
        "Attachments": [{"Type": "string8", "Object": "hi"}]
      }
    }
  }
}`,
	"mod.events.event": `{
  "Type": "mod.events.event",
  "Object": {
    "ID": "a1b2c3d4e5f60718",
    "SourceID": "02bef8840eb35ef2ae3c83c07cb5779278904f99cb4103f71e37cc69931ae5e15f",
    "Timestamp": "2026-06-19T10:00:00Z",
    "Data": {
      "Type": "mod.nodes.link_created_event",
      "Object": {
        "RemoteIdentity": "037f990e61acee8a7697966afd29dd88f3b1f8a7b14d625c4f8742bd952003a590",
        "LinkID": "7c1a93b50f2e4d18",
        "LinkCount": 1
      }
    }
  }
}`,
}

// slotOf returns the interface-typed slot of one of the four types.
func slotOf(t *testing.T, o astral.Object) astral.Object {
	t.Helper()
	switch v := o.(type) {
	case *ether.Broadcast:
		return v.Object
	case *ether.SignedBroadcast:
		return v.Object
	case *ether.EventBroadcastReceived:
		return v.Object
	case *events.Event:
		return v.Data
	}
	t.Fatalf("no object slot on %T", o)
	return nil
}

func receiveOne(t *testing.T, doc []byte) astral.Object {
	t.Helper()
	o, err := channel.NewJSONReceiver(bytes.NewReader(doc)).Receive()
	if err != nil {
		t.Fatalf("receive: %v", err)
	}
	return o
}

func sendOne(t *testing.T, o astral.Object) []byte {
	t.Helper()
	var buf bytes.Buffer
	if err := channel.NewJSONSender(&buf).Send(o); err != nil {
		t.Fatalf("send: %v", err)
	}
	return buf.Bytes()
}

func parseJSON(t *testing.T, b []byte) any {
	t.Helper()
	var v any
	if err := json.Unmarshal(b, &v); err != nil {
		t.Fatalf("parse %s: %v", b, err)
	}
	return v
}

func TestObjectSlotSpecExamplesRoundTrip(t *testing.T) {
	wantSlot := map[string]reflect.Type{
		"mod.ether.broadcast":                 reflect.TypeOf(&nearby.Status{}),
		"mod.ether.signed_broadcast":          reflect.TypeOf(&nearby.Status{}),
		"mod.ether.events.broadcast_received": reflect.TypeOf(&nearby.Status{}),
		"mod.events.event":                    reflect.TypeOf(&nodes.LinkCreatedEvent{}),
	}

	for name, doc := range objectSlotSpecExamples {
		t.Run(name, func(t *testing.T) {
			o := receiveOne(t, []byte(doc))
			if o.ObjectType() != name {
				t.Fatalf("received %s, want %s", o.ObjectType(), name)
			}
			if got := reflect.TypeOf(slotOf(t, o)); got != wantSlot[name] {
				t.Fatalf("slot holds %v, want %v", got, wantSlot[name])
			}

			out := sendOne(t, o)
			if !reflect.DeepEqual(parseJSON(t, out), parseJSON(t, []byte(doc))) {
				t.Errorf("re-encoded document differs from the spec example:\n got %s\nwant %s", out, doc)
			}
		})
	}
}

func sampleObjectSlotValues() []astral.Object {
	hello := astral.String8("hello")
	at := astral.Time(time.Unix(1700000000, 123456789))
	b := ether.Broadcast{Timestamp: at, Source: astral.GenerateIdentity(), Object: &hello}

	return []astral.Object{
		&b,
		&ether.SignedBroadcast{Broadcast: b, Signature: &crypto.Signature{Scheme: "asn1", Data: []byte{1, 2, 3}}},
		&ether.EventBroadcastReceived{SourceIP: ip.IP{192, 168, 1, 42}, Object: &hello},
		&events.Event{ID: astral.NewNonce(), SourceID: astral.GenerateIdentity(), Timestamp: at, Data: &hello},
	}
}

// slotKey is the payload key holding the interface-typed slot.
func slotKey(o astral.Object) string {
	if _, ok := o.(*events.Event); ok {
		return "Data"
	}
	return "Object"
}

func TestObjectSlotSendReceive(t *testing.T) {
	for _, sent := range sampleObjectSlotValues() {
		t.Run(sent.ObjectType(), func(t *testing.T) {
			out := sendOne(t, sent)

			var env struct {
				Type   string
				Object map[string]json.RawMessage
			}
			if err := json.Unmarshal(out, &env); err != nil {
				t.Fatalf("parse %s: %v", out, err)
			}
			slot := parseJSON(t, env.Object[slotKey(sent)])
			want := map[string]any{"Type": "string8", "Object": "hello"}
			if !reflect.DeepEqual(slot, want) {
				t.Fatalf("slot encodes as %v, want %v (document %s)", slot, want, out)
			}

			got := receiveOne(t, out)
			if v, ok := slotOf(t, got).(*astral.String8); !ok || *v != "hello" {
				t.Fatalf("slot decodes as %#v, want *astral.String8 \"hello\"", slotOf(t, got))
			}
			if again := sendOne(t, got); !bytes.Equal(again, out) {
				t.Errorf("re-encoded document differs:\n got %s\nwant %s", again, out)
			}
		})
	}
}

func TestObjectSlotAbsentIsNull(t *testing.T) {
	for _, sent := range []astral.Object{
		&ether.Broadcast{Source: astral.GenerateIdentity()},
		&ether.SignedBroadcast{Broadcast: ether.Broadcast{Source: astral.GenerateIdentity()}},
		&ether.EventBroadcastReceived{SourceIP: ip.IP{127, 0, 0, 1}},
		&events.Event{SourceID: astral.GenerateIdentity()},
	} {
		t.Run(sent.ObjectType(), func(t *testing.T) {
			out := sendOne(t, sent)

			var env struct{ Object map[string]json.RawMessage }
			if err := json.Unmarshal(out, &env); err != nil {
				t.Fatalf("parse %s: %v", out, err)
			}
			raw, ok := env.Object[slotKey(sent)]
			if !ok || string(raw) != "null" {
				t.Fatalf("absent slot encodes as %q, want null (document %s)", raw, out)
			}

			if got := slotOf(t, receiveOne(t, out)); got != nil {
				t.Errorf("absent slot decodes as %#v, want nil", got)
			}
		})
	}
}

// SignedBroadcast keeps the embedded Broadcast's keys flat beside Signature, as
// encoding/json writes an embedded struct; a nested "Broadcast" key is rejected.
func TestSignedBroadcastJSONIsFlat(t *testing.T) {
	sent := sampleObjectSlotValues()[1]
	out := sendOne(t, sent)

	var env struct{ Object map[string]json.RawMessage }
	if err := json.Unmarshal(out, &env); err != nil {
		t.Fatalf("parse %s: %v", out, err)
	}
	for _, k := range []string{"Timestamp", "Source", "Object", "Signature"} {
		if _, ok := env.Object[k]; !ok {
			t.Errorf("payload lacks key %q: %s", k, out)
		}
	}
	if len(env.Object) != 4 {
		t.Errorf("payload has %d keys, want 4: %s", len(env.Object), out)
	}

	nested := `{"Type":"mod.ether.signed_broadcast","Object":{"Broadcast":{"Timestamp":"2026-06-19T10:00:00Z"}}}`
	if _, err := channel.NewJSONReceiver(bytes.NewReader([]byte(nested))).Receive(); err == nil {
		t.Error("nested Broadcast key decodes without error")
	}
}
