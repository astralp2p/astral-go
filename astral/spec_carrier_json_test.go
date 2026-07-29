package astral

import (
	"encoding/json"
	"strings"
	"testing"
)

// specCarriers is the closed set of Spec implementers, each populated so every field
// carries a distinguishable non-zero value.
func specCarriers() []Spec {
	return []Spec{
		&PrimitiveSpec{PrimitiveType: "uint32"},
		&RefSpec{Type: "some.type"},
		&SliceSpec{Type: "uint8"},
		&ArraySpec{Type: "uint8", Length: 4},
		&MapSpec{KeyType: "string16", ValueType: "uint8"},
		&PtrSpec{Type: "some.type"},
		&ObjectSpec{},
	}
}

// TestSpecCarrier_JSONRejectsExcessKeys pins carrier payloads being strict.
//
// The carriers had no UnmarshalJSON, so encoding/json filled them reflectively and
// discarded keys it did not recognise. A misspelled key inside the {"Type","Object"}
// envelope — "Len" for ArraySpec.Length — decoded with err == nil to a zero-valued
// carrier, registering a corrupted schema silently and permanently. Blueprint and Field
// were already strict at their own level; this closes the level below them.
func TestSpecCarrier_JSONRejectsExcessKeys(t *testing.T) {
	tests := []struct {
		name    string
		typ     string
		payload string
	}{
		{"array_spec Length misspelled as Len", "astral.blueprint.array_spec", `{"Type":"uint8","Len":4}`},
		{"slice_spec unknown key", "astral.blueprint.slice_spec", `{"Type":"uint8","Nope":1}`},
		{"primitive_spec unknown key", "astral.blueprint.primitive_spec", `{"PrimitiveType":"uint32","Extra":1}`},
		{"ref_spec unknown key", "astral.blueprint.ref_spec", `{"Type":"some.type","Extra":1}`},
		{"ptr_spec unknown key", "astral.blueprint.ptr_spec", `{"Type":"some.type","Extra":1}`},
		{"map_spec ValueType misspelled", "astral.blueprint.map_spec", `{"KeyType":"string16","ValType":"uint8"}`},
		{"object_spec takes no keys at all", "astral.blueprint.object_spec", `{"Type":"uint8"}`},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			doc := `{"Name":"x","Spec":{"Type":"` + test.typ + `","Object":` + test.payload + `}}`

			var field Field
			err := json.Unmarshal([]byte(doc), &field)
			if err == nil {
				t.Fatalf("decoded a misspelled payload without error, got spec %#v", field.Spec)
			}
			if !strings.Contains(err.Error(), "excess fields") {
				t.Fatalf("error: want an excess-fields rejection, got %v", err)
			}
		})
	}
}

// TestSpecCarrier_JSONRoundTrip pins the strictness not costing well-formed input: every
// carrier still survives a round trip in its real wire position, the Spec slot of a Field.
func TestSpecCarrier_JSONRoundTrip(t *testing.T) {
	for _, carrier := range specCarriers() {
		t.Run(carrier.ObjectType(), func(t *testing.T) {
			encoded, err := json.Marshal(&Field{Name: "x", Spec: carrier})
			if err != nil {
				t.Fatalf("Marshal returned error: %v", err)
			}

			var decoded Field
			if err = json.Unmarshal(encoded, &decoded); err != nil {
				t.Fatalf("Unmarshal returned error: %v (from %s)", err, encoded)
			}

			if decoded.Spec == nil {
				t.Fatalf("Spec decoded to nil from %s", encoded)
			}
			if got := decoded.Spec.ObjectType(); got != carrier.ObjectType() {
				t.Fatalf("ObjectType: want %q, got %q", carrier.ObjectType(), got)
			}

			// Re-encoding must reproduce the same document: the values survived, not just
			// the type tag. A carrier that decoded to its zero value fails here.
			reencoded, err := json.Marshal(&decoded)
			if err != nil {
				t.Fatalf("re-Marshal returned error: %v", err)
			}
			if string(reencoded) != string(encoded) {
				t.Fatalf("round trip changed the document:\n want %s\n  got %s", encoded, reencoded)
			}
		})
	}
}

// TestSpecCarrier_JSONWireFormUnchanged pins the emitted document against its literal bytes.
//
// Adding MarshalJSON to the carriers routes standalone emission through Objectify, which
// orders keys alphabetically. In the wire position the payload already went through
// Objectify via the Field's interface slot, so these bytes are unchanged by that switch —
// this test is what makes that claim checkable rather than asserted.
func TestSpecCarrier_JSONWireFormUnchanged(t *testing.T) {
	tests := []struct {
		spec Spec
		want string
	}{
		{&ArraySpec{Type: "uint8", Length: 4}, `{"Name":"x","Spec":{"Type":"astral.blueprint.array_spec","Object":{"Length":4,"Type":"uint8"}}}`},
		{&MapSpec{KeyType: "string16", ValueType: "uint8"}, `{"Name":"x","Spec":{"Type":"astral.blueprint.map_spec","Object":{"KeyType":"string16","ValueType":"uint8"}}}`},
		{&PrimitiveSpec{PrimitiveType: "uint32"}, `{"Name":"x","Spec":{"Type":"astral.blueprint.primitive_spec","Object":{"PrimitiveType":"uint32"}}}`},
		{&ObjectSpec{}, `{"Name":"x","Spec":{"Type":"astral.blueprint.object_spec","Object":{}}}`},
	}

	for _, test := range tests {
		t.Run(test.spec.ObjectType(), func(t *testing.T) {
			got, err := json.Marshal(&Field{Name: "x", Spec: test.spec})
			if err != nil {
				t.Fatalf("Marshal returned error: %v", err)
			}
			if string(got) != test.want {
				t.Fatalf("wire form:\n want %s\n  got %s", test.want, got)
			}
		})
	}
}

// TestSpecCarrier_BinaryUnaffected pins the binary codec being untouched by the JSON
// change: WriteTo/ReadFrom already routed through Objectify and keep their bytes.
func TestSpecCarrier_BinaryUnaffected(t *testing.T) {
	for _, carrier := range specCarriers() {
		t.Run(carrier.ObjectType(), func(t *testing.T) {
			var buf strings.Builder
			if _, err := carrier.WriteTo(&buf); err != nil {
				t.Fatalf("WriteTo returned error: %v", err)
			}

			decoded := New(carrier.ObjectType())
			if decoded == nil {
				t.Fatalf("New returned nil for %q", carrier.ObjectType())
			}
			if _, err := decoded.ReadFrom(strings.NewReader(buf.String())); err != nil {
				t.Fatalf("ReadFrom returned error: %v", err)
			}

			var out strings.Builder
			if _, err := decoded.WriteTo(&out); err != nil {
				t.Fatalf("re-WriteTo returned error: %v", err)
			}
			if out.String() != buf.String() {
				t.Fatalf("binary round trip changed the bytes: want %x, got %x", buf.String(), out.String())
			}
		})
	}
}
