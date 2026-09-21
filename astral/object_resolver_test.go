package astral

import (
	"bytes"
	"errors"
	"testing"
)

// The published vector for the SHA-256 ObjectID of "hello".
// Source: .ai/system primitive-types/object_id.sha256.md
const helloObjectID = "data1km81js7f9cfdbauqoq3kash6f8o5naxfa878ejx8gbbuckjazgbr"

// A WriteResolver names the same ObjectID whether or not it wraps a writer: the
// pass-through form counts the bytes it forwards toward Size.
func TestWriteResolver_Write_PassThroughKeepsSize(t *testing.T) {
	var data = []byte("hello")

	direct := NewWriteResolver(nil)
	if _, err := direct.Write(data); err != nil {
		t.Fatalf("write: %v", err)
	}

	var sink bytes.Buffer
	wrapped := NewWriteResolver(&sink)
	if _, err := wrapped.Write(data); err != nil {
		t.Fatalf("write: %v", err)
	}

	if got := wrapped.Resolve(); got.String() != direct.Resolve().String() {
		t.Fatalf("want %v, got %v", direct.Resolve(), got)
	}
	if got := wrapped.Resolve(); got.String() != helloObjectID {
		t.Fatalf("want %v, got %v", helloObjectID, got)
	}
	if got := wrapped.Resolve().Size; got != uint64(len(data)) {
		t.Fatalf("want Size %d, got %d", len(data), got)
	}
	if got := sink.Bytes(); !bytes.Equal(got, data) {
		t.Fatalf("want %v in the sink, got %v", data, got)
	}
}

// A short write counts only the bytes the underlying writer accepted.
func TestWriteResolver_Write_ShortWriteCountsAccepted(t *testing.T) {
	var sink = shortWriter{limit: 3}

	r := NewWriteResolver(&sink)
	n, err := r.Write([]byte("hello"))
	if err == nil {
		t.Fatal("want the short write's error, got nil")
	}
	if n != 3 {
		t.Fatalf("want n 3, got %d", n)
	}

	expected := NewWriteResolver(nil)
	expected.Write([]byte("hel"))

	if got := r.Resolve(); got.String() != expected.Resolve().String() {
		t.Fatalf("want %v, got %v", expected.Resolve(), got)
	}
}

var errShortWriter = errors.New("short write")

// shortWriter accepts at most limit bytes and then reports an error.
type shortWriter struct {
	limit int
}

func (w *shortWriter) Write(p []byte) (int, error) {
	if len(p) <= w.limit {
		w.limit -= len(p)
		return len(p), nil
	}

	n := w.limit
	w.limit = 0

	return n, errShortWriter
}
