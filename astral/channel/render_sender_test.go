package channel

import (
	"bytes"
	"strings"
	"testing"

	"github.com/astralp2p/astral-go/astral"
	"github.com/astralp2p/astral-go/astral/fmt"
)

// Send renders the whole line before it writes, and rendering has no error
// path, so an object the binary encoding rejects still goes out as one
// complete line and the next object's line follows intact.
func TestRenderSender_SendWritesWholeLines(t *testing.T) {
	var buf bytes.Buffer
	s := NewRenderSender(&buf)

	tooLong := astral.String16(strings.Repeat("q", 1<<16))
	if err := s.Send(&tooLong); err != nil {
		t.Fatalf("Send: %v", err)
	}
	if want := fmt.Sprint(&tooLong) + "\n"; buf.String() != want {
		t.Fatalf("Send wrote %d bytes, want one %d-byte line", buf.Len(), len(want))
	}
	buf.Reset()

	next := astral.String8("next")
	if err := s.Send(&next); err != nil {
		t.Fatalf("Send: %v", err)
	}
	if want := fmt.Sprint(&next) + "\n"; buf.String() != want {
		t.Fatalf("Send wrote %q, want %q", buf.String(), want)
	}
}
