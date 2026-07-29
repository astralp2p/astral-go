package channel

import (
	"bytes"
	"errors"
	"io"
	"testing"

	"github.com/astralp2p/astral-go/astral"
)

func encodeObjects(t *testing.T, objects ...astral.Object) *bytes.Buffer {
	t.Helper()
	var buf bytes.Buffer
	s := NewBinarySender(&buf)
	for _, o := range objects {
		if err := s.Send(o); err != nil {
			t.Fatalf("encode %s: %v", o.ObjectType(), err)
		}
	}
	return &buf
}

func decodeTypes(t *testing.T, buf *bytes.Buffer) []string {
	t.Helper()
	r := NewBinaryReceiver(bytes.NewReader(buf.Bytes()))
	var types []string
	for {
		o, err := r.Receive()
		if errors.Is(err, io.EOF) {
			return types
		}
		if err != nil {
			t.Fatalf("decode: %v", err)
		}
		types = append(types, o.ObjectType())
	}
}

func wantTypes(t *testing.T, got, want []string) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("got %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("reply %d: got %v, want %v", i, got, want)
		}
	}
}

// TestBatch_ExplicitEOS_MirrorsTerminator: an input stream terminated by an
// explicit EOS gets one reply per input followed by a terminating EOS.
func TestBatch_ExplicitEOS_MirrorsTerminator(t *testing.T) {
	a, b := astral.String8("a"), astral.String8("b")
	in := encodeObjects(t, &a, &b, &astral.EOS{})
	var out bytes.Buffer

	err := Batch(New(Join(in, &out)), func(*astral.String8) astral.Object {
		return &astral.Ack{}
	})
	if err != nil {
		t.Fatalf("batch: %v", err)
	}

	wantTypes(t, decodeTypes(t, &out), []string{"ack", "ack", "eos"})
}

// TestBatch_EOF_NoTrailingEOS: an input stream ended by EOF gets one reply per
// input and no terminating EOS.
func TestBatch_EOF_NoTrailingEOS(t *testing.T) {
	a := astral.String8("a")
	in := encodeObjects(t, &a)
	var out bytes.Buffer

	err := Batch(New(Join(in, &out)), func(*astral.String8) astral.Object {
		return &astral.Ack{}
	})
	if err != nil {
		t.Fatalf("batch: %v", err)
	}

	wantTypes(t, decodeTypes(t, &out), []string{"ack"})
}

// TestBatch_WrongType_AnsweredInBand: a wrong-typed input is answered with an
// error_message and the batch continues.
func TestBatch_WrongType_AnsweredInBand(t *testing.T) {
	a, b := astral.String8("a"), astral.String8("b")
	in := encodeObjects(t, &a, &astral.Ack{}, &b, &astral.EOS{})
	var out bytes.Buffer

	err := Batch(New(Join(in, &out)), func(*astral.String8) astral.Object {
		return &astral.Ack{}
	})
	if err != nil {
		t.Fatalf("batch: %v", err)
	}

	wantTypes(t, decodeTypes(t, &out), []string{"ack", "error_message", "ack", "eos"})
}

// TestBatch_DegenerateObject_EOSStillTerminates: with T = astral.Object the
// typed arm is the catch-all, yet an explicit EOS still terminates the batch
// instead of reaching fn, and is mirrored.
func TestBatch_DegenerateObject_EOSStillTerminates(t *testing.T) {
	a := astral.String8("a")
	in := encodeObjects(t, &a, &astral.Ack{}, &astral.EOS{})
	var out bytes.Buffer

	var seen int
	err := Batch(New(Join(in, &out)), func(astral.Object) astral.Object {
		seen++
		return &astral.Ack{}
	})
	if err != nil {
		t.Fatalf("batch: %v", err)
	}
	if seen != 2 {
		t.Fatalf("fn saw %d objects, want 2", seen)
	}

	wantTypes(t, decodeTypes(t, &out), []string{"ack", "ack", "eos"})
}

// TestBatch_ErrorReply_ContinuesBatch: fn answering an error object reports a
// failed input without ending the batch.
func TestBatch_ErrorReply_ContinuesBatch(t *testing.T) {
	a, b := astral.String8("fail"), astral.String8("ok")
	in := encodeObjects(t, &a, &b, &astral.EOS{})
	var out bytes.Buffer

	err := Batch(New(Join(in, &out)), func(s *astral.String8) astral.Object {
		if string(*s) == "fail" {
			return astral.NewError("no")
		}
		return &astral.Ack{}
	})
	if err != nil {
		t.Fatalf("batch: %v", err)
	}

	wantTypes(t, decodeTypes(t, &out), []string{"error_message", "ack", "eos"})
}
