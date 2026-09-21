package astral

import (
	"crypto/sha256"
	"encoding/binary"
	"strings"
	"testing"
)

// The published vector for an object whose Size is 71, and its partial form
// (primitive-types/object_id.sha256.md). The partial body starts with 'b'.
const (
	vectorFullID    = "data1rxqff36hhoddbhwbsd5c1smbpoh9oq5pgum6n6g4bg1esia4psp1r"
	vectorPartialID = "data0bqff36hhoddbhwbsd5c1smbpoh9oq5pgum6n6g4bg1esia4psp1r"
	vectorSize      = 71
)

// The published partial form of helloUntypedID (primitive-types/object_id.sha256.md). The
// body starts with 'y', which the partial form never strips.
const helloPartialID = "data0ym81js7f9cfdbauqoq3kash6f8o5naxfa878ejx8gbbuckjazgbr"

func mustParseID(t *testing.T, s string) *ObjectID {
	t.Helper()

	id, err := ParseID(s)
	if err != nil {
		t.Fatalf("ParseID(%q): %v", s, err)
	}
	return id
}

func TestObjectID_PartialStringMatchesThePublishedVector(t *testing.T) {
	full := mustParseID(t, vectorFullID)

	if full.Size != vectorSize {
		t.Errorf("want size %d, got %d", vectorSize, full.Size)
	}
	if got := full.PartialString(); got != vectorPartialID {
		t.Errorf("want the published vector\n  %s\ngot\n  %s", vectorPartialID, got)
	}

	partial := mustParseID(t, vectorPartialID)
	if partial.Size != 0 {
		t.Errorf("want size 0 from the partial form, got %d", partial.Size)
	}
	if partial.Hash != full.Hash {
		t.Errorf("want the full id's hash\n  % x\ngot\n  % x", full.Hash, partial.Hash)
	}
}

func TestObjectID_PartialStringMatchesTheHelloVector(t *testing.T) {
	blob := Blob("hello")

	id, err := ResolveObjectID(&blob)
	if err != nil {
		t.Fatalf("ResolveObjectID: %v", err)
	}

	if got := id.PartialString(); got != helloPartialID {
		t.Errorf("want\n  %s\ngot\n  %s", helloPartialID, got)
	}

	partial := mustParseID(t, helloPartialID)
	if !partial.IsEqual(&ObjectID{Hash: id.Hash}) {
		t.Errorf("want size 0 and the hello hash, got %d % x", partial.Size, partial.Hash)
	}
}

// The data0 body is characters [12:64] of the unstripped encoding of a zero Size and the
// Hash, so its first character is 'y' or 'b'. A thousand hashes produce both.
func TestObjectID_PartialStringRoundTrips(t *testing.T) {
	firstChars := map[byte]int{}

	for i := range 1000 {
		var seed [8]byte
		binary.BigEndian.PutUint64(seed[:], uint64(i))
		src := ObjectID{Size: uint64(i) + 1, Hash: sha256.Sum256(seed[:])}

		s := src.PartialString()
		if len(s) != len(partialIDPrefix)+52 || !strings.HasPrefix(s, partialIDPrefix) {
			t.Fatalf("want data0 and a 52-character body, got %q", s)
		}
		firstChars[s[len(partialIDPrefix)]]++

		got := mustParseID(t, s)
		if !got.IsEqual(&ObjectID{Hash: src.Hash}) {
			t.Fatalf("%s: want size 0 and hash % x, got %d % x", s, src.Hash, got.Size, got.Hash)
		}
	}

	if len(firstChars) != 2 || firstChars['y'] == 0 || firstChars['b'] == 0 {
		t.Errorf("want only 'y' and 'b' as first body characters, got %v", firstChars)
	}
}

// String strips every leading 'y'; PartialString keeps all 52 body characters.
func TestObjectID_PartialStringKeepsLeadingY(t *testing.T) {
	var src ObjectID
	src.Hash[31] = 0x01

	s := src.PartialString()
	want := partialIDPrefix + strings.Repeat("y", 51) + "b"
	if s != want {
		t.Fatalf("want\n  %s\ngot\n  %s", want, s)
	}
	if full := src.String(); full != idPrefix+"b" {
		t.Errorf("want the stripped data1 form %sb, got %s", idPrefix, full)
	}

	if got := mustParseID(t, s); !got.IsEqual(&src) {
		t.Errorf("want %s, got %s", &src, got)
	}
}

func TestParseID_RejectsMalformedPartialBodies(t *testing.T) {
	body := strings.TrimPrefix(vectorPartialID, partialIDPrefix)

	// The decoder reads a tail of 1, 3, 4 or 6 '=' as padding and skips '\n' and '\r'.
	// Those bodies decode without error to fewer than 40 bytes, and a 52-character body
	// followed by '\n' decodes to exactly 40.
	cases := map[string]string{
		"empty body":      "",
		"51 characters":   body[:51],
		"53 characters":   body + "y",
		"53 with a \\n":   body + "\n",
		"60 characters":   body + body[:8],
		"uppercase":       strings.ToUpper(body),
		"uppercase tail":  body[:1] + strings.ToUpper(body[1:]),
		"not in alphabet": body[:51] + "l",
		"= tail of 1":     body[:51] + "=",
		"= tail of 3":     body[:49] + "===",
		"= tail of 4":     body[:48] + "====",
		"= tail of 6":     body[:46] + "======",
		"8 embedded \\n":  body[:22] + strings.Repeat("\n", 8) + body[22:44],
		"8 embedded \\r":  body[:22] + strings.Repeat("\r", 8) + body[22:44],
	}
	// Every other first character sets one of the last four Size bits.
	for _, c := range zBase32CharSet[2:] {
		cases["size bits in first character "+string(c)] = string(c) + body[1:]
	}

	for name, b := range cases {
		t.Run(name, func(t *testing.T) {
			id, err := ParseID(partialIDPrefix + b)
			if err == nil {
				t.Fatalf("want an error, got %d % x", id.Size, id.Hash)
			}
			if id != nil {
				t.Errorf("want a nil id beside the error, got %s", id)
			}
		})
	}
}

// Encoders emit data1 for every id, including a Size 0 one, and strip its leading 'y'.
func TestObjectID_ZeroSizeEncodesAsData1(t *testing.T) {
	cases := map[string]struct {
		partial string
		want    string
	}{
		"vector": {vectorPartialID, "data1bqff36hhoddbhwbsd5c1smbpoh9oq5pgum6n6g4bg1esia4psp1r"},
		"hello":  {helloPartialID, "data1m81js7f9cfdbauqoq3kash6f8o5naxfa878ejx8gbbuckjazgbr"},
	}

	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			id := mustParseID(t, c.partial)

			if got := id.String(); got != c.want {
				t.Errorf("String: want %s, got %s", c.want, got)
			}

			j, err := id.MarshalJSON()
			if err != nil {
				t.Fatalf("MarshalJSON: %v", err)
			}
			if want := `"` + c.want + `"`; string(j) != want {
				t.Errorf("MarshalJSON: want %s, got %s", want, j)
			}

			txt, err := id.MarshalText()
			if err != nil {
				t.Fatalf("MarshalText: %v", err)
			}
			if string(txt) != c.want {
				t.Errorf("MarshalText: want %s, got %s", c.want, txt)
			}

			val, err := id.Value()
			if err != nil {
				t.Fatalf("Value: %v", err)
			}
			if val != c.want {
				t.Errorf("Value: want %s, got %v", c.want, val)
			}

			if back := mustParseID(t, c.want); !back.IsEqual(id) {
				t.Errorf("data1 input: want %s, got %s", id, back)
			}
		})
	}
}

// The Empty Object's Object ID has Size 0, so it is also a Partial Object ID, and its two
// text forms carry the same 52-character body (primitive-types/object_id.sha256.md).
func TestObjectID_EmptyObjectFullAndPartialFormsAgree(t *testing.T) {
	var blob Blob

	id, err := ResolveObjectID(&blob)
	if err != nil {
		t.Fatalf("ResolveObjectID: %v", err)
	}

	if id.Size != 0 {
		t.Fatalf("want size 0, got %d", id.Size)
	}

	full, partial := id.String(), id.PartialString()
	if strings.TrimPrefix(full, idPrefix) != strings.TrimPrefix(partial, partialIDPrefix) {
		t.Errorf("want one body\n  %s\n  %s", full, partial)
	}

	if got := mustParseID(t, partial); !got.IsEqual(id) {
		t.Errorf("want %s, got %s", id, got)
	}
}

func TestObjectID_DecodersAcceptThePartialForm(t *testing.T) {
	want := ObjectID{Hash: mustParseID(t, vectorFullID).Hash}

	var fromJSON ObjectID
	if err := fromJSON.UnmarshalJSON([]byte(`"` + vectorPartialID + `"`)); err != nil {
		t.Fatalf("UnmarshalJSON: %v", err)
	}
	var fromText ObjectID
	if err := fromText.UnmarshalText([]byte(vectorPartialID)); err != nil {
		t.Fatalf("UnmarshalText: %v", err)
	}
	var fromSQL ObjectID
	if err := fromSQL.Scan(vectorPartialID); err != nil {
		t.Fatalf("Scan: %v", err)
	}

	for name, got := range map[string]ObjectID{"json": fromJSON, "text": fromText, "sql": fromSQL} {
		if !got.IsEqual(&want) {
			t.Errorf("%s: want %s, got %s", name, &want, &got)
		}
	}
}

// A body of 52 'y' characters decodes to the zero id, the value ParseID("data1") returns.
// The ticket "Add partial ObjectID support", open question 3, governs whether it is
// accepted; this test pins the current behaviour of the general rules.
func TestParseID_PartialBodyOfYIsTheZeroID(t *testing.T) {
	s := partialIDPrefix + strings.Repeat("y", 52)

	if got := (ObjectID{}).PartialString(); got != s {
		t.Errorf("want the zero id to render as %s, got %s", s, got)
	}

	id := mustParseID(t, s)
	if !id.IsZero() {
		t.Errorf("want the zero id, got %d % x", id.Size, id.Hash)
	}
	if !id.IsEqual(mustParseID(t, idPrefix)) {
		t.Errorf("want the value of ParseID(%q), got %s", idPrefix, id)
	}
}
