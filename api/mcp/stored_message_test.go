package mcp

import (
	"bytes"
	"encoding/json"
	"testing"
	"time"

	"github.com/astralp2p/astral-go/astral"
	"github.com/astralp2p/astral-go/astral/channel"
)

func ptrTime(t time.Time) *astral.Time {
	v := astral.Time(t)
	return &v
}

func sampleStoredMessage() *StoredMessage {
	return &StoredMessage{
		Cursor:     42,
		ID:         NewMessageID(),
		Box:        BoxInbox,
		Sender:     astral.GenerateIdentity(),
		Recipient:  astral.GenerateIdentity(),
		Content:    "the index is rebuilt",
		ParentID:   NewMessageID(),
		CreatedAt:  astral.Time(timeWithNanos()),
		ReadAt:     ptrTime(timeWithNanos()),
		ArchivedAt: ptrTime(timeWithNanos()),
	}
}

func TestStoredMessage_BinaryRoundTrip(t *testing.T) {
	src := sampleStoredMessage()

	data, err := astral.EncodeBytes(src)
	if err != nil {
		t.Fatalf("encode: %v", err)
	}

	obj, _, err := astral.Decode(bytes.NewReader(data))
	if err != nil {
		t.Fatalf("decode: %v", err)
	}

	dst, ok := obj.(*StoredMessage)
	if !ok {
		t.Fatalf("want *StoredMessage, got %T", obj)
	}
	if dst.Cursor != src.Cursor {
		t.Fatalf("cursor: want %v, got %v", src.Cursor, dst.Cursor)
	}
	if dst.ID != src.ID || dst.ParentID != src.ParentID {
		t.Fatalf("ids: want %v/%v, got %v/%v", src.ID, src.ParentID, dst.ID, dst.ParentID)
	}
	if dst.Box != src.Box {
		t.Fatalf("box: want %v, got %v", src.Box, dst.Box)
	}
	if !dst.Sender.IsEqual(src.Sender) || !dst.Recipient.IsEqual(src.Recipient) {
		t.Fatal("the parties did not survive the round trip")
	}
	if dst.Content != src.Content {
		t.Fatalf("content: want %v, got %v", src.Content, dst.Content)
	}
	if !dst.CreatedAt.Time().Equal(src.CreatedAt.Time()) {
		t.Fatalf("created: want %v, got %v", src.CreatedAt.Time(), dst.CreatedAt.Time())
	}
}

// An unset instant is the absence of the fact. A row that names no collection
// must decode as one, so a reader cannot mistake a zero for a stamp.
func TestStoredMessage_AnUnsetInstantSurvivesAsUnset(t *testing.T) {
	src := &StoredMessage{
		ID:        NewMessageID(),
		Box:       BoxOutbox,
		Sender:    astral.GenerateIdentity(),
		Recipient: astral.GenerateIdentity(),
		Content:   "sent, fate unknown",
		CreatedAt: astral.Time(timeWithNanos()),
	}

	data, err := astral.EncodeBytes(src)
	if err != nil {
		t.Fatalf("encode: %v", err)
	}

	obj, _, err := astral.Decode(bytes.NewReader(data))
	if err != nil {
		t.Fatalf("decode: %v", err)
	}

	dst := obj.(*StoredMessage)
	for name, got := range map[string]*astral.Time{
		"landed_at":  dst.LandedAt,
		"failed_at":  dst.FailedAt,
		"fetched_at": dst.FetchedAt,
		"read_at":    dst.ReadAt,
	} {
		if got != nil {
			t.Fatalf("%v decoded as %v, where nothing stamped it", name, got.Time())
		}
	}
	if dst.Err != nil {
		t.Fatalf("err decoded as %q, where the node refused nothing", *dst.Err)
	}
}

// The record travels the same JSON channel the agent records do, which is the
// path an op under out=json takes.
func TestStoredMessage_JSONChannelRoundTrip(t *testing.T) {
	src := sampleStoredMessage()

	var buf bytes.Buffer
	if err := channel.NewJSONSender(&buf).Send(src); err != nil {
		t.Fatal(err)
	}

	obj, err := channel.NewJSONReceiver(bytes.NewReader(buf.Bytes())).Receive()
	if err != nil {
		t.Fatalf("receive %q: %v", buf.String(), err)
	}

	dst, ok := obj.(*StoredMessage)
	if !ok {
		t.Fatalf("want *StoredMessage, got %T", obj)
	}
	if dst.ID != src.ID {
		t.Fatalf("id: want %v, got %v", src.ID, dst.ID)
	}
	if dst.ReadAt == nil || !dst.ReadAt.Time().Equal(src.ReadAt.Time()) {
		t.Fatalf("read_at: want %v, got %v", src.ReadAt.Time(), dst.ReadAt.Time())
	}
}

// Message is the frame and carries no party; StoredMessage is the record and
// carries both. A field crossing that line is the mistake the pair exists to
// prevent.
func TestTheFrameNamesNoPartyAndTheRecordNamesBoth(t *testing.T) {
	frame, err := json.Marshal(&Message{ID: NewMessageID(), Content: "x"})
	if err != nil {
		t.Fatal(err)
	}
	var frameFields map[string]json.RawMessage
	if err = json.Unmarshal(frame, &frameFields); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"Sender", "Recipient", "Box", "CreatedAt"} {
		if _, found := frameFields[name]; found {
			t.Fatalf("Message carries %v: %s", name, frame)
		}
	}

	record, err := json.Marshal(sampleStoredMessage())
	if err != nil {
		t.Fatal(err)
	}
	var recordFields map[string]json.RawMessage
	if err = json.Unmarshal(record, &recordFields); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"Sender", "Recipient", "Box", "CreatedAt"} {
		if _, found := recordFields[name]; !found {
			t.Fatalf("StoredMessage carries no %v: %s", name, record)
		}
	}
}
