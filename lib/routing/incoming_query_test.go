package routing

import (
	"context"
	"errors"
	"io"
	"testing"
	"time"

	"github.com/astralp2p/astral-go/astral"
	"github.com/astralp2p/astral-go/lib/query"
)

// IncomingQuery is the server-side handle on a query. Its central rule is that
// it resolves exactly once: the first Accept or Reject wins and every later
// call is refused, so a handler cannot answer twice.

func newTestIncomingQuery(t *testing.T) *IncomingQuery {
	t.Helper()

	id := astral.GenerateIdentity()
	return NewIncomingQuery(query.New(id, id, "op?a=1", nil), nopWriteCloser{}, "")
}

func TestIncomingQuery_ExposesTheQuery(t *testing.T) {
	id := astral.GenerateIdentity()
	q := query.New(id, id, "op?a=1", nil)

	in := NewIncomingQuery(q, nopWriteCloser{}, astral.OriginNetwork)

	if in.QueryString() != "op?a=1" {
		t.Fatalf("QueryString: want %q, got %q", "op?a=1", in.QueryString())
	}
	if !in.Caller().IsEqual(q.Caller) {
		t.Fatal("Caller: want the query's caller")
	}
	if !in.Target().IsEqual(q.Target) {
		t.Fatal("Target: want the query's target")
	}
	if in.Nonce() != q.Nonce {
		t.Fatalf("Nonce: want %v, got %v", q.Nonce, in.Nonce())
	}
	if in.Origin() != astral.OriginNetwork {
		t.Fatalf("Origin: want %q, got %q", astral.OriginNetwork, in.Origin())
	}
}

// A locally originated query carries the empty origin.
func TestIncomingQuery_LocalOriginIsEmpty(t *testing.T) {
	if got := newTestIncomingQuery(t).Origin(); got != "" {
		t.Fatalf("want empty, got %q", got)
	}
}

// AcceptRaw resolves the query and yields an inbound connection with the
// identities in responder order.
func TestIncomingQuery_AcceptRaw(t *testing.T) {
	in := newTestIncomingQuery(t)

	conn := in.AcceptRaw()
	if conn == nil {
		t.Fatal("want a connection, got nil")
	}

	astralConn, ok := conn.(astral.Conn)
	if !ok {
		t.Fatalf("want an astral.Conn, got %T", conn)
	}
	if !astralConn.LocalIdentity().IsEqual(in.Query.Caller) {
		t.Fatal("LocalIdentity: want the caller")
	}
}

// The second resolution is refused. AcceptRaw cannot return an error, so it
// returns a connection that fails on use instead.
func TestIncomingQuery_SecondAcceptYieldsAFailingConn(t *testing.T) {
	in := newTestIncomingQuery(t)

	in.AcceptRaw()
	second := in.AcceptRaw()

	errorConn, ok := second.(ErrorConn)
	if !ok {
		t.Fatalf("want an ErrorConn, got %T", second)
	}
	if errorConn.Err == nil {
		t.Fatal("want the conn to carry an error, got nil")
	}

	if _, err := second.Read(make([]byte, 1)); err == nil {
		t.Fatal("want a read error on the second connection, got nil")
	}
}

// Reject after Accept is refused, so a deferred Reject cannot undo an accept.
// This is what makes Op's "reject at the end in case the op did not respond"
// safe.
func TestIncomingQuery_RejectAfterAcceptIsRefused(t *testing.T) {
	in := newTestIncomingQuery(t)

	in.AcceptRaw()

	if err := in.Reject(); err == nil {
		t.Fatal("want an error rejecting a resolved query, got nil")
	}
	if err := in.RejectWithCode(7); err == nil {
		t.Fatal("want an error rejecting a resolved query, got nil")
	}
}

func TestIncomingQuery_AcceptAfterRejectIsRefused(t *testing.T) {
	in := newTestIncomingQuery(t)

	if err := in.Reject(); err != nil {
		t.Fatalf("Reject: unexpected error: %v", err)
	}

	if _, ok := in.AcceptRaw().(ErrorConn); !ok {
		t.Fatal("want an ErrorConn after a rejection")
	}
}

func TestIncomingQuery_DoubleRejectIsRefused(t *testing.T) {
	in := newTestIncomingQuery(t)

	if err := in.Reject(); err != nil {
		t.Fatalf("first Reject: unexpected error: %v", err)
	}
	if err := in.Reject(); err == nil {
		t.Fatal("want an error on the second Reject, got nil")
	}
}

// await hands the caller the writer the responder produced.
func TestIncomingQuery_AwaitReturnsTheAcceptedWriter(t *testing.T) {
	in := newTestIncomingQuery(t)

	go in.AcceptRaw()

	w, err := in.await(astral.NewContext(nil))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if w == nil {
		t.Fatal("want a writer, got nil")
	}
}

func TestIncomingQuery_AwaitReturnsTheRejection(t *testing.T) {
	in := newTestIncomingQuery(t)

	go in.RejectWithCode(7)

	w, err := in.await(astral.NewContext(nil))

	var rejected *astral.ErrRejected
	if !errors.As(err, &rejected) {
		t.Fatalf("want ErrRejected, got %v", err)
	}
	if rejected.Code != 7 {
		t.Fatalf("Code: want 7, got %d", rejected.Code)
	}
	if w != nil {
		t.Fatal("want no writer on rejection, got one")
	}
}

// Code 0 is not a rejection code on the wire, so RejectWithCode substitutes 1.
func TestIncomingQuery_RejectWithZeroCodeSubstitutesOne(t *testing.T) {
	in := newTestIncomingQuery(t)

	go in.RejectWithCode(0)

	_, err := in.await(astral.NewContext(nil))

	var rejected *astral.ErrRejected
	if !errors.As(err, &rejected) {
		t.Fatalf("want ErrRejected, got %v", err)
	}
	if rejected.Code != 1 {
		t.Fatalf("Code: want 1, got %d", rejected.Code)
	}
}

// A cancelled context ends the wait with the cancellation code rather than the
// generic rejection.
func TestIncomingQuery_AwaitHonoursContextCancellation(t *testing.T) {
	in := newTestIncomingQuery(t)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	w, err := in.await(astral.NewContext(ctx))

	var rejected *astral.ErrRejected
	if !errors.As(err, &rejected) {
		t.Fatalf("want ErrRejected, got %v", err)
	}
	if rejected.Code != astral.CodeCanceled {
		t.Fatalf("Code: want %d, got %d", astral.CodeCanceled, rejected.Code)
	}
	if w != nil {
		t.Fatal("want no writer, got one")
	}
}

// A handler that never answers is cut off by the 5-second deadline. The
// duration is a literal in await, so this test costs that wall clock and is
// skipped in short mode.
func TestIncomingQuery_AwaitTimesOut(t *testing.T) {
	if testing.Short() {
		t.Skip("the await deadline is a hard-coded 5s")
	}

	in := newTestIncomingQuery(t)

	start := time.Now()
	w, err := in.await(astral.NewContext(nil))
	elapsed := time.Since(start)

	if !errors.Is(err, &astral.ErrRejected{}) {
		t.Fatalf("want ErrRejected, got %v", err)
	}
	if w != nil {
		t.Fatal("want no writer, got one")
	}
	if elapsed < 4*time.Second {
		t.Fatalf("want the wait to reach the deadline, got %v", elapsed)
	}
}

// Accept wraps the raw connection in a channel for object-level ops.
func TestIncomingQuery_AcceptReturnsAChannel(t *testing.T) {
	in := newTestIncomingQuery(t)

	ch := in.Accept()
	if ch == nil {
		t.Fatal("want a channel, got nil")
	}
	ch.Close()
}

var _ io.ReadWriteCloser = ErrorConn{}
