package channel

import (
	"bytes"
	"strings"
	"testing"
)

// TestJSONReceiver_RejectsMalformedEnvelope pins the strictness reaching the transport.
//
// Receive decoded each line into an astral.JSONAdapter and then skipped the payload when
// Object was nil, so a misspelled "Object" key yielded a zero-valued object of the named
// type with no error — the same silent corruption interfaceValue had, one level up and on
// every JSON op reply rather than only on interface-typed slots. The check lives on
// JSONAdapter, so closing it there closes this route too.
func TestJSONReceiver_RejectsMalformedEnvelope(t *testing.T) {
	tests := []struct {
		name string
		line string
		want string
	}{
		{"Object misspelled", `{"Type":"uint32","Obejct":42}`, "excess fields in json envelope"},
		{"excess key beside a valid pair", `{"Type":"uint32","Object":42,"Extra":1}`, "excess fields in json envelope"},
		{"case-colliding payload keys", `{"Type":"uint32","Object":42,"OBJECT":7}`, "duplicate fields due to case insensitivity"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			rcv := NewJSONReceiver(bytes.NewBufferString(test.line + "\n"))

			obj, err := rcv.Receive()
			if err == nil {
				t.Fatalf("received a malformed envelope without error, got %#v", obj)
			}
			if !strings.Contains(err.Error(), test.want) {
				t.Fatalf("error: want %q, got %v", test.want, err)
			}
		})
	}
}

// TestJSONReceiver_AcceptsConformingEnvelope pins well-formed traffic still arriving:
// canonical form, case-folded keys, and the payloadless control frame whose "Object" key
// carries null. Nothing astral-go, astral-py or astral-js emits is refused.
func TestJSONReceiver_AcceptsConformingEnvelope(t *testing.T) {
	tests := []struct {
		name string
		line string
		want string
	}{
		{"canonical", `{"Type":"uint32","Object":42}`, "uint32"},
		{"lowercase keys", `{"type":"uint32","object":42}`, "uint32"},
		{"reversed key order", `{"Object":42,"Type":"uint32"}`, "uint32"},
		{"null payload control frame", `{"Type":"ack","Object":null}`, "ack"},
		{"empty payload", `{"Type":"ack","Object":{}}`, "ack"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			rcv := NewJSONReceiver(bytes.NewBufferString(test.line + "\n"))

			obj, err := rcv.Receive()
			if err != nil {
				t.Fatalf("Receive returned error: %v (from %s)", err, test.line)
			}
			if got := obj.ObjectType(); got != test.want {
				t.Fatalf("ObjectType: want %q, got %q", test.want, got)
			}
		})
	}
}
