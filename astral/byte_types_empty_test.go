package astral

import (
	"io"
	"testing"
	"time"
)

// An empty payload is not an edge case here: Ack, EOS and Nil all embed EmptyObject and
// serialise to zero bytes, and astral/channel frames every object as a bytes32, so the
// framework's three terminators are exactly the values that take this path.
//
// The hazard is io.Pipe, which in-process routing uses in both directions. A pipe Write
// blocks until a reader takes the bytes — and Go's implementation runs its handoff at
// least once even for an empty slice, so a zero-length Write parks like any other. The
// peer never issues the matching Read, because ReadFrom of a zero length returns without
// touching the reader. Over TCP the same call is a harmless no-op, which is why this
// never showed up in normal use.

const emptyWriteWait = 2 * time.Second

// pipeRoundTrip writes value through an io.Pipe and reads it back, bounded. It returns
// false if either side is still parked when the bound expires.
func pipeRoundTrip(t *testing.T, write func(io.Writer) error, read func(io.Reader) error) bool {
	t.Helper()

	pr, pw := io.Pipe()
	t.Cleanup(func() { pr.Close(); pw.Close() })

	wrote := make(chan error, 1)
	go func() { wrote <- write(pw) }()

	done := make(chan error, 1)
	go func() { done <- read(pr) }()

	for range 2 {
		select {
		case err := <-wrote:
			if err != nil {
				t.Fatalf("write: %v", err)
			}
			wrote = nil
		case err := <-done:
			if err != nil {
				t.Fatalf("read: %v", err)
			}
			done = nil
		case <-time.After(emptyWriteWait):
			return false
		}
	}
	return true
}

// The defect, at the narrowest reproduction: one empty bytes32 across a pipe.
func TestBytes_EmptyPayloadRoundTripsOverAPipe(t *testing.T) {
	ok := pipeRoundTrip(t,
		func(w io.Writer) error { _, err := Bytes32(nil).WriteTo(w); return err },
		func(r io.Reader) error { var b Bytes32; _, err := b.ReadFrom(r); return err },
	)
	if !ok {
		t.Errorf("an empty bytes32 did not cross an io.Pipe within %v: the zero-length "+
			"Write parks and the reader never issues a matching Read", emptyWriteWait)
	}
}

// All four widths share the shape, so all four are pinned.
func TestBytes_EveryWidthCrossesAPipeEmpty(t *testing.T) {
	for _, c := range []struct {
		name  string
		write func(io.Writer) error
		read  func(io.Reader) error
	}{
		{"bytes8", func(w io.Writer) error { _, err := Bytes8(nil).WriteTo(w); return err },
			func(r io.Reader) error { var b Bytes8; _, err := b.ReadFrom(r); return err }},
		{"bytes16", func(w io.Writer) error { _, err := Bytes16(nil).WriteTo(w); return err },
			func(r io.Reader) error { var b Bytes16; _, err := b.ReadFrom(r); return err }},
		{"bytes32", func(w io.Writer) error { _, err := Bytes32(nil).WriteTo(w); return err },
			func(r io.Reader) error { var b Bytes32; _, err := b.ReadFrom(r); return err }},
		{"bytes64", func(w io.Writer) error { _, err := Bytes64(nil).WriteTo(w); return err },
			func(r io.Reader) error { var b Bytes64; _, err := b.ReadFrom(r); return err }},
	} {
		t.Run(c.name, func(t *testing.T) {
			if !pipeRoundTrip(t, c.write, c.read) {
				t.Errorf("empty %s did not cross an io.Pipe within %v", c.name, emptyWriteWait)
			}
		})
	}
}

// The count must not change: an empty payload is the length prefix and nothing else, and
// a caller that trusted the old count would otherwise see it shift.
func TestBytes_EmptyPayloadStillReportsItsPrefix(t *testing.T) {
	for _, c := range []struct {
		name string
		want int64
		fn   func(io.Writer) (int64, error)
	}{
		{"bytes8", 1, func(w io.Writer) (int64, error) { return Bytes8(nil).WriteTo(w) }},
		{"bytes16", 2, func(w io.Writer) (int64, error) { return Bytes16(nil).WriteTo(w) }},
		{"bytes32", 4, func(w io.Writer) (int64, error) { return Bytes32(nil).WriteTo(w) }},
		{"bytes64", 8, func(w io.Writer) (int64, error) { return Bytes64(nil).WriteTo(w) }},
	} {
		n, err := c.fn(io.Discard)
		if err != nil {
			t.Errorf("%s: %v", c.name, err)
			continue
		}
		if n != c.want {
			t.Errorf("%s: want %d bytes reported, got %d", c.name, c.want, n)
		}
	}
}
