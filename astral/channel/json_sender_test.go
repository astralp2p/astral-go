package channel

import (
	"bytes"
	"math"
	"testing"

	"github.com/astralp2p/astral-go/astral"
)

// A Send whose object fails to encode writes nothing: the object is marshaled
// before the line is encoded and written, so the next object decodes.
func TestJSONSender_FailedSendWritesNothing(t *testing.T) {
	var buf bytes.Buffer
	s := NewJSONSender(&buf)

	nan := astral.Float32(math.NaN())
	if err := s.Send(&nan); err == nil {
		t.Fatal("Send encoded a NaN float32 as JSON")
	}
	if buf.Len() != 0 {
		t.Fatalf("a failed Send wrote %d bytes", buf.Len())
	}

	next := astral.String8("next")
	if err := s.Send(&next); err != nil {
		t.Fatalf("Send: %v", err)
	}

	got, err := NewJSONReceiver(&buf).Receive()
	if err != nil {
		t.Fatalf("Receive after a failed Send: %v", err)
	}
	if s, ok := got.(*astral.String8); !ok || *s != next {
		t.Fatalf("received %T %v after a failed Send, want string8 %q", got, got, next)
	}
}
