package mcp

import (
	"bytes"
	"testing"

	"github.com/astralp2p/astral-go/astral"
)

func sampleMessage() *Message {
	return &Message{
		ID:      astral.String8("7f3a1c9e5b024d6810af2e7c94b5d3a6"),
		Topic:   astral.String8("build"),
		Content: astral.String32("the index is rebuilt"),
		ReplyTo: astral.String8("0b1d4e8a26c37f905ea1c4b83d76f2e5"),
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
	if dst.Topic != src.Topic {
		t.Fatalf("topic: want %v, got %v", src.Topic, dst.Topic)
	}
	if dst.Content != src.Content {
		t.Fatalf("content: want %v, got %v", src.Content, dst.Content)
	}
	if dst.ReplyTo != src.ReplyTo {
		t.Fatalf("reply_to: want %v, got %v", src.ReplyTo, dst.ReplyTo)
	}
}

// A message that answers nothing carries an empty ReplyTo, and the empty
// string is the absence — not a field the encoding may drop.
func TestMessage_EmptyReplyToSurvives(t *testing.T) {
	src := sampleMessage()
	src.ReplyTo = ""

	if dst := roundTrip(t, src); dst.ReplyTo != "" {
		t.Fatalf("reply_to: want empty, got %v", dst.ReplyTo)
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
