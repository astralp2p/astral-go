package mcp

import (
	"bytes"
	"testing"

	"github.com/astralp2p/astral-go/astral"
)

func TestReceipt_BinaryRoundTrip(t *testing.T) {
	src := &Receipt{ID: mustParseID("7f3a1c9e5b024d6810af2e7c94b5d3a6")}

	data, err := astral.EncodeBytes(src)
	if err != nil {
		t.Fatalf("encode: %v", err)
	}

	obj, _, err := astral.Decode(bytes.NewReader(data))
	if err != nil {
		t.Fatalf("decode: %v", err)
	}

	dst, ok := obj.(*Receipt)
	if !ok {
		t.Fatalf("want *Receipt, got %T", obj)
	}
	if dst.ID != src.ID {
		t.Fatalf("id: want %v, got %v", src.ID, dst.ID)
	}
}

// A Receipt and a Message are distinct wire types, so a node that reads one
// where it expects the other rejects rather than mistaking the id.
func TestReceipt_IsNotAMessage(t *testing.T) {
	if (&Receipt{}).ObjectType() == (&Message{}).ObjectType() {
		t.Fatal("receipt and message share an object type")
	}
}
