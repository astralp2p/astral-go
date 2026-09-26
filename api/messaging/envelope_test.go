package messaging

import (
	"reflect"
	"testing"

	"github.com/astralp2p/astral-go/astral"
)

// fullStoredMessage sets every field, so a copy that drops one shows it.
func fullStoredMessage() *StoredMessage {
	m := sampleStoredMessage()
	err := astral.String16("refused by the peer")

	m.ReceiptDueAt = ptrTime(timeWithNanos())
	m.ReceiptStoredAt = ptrTime(timeWithNanos())
	m.LandedAt = ptrTime(timeWithNanos())
	m.FailedAt = ptrTime(timeWithNanos())
	m.FetchedAt = ptrTime(timeWithNanos())
	m.Err = &err
	return m
}

// An envelope is the record less its body, field for field and in the
// record's order. The order is the wire format, so a field added to one and not
// the other, or in another place, fails here.
func TestEnvelope_IsTheRecordLessContent(t *testing.T) {
	var record []string
	for _, f := range reflect.VisibleFields(reflect.TypeOf(StoredMessage{})) {
		if f.Name != "Content" {
			record = append(record, f.Name)
		}
	}

	var envelope []string
	for _, f := range reflect.VisibleFields(reflect.TypeOf(Envelope{})) {
		envelope = append(envelope, f.Name)
	}

	if !reflect.DeepEqual(envelope, record) {
		t.Fatalf("envelope fields %v, want %v", envelope, record)
	}
}

func TestStoredMessage_EnvelopeCopiesEveryFieldButContent(t *testing.T) {
	src := fullStoredMessage()
	env := src.Envelope()

	record := reflect.ValueOf(src).Elem()
	copied := reflect.ValueOf(env).Elem()

	for i := 0; i < copied.NumField(); i++ {
		name := copied.Type().Field(i).Name
		want := record.FieldByName(name)

		if want.IsZero() {
			t.Fatalf("%v is unset in the sample, so the copy proves nothing", name)
		}
		if !reflect.DeepEqual(copied.Field(i).Interface(), want.Interface()) {
			t.Fatalf("%v: want %v, got %v", name, want, copied.Field(i))
		}
	}
}

func TestStoredMessage_EnvelopeOfNilIsNil(t *testing.T) {
	var m *StoredMessage
	if env := m.Envelope(); env != nil {
		t.Fatalf("want nil, got %+v", env)
	}
}

// A listing hands no body out. An envelope must therefore carry no content
// field at all — an empty one reads as a message whose words were empty.
func TestEnvelope_CarriesNoContentField(t *testing.T) {
	fields := jsonFields(t, fullStoredMessage().Envelope())

	if _, found := fields["Content"]; found {
		t.Fatalf("Envelope carries a Content field: %v", fields)
	}
	for _, name := range []string{"Cursor", "ID", "Box", "Sender", "Recipient", "CreatedAt"} {
		if _, found := fields[name]; !found {
			t.Fatalf("Envelope carries no %v field: %v", name, fields)
		}
	}
}

// Every stamp crosses the wire as set, and the unset ones stay unset.
func TestEnvelope_StampsSurviveTheWire(t *testing.T) {
	src := fullStoredMessage().Envelope()
	src.FailedAt, src.Err = nil, nil

	for name, got := range map[string]astral.Object{
		"binary": binaryRoundTrip(t, src),
		"json":   jsonRoundTrip(t, src),
	} {
		dst := got.(*Envelope)
		if dst.FetchedAt == nil || !dst.FetchedAt.Time().Equal(src.FetchedAt.Time()) {
			t.Fatalf("%v: fetched_at want %v, got %v", name, src.FetchedAt.Time(), dst.FetchedAt)
		}
		if dst.FailedAt != nil || dst.Err != nil {
			t.Fatalf("%v: unset failed_at/err decoded as %v/%v", name, dst.FailedAt, dst.Err)
		}
		if !dst.Sender.IsEqual(src.Sender) || !dst.Recipient.IsEqual(src.Recipient) {
			t.Fatalf("%v: the parties did not survive", name)
		}
	}
}
