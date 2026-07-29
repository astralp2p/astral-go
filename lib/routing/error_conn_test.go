package routing

import (
	"errors"
	"testing"
)

// ErrorConn stands in for a connection that could not be established. Every
// read and write reports the stored error, so a caller that ignores the
// handshake still fails on first use rather than silently succeeding.

var errConn = errors.New("connection unavailable")

func TestErrorConn_ReadAndWriteReportTheError(t *testing.T) {
	conn := ErrorConn{Err: errConn}

	n, err := conn.Read(make([]byte, 8))
	if !errors.Is(err, errConn) {
		t.Fatalf("Read: want %v, got %v", errConn, err)
	}
	if n != 0 {
		t.Fatalf("Read: want 0 bytes, got %d", n)
	}

	n, err = conn.Write([]byte("payload"))
	if !errors.Is(err, errConn) {
		t.Fatalf("Write: want %v, got %v", errConn, err)
	}
	if n != 0 {
		t.Fatalf("Write: want 0 bytes, got %d", n)
	}
}

// Close succeeds: tearing down a connection that never opened is not itself a
// failure, so a deferred Close cannot mask the real error.
func TestErrorConn_CloseSucceeds(t *testing.T) {
	if err := (ErrorConn{Err: errConn}).Close(); err != nil {
		t.Fatalf("want nil, got %v", err)
	}
}

// A zero ErrorConn reports a nil error rather than panicking.
func TestErrorConn_ZeroValueIsUsable(t *testing.T) {
	var conn ErrorConn

	if _, err := conn.Read(make([]byte, 1)); err != nil {
		t.Fatalf("Read: want nil, got %v", err)
	}
	if _, err := conn.Write([]byte("x")); err != nil {
		t.Fatalf("Write: want nil, got %v", err)
	}
}
