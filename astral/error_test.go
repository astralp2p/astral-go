package astral

import (
	"encoding/json"
	"testing"
)

// TestErrorMessage_UnmarshalJSON_DecodesMessage pins the message string surviving a
// JSON decode.
//
// The method previously read json.Unmarshal(bytes, &msg). The receiver is already
// *ErrorMessage, so &msg is **ErrorMessage; encoding/json dereferences it, finds
// *ErrorMessage implements json.Unmarshaler, and re-invokes this method with the same
// bytes — unbounded recursion, killed by the runtime as a stack overflow. A crash
// fails this test by taking the binary down, which is the intent.
func TestErrorMessage_UnmarshalJSON_DecodesMessage(t *testing.T) {
	var msg ErrorMessage

	if err := msg.UnmarshalJSON([]byte(`"boom"`)); err != nil {
		t.Fatalf("UnmarshalJSON returned error: %v", err)
	}
	if got := msg.Error(); got != "boom" {
		t.Fatalf("Error: want %q, got %q", "boom", got)
	}
}

// TestErrorMessage_JSON_RoundTrip pins the encode and decode halves agreeing: the wire
// form is the bare message string, as MarshalJSON emits it.
func TestErrorMessage_JSON_RoundTrip(t *testing.T) {
	encoded, err := json.Marshal(NewError("boom"))
	if err != nil {
		t.Fatalf("Marshal returned error: %v", err)
	}
	if want := `"boom"`; string(encoded) != want {
		t.Fatalf("Marshal: want %s, got %s", want, encoded)
	}

	var decoded ErrorMessage
	if err = json.Unmarshal(encoded, &decoded); err != nil {
		t.Fatalf("Unmarshal returned error: %v", err)
	}
	if got := decoded.Error(); got != "boom" {
		t.Fatalf("Error: want %q, got %q", "boom", got)
	}
}

// TestErrorMessage_UnmarshalJSON_RejectsNonString pins the decode reporting a type
// mismatch rather than accepting it. The recursive form could not reject anything: it
// never reached a concrete type to check against.
func TestErrorMessage_UnmarshalJSON_RejectsNonString(t *testing.T) {
	var msg ErrorMessage

	if err := msg.UnmarshalJSON([]byte(`{"err":"boom"}`)); err == nil {
		t.Fatal("UnmarshalJSON accepted a JSON object, want a type error")
	}
}

// TestErrorMessage_UnmarshalJSON_Empty pins the empty message decoding cleanly — an
// Error carrying no text is well-formed on the wire.
func TestErrorMessage_UnmarshalJSON_Empty(t *testing.T) {
	msg := ErrorMessage{err: "stale"}

	if err := msg.UnmarshalJSON([]byte(`""`)); err != nil {
		t.Fatalf("UnmarshalJSON returned error: %v", err)
	}
	if got := msg.Error(); got != "" {
		t.Fatalf("Error: want %q, got %q", "", got)
	}
}
