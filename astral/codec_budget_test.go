package astral

import (
	"bytes"
	"errors"
	"io"
	"testing"
)

// The codec ledger charges three budgets. Depth and frames are charged in enter, which
// returns through the frame's own error path, so they were always enforced. Bytes were
// charged in Read, which returns the count and the error together — and that is the one
// place an error does not survive, because io.ReadFull ends with
//
//	if n >= min { err = nil }
//
// so a read that was satisfied discards it. Every length-prefixed payload in the codec
// reads through io.ReadFull, which made MaxCodecBytes inert on exactly the path it was
// written to bound.

// countingReader yields zeros forever and records how many bytes it was asked for, so a
// test can assert the ledger stopped pulling rather than merely reported something.
type countingReader struct{ served int64 }

func (c *countingReader) Read(p []byte) (int, error) {
	for i := range p {
		p[i] = 0
	}
	c.served += int64(len(p))
	return len(p), nil
}

// The regression test for the swallowed error. An objectReader whose budget is nearly
// spent must not let io.ReadFull complete: this is the exact call shape that hid the
// defect, so asserting on Read alone would not have caught it.
func TestBudget_ReadFullCannotOutrunTheByteBudget(t *testing.T) {
	src := &countingReader{}
	or := attachReader(src, nil)
	or.b.bytes = MaxCodecBytes - 16 // 16 bytes of room left

	buf := make([]byte, 1024)
	n, err := io.ReadFull(or, buf)

	if err == nil {
		t.Fatalf("io.ReadFull completed %d bytes past a spent budget", n)
	}
	if !errors.Is(err, ErrDepthExceeded) {
		t.Errorf("want ErrDepthExceeded, got %v", err)
	}
	if n > 16 {
		t.Errorf("want at most the 16 bytes of remaining budget, got %d", n)
	}
	if src.served > 16 {
		t.Errorf("the reader was asked for %d bytes with 16 of budget left; the "+
			"clamp is what bounds it, not the accounting afterwards", src.served)
	}
}

// A budget with room left must not be disturbed: the clamp only engages at the ceiling.
func TestBudget_ReadIsUnaffectedWithRoomLeft(t *testing.T) {
	or := attachReader(&countingReader{}, nil)

	buf := make([]byte, 1024)
	n, err := io.ReadFull(or, buf)

	if err != nil {
		t.Fatalf("want a clean read well under the budget, got %v", err)
	}
	if n != len(buf) {
		t.Errorf("want %d bytes, got %d", len(buf), n)
	}
	if or.b.bytes != int64(len(buf)) {
		t.Errorf("want %d charged, got %d", len(buf), or.b.bytes)
	}
}

// ErrUnexpectedObject is a registered struct with a polymorphic field that goes through
// Objectify, so a chain of them drives the reflection codec's interface, ptr and struct
// frames — the six frames that were guarded on the read side and on neither side of the
// write. Using a type the default registry already holds keeps this test off the
// per-call registry, which a polymorphic field does not honour today (interfaceValue
// resolves through the package-level New).
func nestBudget(n int) Object {
	s := String8("x")
	var o Object = &s
	for range n {
		o = &ErrUnexpectedObject{Object: o}
	}
	return o
}

// The defect that matters on the wire: astral-go encoded to any depth and refused to
// decode past its cap, so it put objects on the wire that no conforming peer — itself
// included — would accept. The property is one-directional and exact: whatever encodes
// must decode.
func TestBudget_WhateverEncodesCanBeDecoded(t *testing.T) {
	for depth := 1; depth <= 4*MaxBlueprintDepth; depth++ {
		var buf bytes.Buffer
		_, encErr := Encode(&buf, nestBudget(depth))
		_, _, decErr := Decode(bytes.NewReader(buf.Bytes()))

		switch {
		case encErr == nil && decErr != nil:
			t.Fatalf("depth %d: encoded %d bytes that cannot be decoded (%v) — the "+
				"encoder is looser than the decoder", depth, buf.Len(), decErr)
		case encErr != nil && !errors.Is(encErr, ErrDepthExceeded):
			t.Fatalf("depth %d: want ErrDepthExceeded from the encoder, got %v", depth, encErr)
		}
	}
}

// The encoder must actually stop somewhere, or the test above passes by never refusing.
func TestBudget_EncodeRefusesPastTheCap(t *testing.T) {
	firstRefusal := -1

	for depth := 1; depth <= 4*MaxBlueprintDepth; depth++ {
		var buf bytes.Buffer
		if _, err := Encode(&buf, nestBudget(depth)); err != nil {
			if !errors.Is(err, ErrDepthExceeded) {
				t.Fatalf("depth %d: want ErrDepthExceeded, got %v", depth, err)
			}
			firstRefusal = depth
			break
		}
	}

	if firstRefusal < 0 {
		t.Fatalf("the encoder accepted every depth up to %d; it is unbounded",
			4*MaxBlueprintDepth)
	}
	if firstRefusal > MaxBlueprintDepth {
		t.Errorf("first refusal at depth %d, past the cap of %d",
			firstRefusal, MaxBlueprintDepth)
	}
}
