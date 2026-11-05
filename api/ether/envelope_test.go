package ether

import (
	"bytes"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/astralp2p/astral-go/astral"
)

func sampleBundle(t *testing.T) *astral.Bundle {
	t.Helper()
	b := astral.NewBundle()
	if err := b.Append(astral.NewString8("hi"), astral.NewString16("there")); err != nil {
		t.Fatalf("Append: %v", err)
	}
	return b
}

func encodeEnvelope(t *testing.T, e *Envelope) []byte {
	t.Helper()
	var buf bytes.Buffer
	n, err := e.WriteTo(&buf)
	if err != nil {
		t.Fatalf("WriteTo: %v", err)
	}
	if n != int64(buf.Len()) {
		t.Fatalf("WriteTo reported %d bytes; wrote %d", n, buf.Len())
	}
	return buf.Bytes()
}

func TestEnvelopeBinaryRoundTrip(t *testing.T) {
	tests := []struct {
		name string
		env  Envelope
	}{
		{"string8", Envelope{Version: EnvelopeVersion, Object: astral.NewString8("hello")}},
		{"bundle", Envelope{Version: EnvelopeVersion, Object: sampleBundle(t)}},
		{"absent object", Envelope{Version: EnvelopeVersion}},
		{"other version", Envelope{Version: 7, Object: astral.NewString8("x")}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			wire := encodeEnvelope(t, &tt.env)
			if wire[0] != byte(tt.env.Version) {
				t.Errorf("first byte = %#x; want the version %#x", wire[0], tt.env.Version)
			}

			var got Envelope
			n, err := got.ReadFrom(bytes.NewReader(wire))
			if err != nil {
				t.Fatalf("ReadFrom: %v", err)
			}
			if n != int64(len(wire)) {
				t.Errorf("ReadFrom reported %d bytes; wire holds %d", n, len(wire))
			}
			if got.Version != tt.env.Version {
				t.Errorf("Version = %d; want %d", got.Version, tt.env.Version)
			}
			if (got.Object == nil) != (tt.env.Object == nil) {
				t.Fatalf("Object = %#v; want %#v", got.Object, tt.env.Object)
			}
			if again := encodeEnvelope(t, &got); !bytes.Equal(again, wire) {
				t.Errorf("re-encoded bytes differ:\n got %x\nwant %x", again, wire)
			}
		})
	}
}

func TestEnvelopeJSONRoundTrip(t *testing.T) {
	tests := []struct {
		name string
		env  Envelope
	}{
		{"string8", Envelope{Version: EnvelopeVersion, Object: astral.NewString8("hello")}},
		{"bundle", Envelope{Version: EnvelopeVersion, Object: sampleBundle(t)}},
		{"absent object", Envelope{Version: EnvelopeVersion}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, err := json.Marshal(tt.env)
			if err != nil {
				t.Fatalf("Marshal: %v", err)
			}

			var got Envelope
			if err := json.Unmarshal(data, &got); err != nil {
				t.Fatalf("Unmarshal %s: %v", data, err)
			}
			if got.Version != tt.env.Version {
				t.Errorf("Version = %d; want %d", got.Version, tt.env.Version)
			}

			again, err := json.Marshal(got)
			if err != nil {
				t.Fatalf("Marshal again: %v", err)
			}
			if !bytes.Equal(again, data) {
				t.Errorf("re-encoded JSON differs:\n got %s\nwant %s", again, data)
			}
			if !bytes.Equal(encodeEnvelope(t, &got), encodeEnvelope(t, &tt.env)) {
				t.Error("binary form differs after the JSON round trip")
			}
		})
	}
}

// A datagram is the version byte, then the slot's string8 type tag and payload, with
// no outer type tag.
func TestEnvelopeWireLayout(t *testing.T) {
	wire := encodeEnvelope(t, &Envelope{Version: EnvelopeVersion, Object: astral.NewString8("hi")})

	want := []byte{1, 7, 's', 't', 'r', 'i', 'n', 'g', '8', 2, 'h', 'i'}
	if !bytes.Equal(wire, want) {
		t.Errorf("wire = %x; want %x", wire, want)
	}
}

// ReadFrom stops at the end of the Envelope, so a receiver detects trailing bytes by
// comparing the count with the datagram length.
func TestEnvelopeReadFromStopsAtItsEnd(t *testing.T) {
	wire := encodeEnvelope(t, &Envelope{Version: EnvelopeVersion, Object: astral.NewString8("hi")})
	datagram := append(append([]byte(nil), wire...), 0xde, 0xad)

	var got Envelope
	n, err := got.ReadFrom(bytes.NewReader(datagram))
	if err != nil {
		t.Fatalf("ReadFrom: %v", err)
	}
	if n != int64(len(wire)) {
		t.Errorf("ReadFrom reported %d bytes; the Envelope holds %d", n, len(wire))
	}
}

// An old mod.ether.signed_broadcast datagram begins with the high byte of a UnixNano
// timestamp, which is never the current Envelope version.
func TestEnvelopeVersionRejectsOldBroadcasts(t *testing.T) {
	for year := 2023; year <= 2034; year++ {
		for _, at := range []time.Time{
			time.Date(year, time.January, 1, 0, 0, 0, 0, time.UTC),
			time.Date(year, time.December, 31, 23, 59, 59, 0, time.UTC),
		} {
			high := byte(uint64(at.UnixNano()) >> 56)
			if high < 0x17 || high > 0x1c {
				t.Errorf("%v: high byte %#x outside 0x17-0x1c", at, high)
			}
			if high == EnvelopeVersion {
				t.Errorf("%v: high byte %#x equals the Envelope version", at, high)
			}
		}
	}
}

// nestedBundles returns a string8 type tag and payload of depth bundles, each holding
// the next.
func nestedBundles(t *testing.T, depth int) []byte {
	t.Helper()

	var payload bytes.Buffer
	if _, err := astral.Uint32(0).WriteTo(&payload); err != nil {
		t.Fatal(err)
	}

	for range depth {
		var member bytes.Buffer
		if _, err := astral.String8("bundle").WriteTo(&member); err != nil {
			t.Fatal(err)
		}
		member.Write(payload.Bytes())

		payload.Reset()
		if _, err := astral.Uint32(1).WriteTo(&payload); err != nil {
			t.Fatal(err)
		}
		if _, err := astral.Bytes32(member.Bytes()).WriteTo(&payload); err != nil {
			t.Fatal(err)
		}
	}

	var out bytes.Buffer
	if _, err := astral.String8("bundle").WriteTo(&out); err != nil {
		t.Fatal(err)
	}
	out.Write(payload.Bytes())
	return out.Bytes()
}

// The slot decodes through the caller's reader, so the depth budget of the call
// bounds what the datagram carries.
func TestEnvelopeObjectSharesTheDepthBudget(t *testing.T) {
	datagram := append([]byte{EnvelopeVersion}, nestedBundles(t, 400)...)

	var got Envelope
	_, err := got.ReadFrom(bytes.NewReader(datagram))
	if !errors.Is(err, astral.ErrDepthExceeded) {
		t.Fatalf("ReadFrom of 400 nested bundles = %v; want ErrDepthExceeded", err)
	}

	shallow := append([]byte{EnvelopeVersion}, nestedBundles(t, 4)...)
	if _, err := got.ReadFrom(bytes.NewReader(shallow)); err != nil {
		t.Fatalf("ReadFrom of 4 nested bundles: %v", err)
	}
}

// The slot resolves its type in the caller's Blueprints registry.
func TestEnvelopeObjectUsesTheCallersRegistry(t *testing.T) {
	data, err := astral.EncodeBytes(&Envelope{Version: EnvelopeVersion, Object: astral.NewString8("hi")})
	if err != nil {
		t.Fatalf("EncodeBytes: %v", err)
	}

	without := astral.NewBlueprints(nil)
	if err := without.Add(&Envelope{}); err != nil {
		t.Fatal(err)
	}
	if _, err := astral.DecodeAs[*Envelope](data, astral.WithBlueprints(without)); !errors.Is(err, astral.ErrBlueprintNotFound) {
		t.Errorf("decode with a registry lacking string8 = %v; want ErrBlueprintNotFound", err)
	}

	with := astral.NewBlueprints(nil)
	if err := with.Add(&Envelope{}, astral.NewString8("")); err != nil {
		t.Fatal(err)
	}
	got, err := astral.DecodeAs[*Envelope](data, astral.WithBlueprints(with))
	if err != nil {
		t.Fatalf("decode with a registry holding string8: %v", err)
	}
	if s, ok := got.Object.(*astral.String8); !ok || *s != "hi" {
		t.Errorf("Object = %#v; want *astral.String8 \"hi\"", got.Object)
	}
}

// Malformed datagrams return an error and never panic.
func TestEnvelopeRejectsMalformedDatagrams(t *testing.T) {
	wire := encodeEnvelope(t, &Envelope{Version: EnvelopeVersion, Object: sampleBundle(t)})

	tests := []struct {
		name     string
		datagram []byte
	}{
		{"empty", nil},
		{"version only", []byte{EnvelopeVersion}},
		{"unknown type", append([]byte{EnvelopeVersion, 4}, "nope"...)},
		{"truncated", wire[:len(wire)-1]},
		{"type tag past the end", []byte{EnvelopeVersion, 200, 'x'}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var got Envelope
			if _, err := got.ReadFrom(bytes.NewReader(tt.datagram)); err == nil {
				t.Errorf("ReadFrom(%x) = nil error; want an error", tt.datagram)
			}
		})
	}

	for cut := range wire {
		var got Envelope
		if _, err := got.ReadFrom(bytes.NewReader(wire[:cut])); err == nil {
			t.Errorf("ReadFrom of the first %d bytes = nil error; want an error", cut)
		}
	}
}
