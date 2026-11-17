package streams

import (
	"errors"
	"io"
	"net"
	"sync"
	"testing"
)

// TestJoinBothSidesCloseAtOnce closes both peers concurrently so both copy
// goroutines finish together; run with -race.
func TestJoinBothSidesCloseAtOnce(t *testing.T) {
	for i := 0; i < 50; i++ {
		l1, l2 := net.Pipe()
		r1, r2 := net.Pipe()

		done := make(chan error, 1)
		go func() {
			_, _, err := Join(l2, r2)
			done <- err
		}()

		var wg sync.WaitGroup
		wg.Add(2)
		go func() { defer wg.Done(); l1.Close() }()
		go func() { defer wg.Done(); r1.Close() }()
		wg.Wait()

		// Closing a peer mid-copy may surface as a closed-pipe write error.
		if err := <-done; err != nil && !errors.Is(err, io.ErrClosedPipe) {
			t.Fatalf("iteration %d: Join returned %v", i, err)
		}
	}
}

type errReader struct{ err error }

func (r errReader) Read([]byte) (int, error)    { return 0, r.err }
func (r errReader) Write(p []byte) (int, error) { return len(p), nil }
func (r errReader) Close() error                { return nil }

func TestJoinReturnsNonEOFError(t *testing.T) {
	want := errors.New("boom")
	l1, l2 := net.Pipe()
	defer l1.Close()

	done := make(chan error, 1)
	go func() {
		_, _, err := Join(l2, errReader{err: want})
		done <- err
	}()
	go io.Copy(io.Discard, l1)

	if err := <-done; !errors.Is(err, want) {
		t.Fatalf("Join returned %v, want %v", err, want)
	}
}
