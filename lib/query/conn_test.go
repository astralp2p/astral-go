package query

import (
	"errors"
	"io"
	"testing"

	"github.com/astralp2p/astral-go/astral"
)

// Conn pairs a Reader with a WriteCloser and carries the two identities of the
// link. Its one behaviour beyond plumbing: a read error tears down the write
// side, so a half-dead link cannot linger.

// closeRecorder records whether Close was called.
type closeRecorder struct {
	io.Writer
	closed bool
}

func (c *closeRecorder) Close() error {
	c.closed = true
	return nil
}

var errRead = errors.New("read failed")

// failingReader returns errRead on every read.
type failingReader struct{}

func (failingReader) Read([]byte) (int, error) { return 0, errRead }

func TestConn_CarriesIdentitiesAndDirection(t *testing.T) {
	local, remote := astral.GenerateIdentity(), astral.GenerateIdentity()

	for _, outbound := range []bool{true, false} {
		conn := NewConn(local, remote, &closeRecorder{Writer: io.Discard}, failingReader{}, outbound)

		if !conn.LocalIdentity().IsEqual(local) {
			t.Fatal("LocalIdentity: want the supplied local identity")
		}
		if !conn.RemoteIdentity().IsEqual(remote) {
			t.Fatal("RemoteIdentity: want the supplied remote identity")
		}

		// Outbound is on the concrete type: astral.Conn does not carry direction
		if conn.(*Conn).Outbound() != outbound {
			t.Fatalf("Outbound: want %v, got %v", outbound, conn.(*Conn).Outbound())
		}
	}
}

// A read error closes the write side. This is the reason Conn wraps Read at
// all, so it is pinned directly.
func TestConn_ReadErrorClosesTheWriteSide(t *testing.T) {
	id := astral.GenerateIdentity()
	writer := &closeRecorder{Writer: io.Discard}

	conn := NewConn(id, id, writer, failingReader{}, true)

	_, err := conn.Read(make([]byte, 4))
	if !errors.Is(err, errRead) {
		t.Fatalf("want %v, got %v", errRead, err)
	}
	if !writer.closed {
		t.Fatal("want the write side closed after a read error, got open")
	}
}

// EOF is a read error like any other, so the ordinary end of a stream also
// closes the write side.
func TestConn_EOFClosesTheWriteSide(t *testing.T) {
	id := astral.GenerateIdentity()
	writer := &closeRecorder{Writer: io.Discard}

	conn := NewConn(id, id, writer, new(eofReader), true)

	if _, err := conn.Read(make([]byte, 4)); !errors.Is(err, io.EOF) {
		t.Fatalf("want io.EOF, got %v", err)
	}
	if !writer.closed {
		t.Fatal("want the write side closed on EOF, got open")
	}
}

type eofReader struct{}

func (*eofReader) Read([]byte) (int, error) { return 0, io.EOF }

// A successful read leaves the connection open.
func TestConn_SuccessfulReadLeavesTheWriteSideOpen(t *testing.T) {
	id := astral.GenerateIdentity()
	writer := &closeRecorder{Writer: io.Discard}

	pipeReader, pipeWriter := io.Pipe()
	go func() {
		pipeWriter.Write([]byte("data"))
	}()

	conn := NewConn(id, id, writer, pipeReader, true)

	buf := make([]byte, 4)
	n, err := conn.Read(buf)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if n != 4 || string(buf) != "data" {
		t.Fatalf("want %q, got %q", "data", string(buf[:n]))
	}
	if writer.closed {
		t.Fatal("want the write side open after a successful read, got closed")
	}
}
