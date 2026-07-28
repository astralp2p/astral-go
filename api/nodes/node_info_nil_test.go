package nodes

import (
	"bytes"
	"testing"

	"github.com/astralp2p/astral-go/astral"
)

// A zero NodeInfo is what objects.new hands to the encoder, so it must encode
// rather than panic: reaching a nil Identity through streams.WriteAllTo calls a
// value-receiver method on a nil pointer, which takes the whole node down.
func TestNodeInfoWriteTo_NilIdentity(t *testing.T) {
	var buf bytes.Buffer

	n, err := NodeInfo{}.WriteTo(&buf)
	if err != nil {
		t.Fatalf("WriteTo: %v", err)
	}

	// string8 alias (1) + identity (33) + endpoint count (1).
	if want := int64(35); n != want {
		t.Errorf("wrote %d bytes, want %d", n, want)
	}
	if got, want := buf.Len(), 35; got != want {
		t.Errorf("buffer holds %d bytes, want %d", got, want)
	}
}

// The nil substitute must be the zero identity, which is the encoding ReadFrom
// already maps back to a zero value -- so the flat 33-byte form is unchanged and
// a zero NodeInfo round-trips.
func TestNodeInfoRoundTrip_NilIdentity(t *testing.T) {
	var buf bytes.Buffer

	if _, err := (NodeInfo{}).WriteTo(&buf); err != nil {
		t.Fatalf("WriteTo: %v", err)
	}

	var got NodeInfo
	if _, err := got.ReadFrom(&buf); err != nil {
		t.Fatalf("ReadFrom: %v", err)
	}

	if got.Identity == nil {
		t.Fatal("ReadFrom left Identity nil; it allocates unconditionally")
	}
	if !got.Identity.IsZero() {
		t.Errorf("Identity = %v, want the zero identity", got.Identity)
	}
	if len(got.Endpoints) != 0 {
		t.Errorf("Endpoints = %v, want none", got.Endpoints)
	}
}

// An identity that is present must still encode as its own 33 bytes, so the
// guard cannot be substituting the zero identity unconditionally.
func TestNodeInfoWriteTo_KeepsRealIdentity(t *testing.T) {
	id := astral.GenerateIdentity()

	var buf bytes.Buffer
	if _, err := (NodeInfo{Identity: id}).WriteTo(&buf); err != nil {
		t.Fatalf("WriteTo: %v", err)
	}

	var got NodeInfo
	if _, err := got.ReadFrom(&buf); err != nil {
		t.Fatalf("ReadFrom: %v", err)
	}
	if !got.Identity.IsEqual(id) {
		t.Errorf("Identity = %v, want %v", got.Identity, id)
	}
}

// ReadFrom must not swallow a failure from the alias/identity read.
//
// A truncated frame does not prove this: it drains the reader, so the following
// endpoint-count read fails too and an error surfaces either way. The failure is
// only visible when the identity read fails while bytes remain -- an invalid key
// is read in full and then rejected by ParsePubKey -- because the next read then
// succeeds and overwrites the error.
func TestNodeInfoReadFrom_InvalidIdentityErrors(t *testing.T) {
	frame := []byte{0x00}                                    // empty alias
	frame = append(frame, bytes.Repeat([]byte{0xff}, 33)...) // 33 bytes, not a valid compressed key
	frame = append(frame, 0x00)                              // endpoint count, reads cleanly

	var got NodeInfo
	if _, err := got.ReadFrom(bytes.NewReader(frame)); err == nil {
		t.Fatal("ReadFrom accepted an unparseable identity")
	}
}
