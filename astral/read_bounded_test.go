package astral

import (
	"bytes"
	"encoding/binary"
	"io"
	"runtime"
	"testing"
)

// A length prefix is the peer's claim about what follows, not a measurement of it.
// Decoding must never reserve memory for bytes that have not arrived: the budget
// below is generous for any honest payload and still orders of magnitude under what
// an unbounded reservation costs.
const allocBudget = 4 << 20 // 4 MiB

// allocatedBy reports the bytes allocated while fn runs.
func allocatedBy(fn func()) uint64 {
	var before, after runtime.MemStats

	runtime.GC()
	runtime.ReadMemStats(&before)

	fn()

	runtime.ReadMemStats(&after)
	return after.TotalAlloc - before.TotalAlloc
}

// claim builds a length-prefixed payload whose header names size bytes and whose
// body carries body — the shape a hostile peer sends.
func claim(size uint32, body []byte) []byte {
	var buf bytes.Buffer
	binary.Write(&buf, ByteOrder, size)
	buf.Write(body)
	return buf.Bytes()
}

func TestBytes32_ReadFromDoesNotReserveAnUnarrivedPayload(t *testing.T) {
	// 1 GiB claimed, four bytes delivered.
	payload := claim(1<<30, []byte("abcd"))

	var b Bytes32
	var err error

	used := allocatedBy(func() {
		_, err = (&b).ReadFrom(bytes.NewReader(payload))
	})

	if err == nil {
		t.Fatal("want an error for a payload shorter than its length prefix")
	}
	if used > allocBudget {
		t.Errorf("allocated %d bytes for a %d-byte payload (%.0fx)",
			used, len(payload), float64(used)/float64(len(payload)))
	}
}

// The destination keeps its previous value when the payload is short: a caller that
// ignores the error must not observe a truncated value that looks complete.
func TestBytes32_ShortReadLeavesTheDestination(t *testing.T) {
	b := Bytes32("original")

	_, err := (&b).ReadFrom(bytes.NewReader(claim(10, []byte("abcd"))))

	if err == nil {
		t.Fatal("want an error for a short payload")
	}
	if string(b) != "original" {
		t.Errorf("want the destination unchanged, got %q", b)
	}
}

func TestBytes32_RoundTripsAnHonestPayload(t *testing.T) {
	want := Bytes32("hello")

	var buf bytes.Buffer
	if _, err := want.WriteTo(&buf); err != nil {
		t.Fatalf("WriteTo: %v", err)
	}

	var got Bytes32
	if _, err := (&got).ReadFrom(&buf); err != nil {
		t.Fatalf("ReadFrom: %v", err)
	}
	if string(got) != string(want) {
		t.Errorf("want %q, got %q", want, got)
	}
}

type sliceHolder struct {
	Items []Uint8
}

func TestSliceValue_ReadFromDoesNotReserveUnarrivedElements(t *testing.T) {
	// 2^31 elements claimed, one delivered.
	payload := claim(1<<31, []byte{0x01})

	var holder sliceHolder
	var err error

	used := allocatedBy(func() {
		_, err = Objectify(&holder).ReadFrom(bytes.NewReader(payload))
	})

	if err == nil {
		t.Fatal("want an error for a count larger than the elements delivered")
	}
	if used > allocBudget {
		t.Errorf("allocated %d bytes for a %d-byte payload (%.0fx)",
			used, len(payload), float64(used)/float64(len(payload)))
	}
}

func TestSliceValue_RoundTripsAnHonestPayload(t *testing.T) {
	want := sliceHolder{Items: []Uint8{1, 2, 3}}

	var buf bytes.Buffer
	if _, err := Objectify(&want).WriteTo(&buf); err != nil {
		t.Fatalf("WriteTo: %v", err)
	}

	var got sliceHolder
	if _, err := Objectify(&got).ReadFrom(&buf); err != nil {
		t.Fatalf("ReadFrom: %v", err)
	}
	if len(got.Items) != 3 || got.Items[0] != 1 || got.Items[2] != 3 {
		t.Errorf("want [1 2 3], got %v", got.Items)
	}
}

// An empty slice stays non-nil, as it was before the growth change.
func TestSliceValue_EmptySliceIsNotNil(t *testing.T) {
	var got sliceHolder

	if _, err := Objectify(&got).ReadFrom(bytes.NewReader(claim(0, nil))); err != nil {
		t.Fatalf("ReadFrom: %v", err)
	}
	if got.Items == nil {
		t.Error("want a non-nil empty slice")
	}
}

func TestReadNBytes_ReportsShortReadsAsUnexpectedEOF(t *testing.T) {
	_, n, err := readNBytes(bytes.NewReader([]byte("ab")), 8)

	if err != io.ErrUnexpectedEOF {
		t.Errorf("want io.ErrUnexpectedEOF, got %v", err)
	}
	if n != 2 {
		t.Errorf("want 2 bytes consumed, got %d", n)
	}
}
