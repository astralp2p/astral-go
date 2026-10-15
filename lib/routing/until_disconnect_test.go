package routing

import (
	"io"
	"testing"
	"time"

	"github.com/astralp2p/astral-go/astral"
)

// requesterConn builds the connection AcceptRaw hands an op: the requester
// writes into a pipe the op reads, and the op's answers leave through another
// pipe that answers drains.
func requesterConn(t *testing.T) (conn *Conn, requester *io.PipeWriter) {
	t.Helper()

	pr, pw := io.Pipe()
	ar, aw := io.Pipe()
	go func() { _, _ = io.Copy(io.Discard, ar) }()

	id := astral.GenerateIdentity()
	return NewConn(id, id, aw, pr, false).(*Conn), pw
}

func awaitDone(t *testing.T, ctx *astral.Context, what string) {
	t.Helper()

	select {
	case <-ctx.Done():
	case <-time.After(2 * time.Second):
		t.Fatalf("the context did not end when %s", what)
	}
}

// The context ends when the requester closes its side, and the answer
// direction stays open for the answer in flight.
func TestUntilDisconnect_EndsWhenTheRequesterCloses(t *testing.T) {
	conn, requester := requesterConn(t)

	ctx, stop := UntilDisconnect(astral.NewContext(nil), conn)
	defer stop()

	select {
	case <-ctx.Done():
		t.Fatal("the context ended before the requester left")
	case <-time.After(50 * time.Millisecond):
	}

	_ = requester.Close()
	awaitDone(t, ctx, "the requester closed")

	if _, err := conn.Write([]byte("answer")); err != nil {
		t.Fatalf("the answer direction closed with the requester: %v", err)
	}
}

// stop ends the watch before the requester closes: the context ends and the
// read side closes, so a late write by the requester fails instead of parking.
func TestUntilDisconnect_StopEndsTheWatch(t *testing.T) {
	conn, requester := requesterConn(t)

	ctx, stop := UntilDisconnect(astral.NewContext(nil), conn)
	stop()
	awaitDone(t, ctx, "stop was called")

	if _, err := requester.Write([]byte("late")); err == nil {
		t.Fatal("a write after stop parked on an open read side")
	}
}

// A connection that is not a Conn is watched through its own reader.
func TestUntilDisconnect_WatchesAPlainConnection(t *testing.T) {
	pr, pw := io.Pipe()
	conn := struct {
		io.Reader
		nopWriteCloser
	}{Reader: pr}

	ctx, stop := UntilDisconnect(astral.NewContext(nil), conn)
	defer stop()

	_ = pw.Close()
	awaitDone(t, ctx, "the requester closed")
}
