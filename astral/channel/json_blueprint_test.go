package channel

import (
	"bytes"
	"testing"

	"github.com/astralp2p/astral-go/astral"
)

// TestJSONChannel_Blueprint_RoundTrip pins the objects.register_blueprint input path: the
// Blueprint DESCRIPTOR itself sent through JSONSender and read back through JSONReceiver.
// Before Blueprint grew MarshalJSON/UnmarshalJSON, plain encoding/json could not fill the
// interface-typed Field.Spec slot, so a field-bearing descriptor latched the receiver with
// a decode error and the channel died without output.
func TestJSONChannel_Blueprint_RoundTrip(t *testing.T) {
	src := astral.NewBlueprint("test.channel.json.descriptor",
		astral.Field{Name: "Author", Spec: &astral.PrimitiveSpec{PrimitiveType: "identity"}},
		astral.Field{Name: "Body", Spec: &astral.PrimitiveSpec{PrimitiveType: "string16"}},
		astral.Field{Name: "Refs", Spec: &astral.SliceSpec{Type: "object_id.sha256"}},
	)

	var buf bytes.Buffer
	sender := NewJSONSender(&buf)
	if err := sender.Send(src); err != nil {
		t.Fatalf("send: %v", err)
	}

	recv := NewJSONReceiver(&buf)
	got, err := recv.Receive()
	if err != nil {
		t.Fatalf("receive: %v", err)
	}

	bp, ok := got.(*astral.Blueprint)
	if !ok {
		t.Fatalf("want *astral.Blueprint, got %T", got)
	}
	if bp.Type != src.Type || len(bp.Fields) != 3 {
		t.Fatalf("unexpected blueprint: %+v", bp)
	}
	if s, ok := bp.Fields[0].Spec.(*astral.PrimitiveSpec); !ok || s.PrimitiveType != "identity" {
		t.Fatalf("field 0 spec: %+v", bp.Fields[0].Spec)
	}
	if s, ok := bp.Fields[2].Spec.(*astral.SliceSpec); !ok || s.Type != "object_id.sha256" {
		t.Fatalf("field 2 spec: %+v", bp.Fields[2].Spec)
	}
}
