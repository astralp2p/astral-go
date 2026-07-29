package channel

import (
	"bytes"
	"testing"

	"github.com/astralp2p/astral-go/astral"
)

// TestJSONChannel_ErrorMessage_RoundTrip pins the reply path every streaming op uses to
// report a failure: astral.Err(err) sent on a JSON channel and read back by a Go peer.
//
// ErrorMessage.UnmarshalJSON previously recursed into itself, so any Go peer reading an
// error_message off an astral.json.v1 channel died with a stack overflow — reachable by a
// remote peer that merely reports an error.
func TestJSONChannel_ErrorMessage_RoundTrip(t *testing.T) {
	var buf bytes.Buffer

	sender := NewJSONSender(&buf)
	if err := sender.Send(astral.NewError("boom")); err != nil {
		t.Fatalf("send: %v", err)
	}

	recv := NewJSONReceiver(&buf)
	got, err := recv.Receive()
	if err != nil {
		t.Fatalf("receive: %v", err)
	}

	msg, ok := got.(*astral.ErrorMessage)
	if !ok {
		t.Fatalf("want *astral.ErrorMessage, got %T", got)
	}
	if msg.Error() != "boom" {
		t.Fatalf("Error: want %q, got %q", "boom", msg.Error())
	}
}

// TestJSONReceiver_ErrorMessage_Wire pins the decode against the wire form literally,
// independent of what this module's sender happens to emit — the bytes an SDK peer in
// another language puts on the channel.
func TestJSONReceiver_ErrorMessage_Wire(t *testing.T) {
	stream := `{"Type":"error_message","Object":"boom"}` + "\n"

	got, err := NewJSONReceiver(bytes.NewBufferString(stream)).Receive()
	if err != nil {
		t.Fatalf("receive: %v", err)
	}

	msg, ok := got.(*astral.ErrorMessage)
	if !ok {
		t.Fatalf("want *astral.ErrorMessage, got %T", got)
	}
	if msg.Error() != "boom" {
		t.Fatalf("Error: want %q, got %q", "boom", msg.Error())
	}
}
