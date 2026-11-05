package astral

import (
	"bytes"
	"encoding/binary"
	"testing"
)

// claim64 builds a payload whose 64-bit length prefix names size bytes and whose body
// carries body.
func claim64(size uint64, body []byte) []byte {
	var buf bytes.Buffer
	binary.Write(&buf, ByteOrder, size)
	buf.Write(body)
	return buf.Bytes()
}

// A String32 or String64 length prefix is the peer's claim. Reserving it before the
// bytes arrived let a 61-byte input claiming 2^40 bytes end the process with an
// out-of-memory fault that no recover catches.
func TestLongStrings_ReadFromDoesNotReserveAnUnarrivedPayload(t *testing.T) {
	body := bytes.Repeat([]byte("x"), 53)

	tests := []struct {
		name    string
		payload []byte
		read    func(*bytes.Reader) error
	}{
		{"string32 claiming 2^32-1", claim(1<<32-1, body), func(r *bytes.Reader) error {
			var s String32
			_, err := s.ReadFrom(r)
			return err
		}},
		{"string64 claiming 2^40", claim64(1<<40, body), func(r *bytes.Reader) error {
			var s String64
			_, err := s.ReadFrom(r)
			return err
		}},
		{"string64 claiming 2^63+1", claim64(1<<63+1, body), func(r *bytes.Reader) error {
			var s String64
			_, err := s.ReadFrom(r)
			return err
		}},
		{"string64 claiming 2^64-1", claim64(1<<64-1, body), func(r *bytes.Reader) error {
			var s String64
			_, err := s.ReadFrom(r)
			return err
		}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var err error
			used := allocatedBy(func() {
				err = tt.read(bytes.NewReader(tt.payload))
			})

			if err == nil {
				t.Fatal("want an error for a payload shorter than its length prefix")
			}
			if used > allocBudget {
				t.Errorf("allocated %d bytes for a %d-byte payload", used, len(tt.payload))
			}
		})
	}
}

// The destination keeps its previous value when the payload is short.
func TestLongStrings_ShortReadLeavesTheDestination(t *testing.T) {
	s32 := String32("original")
	if _, err := s32.ReadFrom(bytes.NewReader(claim(10, []byte("abcd")))); err == nil {
		t.Error("string32: want an error for a short payload")
	}
	if s32 != "original" {
		t.Errorf("string32: want the destination unchanged, got %q", s32)
	}

	s64 := String64("original")
	if _, err := s64.ReadFrom(bytes.NewReader(claim64(10, []byte("abcd")))); err == nil {
		t.Error("string64: want an error for a short payload")
	}
	if s64 != "original" {
		t.Errorf("string64: want the destination unchanged, got %q", s64)
	}
}

func TestLongStrings_RoundTripHonestPayloads(t *testing.T) {
	big := string(bytes.Repeat([]byte("ab"), growChunk))

	for _, want := range []string{"", "hello", big} {
		var buf bytes.Buffer
		if _, err := String32(want).WriteTo(&buf); err != nil {
			t.Fatalf("string32 WriteTo: %v", err)
		}
		if _, err := String64(want).WriteTo(&buf); err != nil {
			t.Fatalf("string64 WriteTo: %v", err)
		}
		total := int64(buf.Len())

		var got32 String32
		var got64 String64
		n32, err := got32.ReadFrom(&buf)
		if err != nil {
			t.Fatalf("string32 ReadFrom: %v", err)
		}
		n64, err := got64.ReadFrom(&buf)
		if err != nil {
			t.Fatalf("string64 ReadFrom: %v", err)
		}

		if string(got32) != want || string(got64) != want {
			t.Errorf("round trip of a %d-byte string differs", len(want))
		}
		if n32+n64 != total {
			t.Errorf("read %d bytes; wrote %d", n32+n64, total)
		}
	}
}
