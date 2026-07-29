package routing

import (
	"bytes"
	"encoding/json"
	"reflect"
	"testing"

	"github.com/astralp2p/astral-go/astral"
	"github.com/astralp2p/astral-go/lib/query"
)

// OpSpec is a wire type: it crosses the ".spec" op to describe an operation to
// a caller, so both codecs must round-trip and the type must be registered.

func sampleOpSpec() *OpSpec {
	return &OpSpec{
		Name: "objects.read",
		Parameters: []query.FieldSpec{
			{Name: "id", Type: "object_id", Required: true},
			{Name: "offset", Type: "uint64"},
		},
	}
}

func TestOpSpec_ObjectType(t *testing.T) {
	want := "routing.op_spec"
	if got := (OpSpec{}).ObjectType(); got != want {
		t.Fatalf("want %q, got %q", want, got)
	}
}

// The type is registered with the default blueprints, so a decoding peer can
// materialise it by name.
func TestOpSpec_IsRegistered(t *testing.T) {
	obj := astral.New((OpSpec{}).ObjectType())

	if obj == nil {
		t.Fatalf("want %q registered, got nil", (OpSpec{}).ObjectType())
	}
	if _, ok := obj.(*OpSpec); !ok {
		t.Fatalf("want *OpSpec, got %T", obj)
	}
}

func TestOpSpec_BinaryRoundTrip(t *testing.T) {
	source := sampleOpSpec()

	var buf bytes.Buffer
	n, err := source.WriteTo(&buf)
	if err != nil {
		t.Fatalf("WriteTo: unexpected error: %v", err)
	}
	if n == 0 {
		t.Fatal("WriteTo: want a non-zero byte count")
	}

	var target OpSpec
	if _, err := target.ReadFrom(&buf); err != nil {
		t.Fatalf("ReadFrom: unexpected error: %v", err)
	}

	if !reflect.DeepEqual(&target, source) {
		t.Fatalf("want %+v, got %+v", source, &target)
	}
}

func TestOpSpec_JSONRoundTrip(t *testing.T) {
	source := sampleOpSpec()

	data, err := json.Marshal(source)
	if err != nil {
		t.Fatalf("Marshal: unexpected error: %v", err)
	}

	var target OpSpec
	if err := json.Unmarshal(data, &target); err != nil {
		t.Fatalf("Unmarshal: unexpected error: %v", err)
	}

	if !reflect.DeepEqual(&target, source) {
		t.Fatalf("want %+v, got %+v", source, &target)
	}
}

// An op with no arguments round-trips as an empty parameter list.
func TestOpSpec_RoundTripsWithoutParameters(t *testing.T) {
	source := &OpSpec{Name: "ping"}

	var buf bytes.Buffer
	if _, err := source.WriteTo(&buf); err != nil {
		t.Fatalf("WriteTo: unexpected error: %v", err)
	}

	var target OpSpec
	if _, err := target.ReadFrom(&buf); err != nil {
		t.Fatalf("ReadFrom: unexpected error: %v", err)
	}

	if target.Name != source.Name {
		t.Fatalf("Name: want %q, got %q", source.Name, target.Name)
	}
	if len(target.Parameters) != 0 {
		t.Fatalf("Parameters: want empty, got %v", target.Parameters)
	}
}
