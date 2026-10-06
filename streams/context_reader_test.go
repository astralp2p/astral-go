package streams

import (
	"context"
	"errors"
	"testing"
	"time"
)

// instantReader returns one byte on every Read without blocking.
type instantReader struct{}

func (instantReader) Read(p []byte) (int, error) {
	p[0] = 'x'
	return 1, nil
}

// gatedReader blocks every Read until release is closed.
type gatedReader struct {
	release chan struct{}
}

func (g gatedReader) Read(p []byte) (int, error) {
	<-g.release
	return copy(p, "data"), nil
}

// TestContextReaderFastReader drives ReadContext over a reader that returns at once, so the background
// read finishes while ReadContext is still waiting on it. Run with -race to detect unsynchronized access
// to the reader's channel; without the race detector a lost wakeup shows up as a timeout.
func TestContextReaderFastReader(t *testing.T) {
	r := NewContextReader(instantReader{})

	done := make(chan error, 1)
	go func() {
		buf := make([]byte, 1)
		for i := 0; i < 1000; i++ {
			if _, err := r.ReadContext(context.Background(), buf); err != nil {
				done <- err
				return
			}
		}
		done <- nil
	}()

	select {
	case err := <-done:
		if err != nil {
			t.Fatal("unexpected error:", err)
		}
	case <-time.After(10 * time.Second):
		t.Fatal("ReadContext did not return")
	}
}

// TestContextReaderCanceledThenBuffered checks that a canceled ReadContext returns the context error and
// that the data from the abandoned read is returned by the next call.
func TestContextReaderCanceledThenBuffered(t *testing.T) {
	g := gatedReader{release: make(chan struct{})}
	r := NewContextReader(g)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	buf := make([]byte, 16)
	n, err := r.ReadContext(ctx, buf)
	if n != 0 || !errors.Is(err, context.Canceled) {
		t.Fatalf("expected 0, context.Canceled; got %d, %v", n, err)
	}

	close(g.release)

	n, err = r.ReadContext(context.Background(), buf)
	if err != nil {
		t.Fatal("unexpected error:", err)
	}
	if string(buf[:n]) != "data" {
		t.Fatalf("expected %q, got %q", "data", buf[:n])
	}
}
