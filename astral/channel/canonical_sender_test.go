package channel

import (
	"bytes"
	"strings"
	"testing"

	"github.com/astralp2p/astral-go/astral"
)

// A Send whose object fails to encode writes nothing. The stamp and type used
// to go out before the payload was encoded, so a failed encode left an orphan
// stamp and type on the stream and every later object decoded from the wrong
// offset.
func TestCanonicalSender_FailedSendWritesNothing(t *testing.T) {
	var buf bytes.Buffer
	s := NewCanonicalSender(&buf)

	tooLong := astral.String16(strings.Repeat("q", 1<<16))
	if err := s.Send(&tooLong); err == nil {
		t.Fatal("Send encoded a string16 longer than 65535 bytes")
	}
	if buf.Len() != 0 {
		t.Fatalf("a failed Send wrote %d bytes", buf.Len())
	}

	next := astral.String8("next")
	if err := s.Send(&next); err != nil {
		t.Fatalf("Send: %v", err)
	}

	got, err := NewCanonicalReceiver(&buf).Receive()
	if err != nil {
		t.Fatalf("Receive after a failed Send: %v", err)
	}
	if s, ok := got.(*astral.String8); !ok || *s != next {
		t.Fatalf("received %T %v after a failed Send, want string8 %q", got, got, next)
	}
}
