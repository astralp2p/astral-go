package astral

import (
	"bytes"
	"crypto/sha256"
	"testing"
)

func TestEmptyID_IsTheIDOfNoBytes(t *testing.T) {
	id, err := Resolve(bytes.NewReader(nil))
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}

	if !id.IsEqual(&EmptyID) {
		t.Errorf("want %v, got %v", EmptyID, id)
	}
	if !id.IsEmpty() {
		t.Errorf("want the ID of no bytes to be empty")
	}
}

func TestIsPartial_EmptyIDIsFull(t *testing.T) {
	id := EmptyID
	if id.IsPartial() {
		t.Errorf("want EmptyID to be a full ID")
	}
}

func TestIsPartial_OtherSizeZeroIDsArePartial(t *testing.T) {
	cases := map[string]*ObjectID{
		"nil":       nil,
		"zero":      {},
		"hash only": {Hash: sha256.Sum256([]byte("hello"))},
	}

	for name, id := range cases {
		if !id.IsPartial() {
			t.Errorf("%v: want a partial ID", name)
		}
	}
}

func TestIsPartial_SizedIDIsFull(t *testing.T) {
	id := &ObjectID{Size: 5, Hash: sha256.Sum256([]byte("hello"))}
	if id.IsPartial() {
		t.Errorf("want a sized ID to be full")
	}
	if id.IsEmpty() {
		t.Errorf("want a sized ID not to be empty")
	}
}

func TestEmptyID_TextRoundTrip(t *testing.T) {
	parsed, err := ParseID(EmptyID.String())
	if err != nil {
		t.Fatalf("ParseID: %v", err)
	}
	if !parsed.IsEmpty() || parsed.IsPartial() {
		t.Errorf("want %v back as a full empty ID, got %v", EmptyID, parsed)
	}

	partial, err := ParseID(EmptyID.PartialString())
	if err != nil {
		t.Fatalf("ParseID partial form: %v", err)
	}
	if !partial.IsEmpty() {
		t.Errorf("want the data0 form of the empty hash to parse as EmptyID, got %v", partial)
	}
}
