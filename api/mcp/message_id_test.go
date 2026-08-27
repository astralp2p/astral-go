package mcp

import (
	"encoding/json"
	"testing"
)

const sampleID = "7f3a1c9e5b024d6810af2e7c94b5d3a6"

func TestMessageID_TextRoundTrip(t *testing.T) {
	id, err := ParseMessageID(sampleID)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if id.String() != sampleID {
		t.Fatalf("string %v, want %v", id, sampleID)
	}
}

// A minted identifier is 128 bits, and two of them differ.
func TestMessageID_MintedIsDistinct(t *testing.T) {
	first, second := NewMessageID(), NewMessageID()

	if first.IsZero() {
		t.Fatal("a minted id is the zero id")
	}
	if first == second {
		t.Fatalf("two mints answered %v", first)
	}
	if len(first.String()) != 32 {
		t.Fatalf("%v renders %v characters, want 32", first, len(first.String()))
	}
}

// The identifier reaches an agent's model as text it copies back into a tool
// call, so its JSON form is the hex string and not an array of bytes.
func TestMessageID_MarshalsAsHexString(t *testing.T) {
	data, err := json.Marshal(mustParseID(sampleID))
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if string(data) != `"`+sampleID+`"` {
		t.Fatalf("marshalled %s", data)
	}

	var dst MessageID
	if err = json.Unmarshal(data, &dst); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if dst.String() != sampleID {
		t.Fatalf("unmarshalled %v", dst)
	}
}

// The identifier is a column, and a driver hands a text column back as either
// shape.
func TestMessageID_ScansBothColumnShapes(t *testing.T) {
	value, err := mustParseID(sampleID).Value()
	if err != nil {
		t.Fatalf("value: %v", err)
	}

	for _, src := range []any{value, []byte(sampleID)} {
		var dst MessageID
		if err = dst.Scan(src); err != nil {
			t.Fatalf("scan %T: %v", src, err)
		}
		if dst.String() != sampleID {
			t.Fatalf("scan %T answered %v", src, dst)
		}
	}
}

// An identifier a sender made up is refused rather than truncated or padded:
// the store keys on it, and a value it cannot name is not one it can hold.
func TestMessageID_ParseRefusesMalformed(t *testing.T) {
	for _, s := range []string{"", "7f3a", sampleID + "00", "zz3a1c9e5b024d6810af2e7c94b5d3a6"} {
		if _, err := ParseMessageID(s); err == nil {
			t.Fatalf("parse %q answered no error", s)
		}
	}
}
