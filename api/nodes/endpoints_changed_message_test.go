package nodes

import (
	"encoding/json"
	"testing"

	"github.com/astralp2p/astral-go/astral"
)

func TestEndpointsChangedMessage_BinaryRoundTrip(t *testing.T) {
	orig := EndpointsChangedMessage{Subject: astral.GenerateIdentity()}

	data, err := astral.EncodeBytes(&orig)
	if err != nil {
		t.Fatalf("encode: %v", err)
	}

	got, err := astral.DecodeAs[*EndpointsChangedMessage](data)
	if err != nil {
		t.Fatalf("decode: %v", err)
	}

	if !got.Subject.IsEqual(orig.Subject) {
		t.Fatalf("Subject = %v; want %v", got.Subject, orig.Subject)
	}
}

func TestEndpointsUpdatedEvent_BinaryRoundTrip(t *testing.T) {
	orig := EndpointsUpdatedEvent{Identity: astral.GenerateIdentity()}

	data, err := astral.EncodeBytes(&orig)
	if err != nil {
		t.Fatalf("encode: %v", err)
	}

	got, err := astral.DecodeAs[*EndpointsUpdatedEvent](data)
	if err != nil {
		t.Fatalf("decode: %v", err)
	}

	if !got.Identity.IsEqual(orig.Identity) {
		t.Fatalf("Identity = %v; want %v", got.Identity, orig.Identity)
	}
}

// TestEndpointsChangedMessage_JSONField: the JSON form names the field Subject, as the spec's example does.
func TestEndpointsChangedMessage_JSONField(t *testing.T) {
	id := astral.GenerateIdentity()
	b, err := json.Marshal(EndpointsChangedMessage{Subject: id})
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var m map[string]string
	if err := json.Unmarshal(b, &m); err != nil {
		t.Fatalf("unmarshal %s: %v", b, err)
	}
	if m["Subject"] != id.String() {
		t.Fatalf("JSON = %s; want Subject %v", b, id)
	}
}
