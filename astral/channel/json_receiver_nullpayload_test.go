package channel

import (
	"bytes"
	"testing"
)

// TestJSONReceiver_KnownType_NullPayload_NoNilObject pins the null-payload panic:
// a KNOWN type carrying an explicit null Object must decode to a non-nil astral.Object.
//
// Before the fix, Receive ran json.Unmarshal([]byte("null"), &object) — a pointer to
// the astral.Object interface. encoding/json nils the pointee on a JSON null literal,
// discarding the non-nil value astral.New had just created, so Receive returned
// (nil, nil) and the caller's obj.ObjectType() dereferenced a nil interface (SIGSEGV).
//
// Ack is a registered payloadless object (EmptyObject) — the canonical shape of a
// control frame like EOS whose wire form is {"Type":"ack","Object":null}.
func TestJSONReceiver_KnownType_NullPayload_NoNilObject(t *testing.T) {
	stream := `{"Type":"ack","Object":null}` + "\n"
	rcv := NewJSONReceiver(bytes.NewBufferString(stream))

	obj, err := rcv.Receive()
	if err != nil {
		t.Fatalf("Receive returned error: %v", err)
	}
	if obj == nil {
		t.Fatal("Receive returned a nil object for a null payload — caller obj.ObjectType() would panic")
	}
	// The reported crash site: this must not panic on a nil interface.
	if got := obj.ObjectType(); got != "ack" {
		t.Fatalf("ObjectType: want %q, got %q", "ack", got)
	}
}
