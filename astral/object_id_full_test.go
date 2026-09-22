package astral

import (
	"crypto/sha256"
	"math"
	"strings"
	"testing"
)

// maxFullID encodes to the longest data1 body: its Size has no leading zero bits, so the
// encoder strips no 'y'.
var maxFullID = ObjectID{Size: math.MaxUint64, Hash: sha256.Sum256([]byte("hello"))}

func TestParseID_AcceptsTheLongestFullBody(t *testing.T) {
	s := maxFullID.String()
	if n := len(strings.TrimPrefix(s, idPrefix)); n != fullIDBodyLen {
		t.Fatalf("want a %d-character body, got %d", fullIDBodyLen, n)
	}

	if id := mustParseID(t, s); !id.IsEqual(&maxFullID) {
		t.Errorf("want %s, got %s", &maxFullID, id)
	}
}

// A body above 64 characters used to decode past the 40-byte buffer and panic from 72
// characters on.
func TestParseID_RejectsAFullBodyAbove64Characters(t *testing.T) {
	body := strings.TrimPrefix(maxFullID.String(), idPrefix)

	cases := map[string]string{
		"65 characters":           strings.Repeat("y", 65),
		"72 characters":           strings.Repeat("y", 72),
		"80 characters":           strings.Repeat("y", 80),
		"128 characters":          strings.Repeat("y", 128),
		"a valid body behind 8 y": strings.Repeat("y", 8) + body,
		// The decoder skips '\n' and '\r', so these decoded to maxFullID before the bound.
		"a valid body and a \\n": body + "\n",
		"8 embedded \\r":         body[:30] + strings.Repeat("\r", 8) + body[30:],
	}

	for name, b := range cases {
		t.Run(name, func(t *testing.T) {
			id, err := ParseID(idPrefix + b)
			if err == nil {
				t.Fatalf("want an error, got %s", id)
			}
			if id != nil {
				t.Errorf("want a nil id beside the error, got %s", id)
			}
		})
	}
}

func TestObjectID_DecodersRejectAFullBodyAbove64Characters(t *testing.T) {
	s := idPrefix + strings.Repeat("y", 72)

	var id ObjectID
	if err := id.UnmarshalJSON([]byte(`"` + s + `"`)); err == nil {
		t.Error("UnmarshalJSON: want an error")
	}
	if err := id.UnmarshalText([]byte(s)); err == nil {
		t.Error("UnmarshalText: want an error")
	}
	if err := id.Scan(s); err == nil {
		t.Error("Scan: want an error")
	}
}
