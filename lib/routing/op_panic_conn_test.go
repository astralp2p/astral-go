package routing

import (
	"errors"
	"io"
	"testing"
	"time"

	"github.com/astralp2p/astral-go/astral"
)

// An op that accepts and then panics leaves the caller holding a pipe with nothing on
// the other end. TestOp_PanicAfterAcceptKeepsTheConnection asserts the caller gets its
// writer, which is right — the recovery must not downgrade a live query. What it does
// not cover is what that writer is still connected to once the op's goroutine has
// unwound, and that is where the defect lived: the test never writes to it.
//
// AcceptRaw builds an io.Pipe, hands the caller the write end and keeps the read end for
// the op. A pipe write parks until a reader takes the bytes, so with the op gone the
// caller's first write never returns.

// probeWait bounds a call that must not block. It only has to outlast a healthy
// round trip, which is microseconds here.
const probeWait = 2 * time.Second

// AcceptAndServe accepts and hands the connection to another goroutine, which is the
// legitimate pattern the teardown must not break: the op returns while the connection
// stays live and serviced.
func (o panicOps) AcceptAndServe(_ *astral.Context, q *IncomingQuery) error {
	conn := q.AcceptRaw()
	go io.Copy(io.Discard, conn)
	return nil
}

// The defect. Before the fix this write blocked forever; now it fails, and fails with
// the reason.
func TestOp_PanicAfterAcceptUnblocksTheCallersWrite(t *testing.T) {
	router, reports := routerWithReports(t)

	w, err := routeTo(t, router, "accept_then_boom")
	if err != nil || w == nil {
		t.Fatalf("want an accepted query, got w=%v err=%v", w, err)
	}
	<-reports // the op has panicked and its goroutine has unwound

	type result struct {
		err error
	}
	done := make(chan result, 1)
	go func() {
		_, werr := w.Write([]byte("anyone there?"))
		done <- result{werr}
	}()

	select {
	case got := <-done:
		if got.err == nil {
			t.Fatal("want the write to fail once the op is gone, got nil")
		}
		// the cause is carried through the pipe, so the caller learns why rather than
		// getting a bare "closed pipe"
		var p *ErrPanic
		if !errors.As(got.err, &p) {
			t.Errorf("want the write error to carry *ErrPanic, got %v", got.err)
		}
	case <-time.After(probeWait):
		t.Fatalf("the caller's write blocked for %v with no reader on the other end", probeWait)
	}
}

// The scope boundary of this fix, and the reason it tears down on panic only. An op is
// allowed to accept and return while another goroutine services the connection; closing
// on every return would break that.
func TestOp_AcceptAndReturnKeepsTheConnectionUsable(t *testing.T) {
	router, reports := routerWithReports(t)

	w, err := routeTo(t, router, "accept_and_serve")
	if err != nil || w == nil {
		t.Fatalf("want an accepted query, got w=%v err=%v", w, err)
	}
	<-reports // the op has returned

	done := make(chan error, 1)
	go func() {
		_, werr := w.Write([]byte("still there?"))
		done <- werr
	}()

	select {
	case werr := <-done:
		if werr != nil {
			t.Errorf("want the connection to stay usable after a clean return, got %v", werr)
		}
	case <-time.After(probeWait):
		t.Fatalf("write blocked for %v: the teardown fired on a query that did not panic", probeWait)
	}
}

// A query that panicked before resolving has no connection to tear down, so the teardown
// must be a noop rather than an error or a second resolve.
func TestOp_PanicBeforeAcceptTearsDownNothing(t *testing.T) {
	router, _ := routerWithReports(t)

	w, err := routeTo(t, router, "boom")

	if w != nil {
		t.Errorf("want no writer, got %v", w)
	}
	var rejected *astral.ErrRejected
	if !errors.As(err, &rejected) {
		t.Fatalf("want *astral.ErrRejected, got %v", err)
	}
	if rejected.Code != astral.CodeInternalError {
		t.Errorf("want code %d, got %d", astral.CodeInternalError, rejected.Code)
	}
}
