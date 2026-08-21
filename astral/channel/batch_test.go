package channel

import (
	"bytes"
	"errors"
	"io"
	"strings"
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

// encodeUnknownType appends a well-framed object whose type is not registered: the
// "unknown type tag on a binary channel" receive failure. Framing stays valid so the
// receiver fails on the decode, not on a truncated read.
func encodeUnknownType(t *testing.T, buf *bytes.Buffer, typeName string) {
	t.Helper()
	if _, err := astral.String8(typeName).WriteTo(buf); err != nil {
		t.Fatalf("encode type: %v", err)
	}
	if _, err := astral.Bytes32(nil).WriteTo(buf); err != nil {
		t.Fatalf("encode payload: %v", err)
	}
}

// TestBatch_ReceiveError_ReportsBeforeClosing: an input the channel cannot decode is
// answered with an error_message before the batch closes.
//
// Batch previously returned the Switch error with nothing sent, so the channel closed
// bare and the peer could not tell a rejected payload from a dropped transport. Replies
// for inputs already handled must survive ahead of the report.
func TestBatch_ReceiveError_ReportsBeforeClosing(t *testing.T) {
	a := astral.String8("a")
	in := encodeObjects(t, &a)
	encodeUnknownType(t, in, "unregistered.x.batch")
	var out bytes.Buffer

	err := Batch(New(Join(in, &out)), func(*astral.String8) astral.Object {
		return &astral.Ack{}
	})
	if err == nil {
		t.Fatal("want the receive error returned, got nil")
	}
	if !errors.Is(err, astral.ErrBlueprintNotFound) {
		t.Fatalf("want ErrBlueprintNotFound, got %v", err)
	}

	wantTypes(t, decodeTypes(t, &out), []string{"ack", "error_message"})
}

// TestBatch_ReceiveError_ReportCarriesTheReason: the reported error_message carries the
// failure text, so the peer learns why rather than only that something went wrong.
// Decoding it is itself the check that a Go peer survives reading the report.
func TestBatch_ReceiveError_ReportCarriesTheReason(t *testing.T) {
	in := encodeObjects(t)
	encodeUnknownType(t, in, "unregistered.x.reason")
	var out bytes.Buffer

	if err := Batch(New(Join(in, &out)), func(*astral.String8) astral.Object {
		return &astral.Ack{}
	}); err == nil {
		t.Fatal("want the receive error returned, got nil")
	}

	got, err := NewBinaryReceiver(bytes.NewReader(out.Bytes())).Receive()
	if err != nil {
		t.Fatalf("decode the report: %v", err)
	}
	msg, ok := got.(*astral.ErrorMessage)
	if !ok {
		t.Fatalf("want *astral.ErrorMessage, got %T", got)
	}
	if !strings.Contains(msg.Error(), "unregistered.x.reason") {
		t.Fatalf("report text: want it to name the offending type, got %q", msg.Error())
	}
}

// TestBatch_NoReceiveError_SendsNoReport: the report is not emitted on the healthy
// paths. An EOF-terminated stream still closes bare — the caller is gone — and an
// EOS-terminated one still ends with exactly one EOS.
func TestBatch_NoReceiveError_SendsNoReport(t *testing.T) {
	t.Run("eof", func(t *testing.T) {
		a := astral.String8("a")
		in := encodeObjects(t, &a)
		var out bytes.Buffer

		if err := Batch(New(Join(in, &out)), func(*astral.String8) astral.Object {
			return &astral.Ack{}
		}); err != nil {
			t.Fatalf("batch: %v", err)
		}
		wantTypes(t, decodeTypes(t, &out), []string{"ack"})
	})

	t.Run("eos", func(t *testing.T) {
		a := astral.String8("a")
		in := encodeObjects(t, &a, &astral.EOS{})
		var out bytes.Buffer

		if err := Batch(New(Join(in, &out)), func(*astral.String8) astral.Object {
			return &astral.Ack{}
		}); err != nil {
			t.Fatalf("batch: %v", err)
		}
		wantTypes(t, decodeTypes(t, &out), []string{"ack", "eos"})
	})
}

// TestBatch_ErrorObjectInput_AnsweredInBand: with T = astral.Object an error
// object on the input stream is an input of an unexpected type — a failed
// upstream stage reporting in-band — so fn never sees it, the reply is an
// error_message, and the batch continues.
func TestBatch_ErrorObjectInput_AnsweredInBand(t *testing.T) {
	a, b := astral.String8("a"), astral.String8("b")
	in := encodeObjects(t, &a, astral.NewError("upstream failed"), &b, &astral.EOS{})
	var out bytes.Buffer

	var seen []string
	err := Batch(New(Join(in, &out)), func(obj astral.Object) astral.Object {
		seen = append(seen, obj.ObjectType())
		return &astral.Ack{}
	})
	if err != nil {
		t.Fatalf("batch: %v", err)
	}

	wantTypes(t, seen, []string{"string8", "string8"})
	wantTypes(t, decodeTypes(t, &out), []string{"ack", "error_message", "ack", "eos"})
}

// TestBatch_ErrorObjectInput_ExactTypeWins: a batch whose T names an error
// object still receives it. The rejection above must not close the door on an
// op that legitimately takes error objects as its payload.
func TestBatch_ErrorObjectInput_ExactTypeWins(t *testing.T) {
	in := encodeObjects(t, astral.NewError("payload"), &astral.EOS{})
	var out bytes.Buffer

	var seen []string
	err := Batch(New(Join(in, &out)), func(msg *astral.ErrorMessage) astral.Object {
		seen = append(seen, msg.Error())
		return &astral.Ack{}
	})
	if err != nil {
		t.Fatalf("batch: %v", err)
	}

	wantTypes(t, seen, []string{"payload"})
	wantTypes(t, decodeTypes(t, &out), []string{"ack", "eos"})
}
