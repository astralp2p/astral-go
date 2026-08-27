package mcp

import (
	"bytes"
	"testing"

	"github.com/astralp2p/astral-go/astral"
)

func mustParseID(s string) MessageID {
	id, err := ParseMessageID(s)
	if err != nil {
		panic(err)
	}
	return id
}

func sampleMessage() *Message {
	return &Message{
		ID:      mustParseID("7f3a1c9e5b024d6810af2e7c94b5d3a6"),
		Content: astral.String32("the index is rebuilt"),
	}
}

func roundTrip(t *testing.T, src *Message) *Message {
	t.Helper()

	data, err := astral.EncodeBytes(src)
	if err != nil {
		t.Fatalf("encode: %v", err)
	}

	obj, _, err := astral.Decode(bytes.NewReader(data))
	if err != nil {
		t.Fatalf("decode: %v", err)
	}

	dst, ok := obj.(*Message)
	if !ok {
		t.Fatalf("want *Message, got %T", obj)
	}
	return dst
}

func TestMessage_BinaryRoundTrip(t *testing.T) {
	src := sampleMessage()
	dst := roundTrip(t, src)

	if dst.ID != src.ID {
		t.Fatalf("id: want %v, got %v", src.ID, dst.ID)
	}
	if dst.Content != src.Content {
		t.Fatalf("content: want %v, got %v", src.Content, dst.Content)
	}
}

// Content is the only field that carries a body, and String32 is what lets it
// hold one larger than a String8 field could.
func TestMessage_ContentHoldsALargeBody(t *testing.T) {
	src := sampleMessage()
	src.Content = astral.String32(bytes.Repeat([]byte("x"), 64<<10))

	if dst := roundTrip(t, src); dst.Content != src.Content {
		t.Fatalf("content: want %v bytes, got %v", len(src.Content), len(dst.Content))
	}
}
