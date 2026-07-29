package astral

import (
	"encoding/json"
	"strings"
	"testing"
)

// TestJSONEnvelope_RejectsExcessKeys pins the {"Type","Object"} container being strict.
//
// interfaceValue decoded the envelope with plain encoding/json, which ignores an
// unrecognised key, and then skipped the payload because JSONAdapter.Object was nil. A
// misspelled "Object" therefore decoded with err == nil to a zero-valued carrier — a
// heterogeneous slice where a uint32 slice was meant, registering a corrupted schema
// silently. The carrier payloads were closed by TestSpecCarrier_JSONRejectsExcessKeys;
// this closes the envelope above them.
func TestJSONEnvelope_RejectsExcessKeys(t *testing.T) {
	tests := []struct {
		name     string
		envelope string
	}{
		{"Object misspelled as Obejct", `{"Type":"astral.blueprint.slice_spec","Obejct":{"Type":"uint32"}}`},
		{"Object misspelled as Objct", `{"Type":"astral.blueprint.slice_spec","Objct":{"Type":"uint32"}}`},
		{"Type misspelled as Typ", `{"Typ":"astral.blueprint.slice_spec","Object":{"Type":"uint32"}}`},
		{"excess key beside a valid pair", `{"Type":"astral.blueprint.slice_spec","Object":{"Type":"uint32"},"Extra":1}`},
		{"payload key hoisted into the envelope", `{"Type":"astral.blueprint.slice_spec","PrimitiveType":"uint32"}`},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			doc := `{"Name":"x","Spec":` + test.envelope + `}`

			var field Field
			err := json.Unmarshal([]byte(doc), &field)
			if err == nil {
				t.Fatalf("decoded a malformed envelope without error, got spec %#v", field.Spec)
			}
			if !strings.Contains(err.Error(), "excess fields in json envelope") {
				t.Fatalf("error: want an excess-fields rejection, got %v", err)
			}
		})
	}
}

// TestJSONEnvelope_RejectsCaseCollision pins an envelope carrying two keys that fold to the
// same name being rejected rather than resolved.
//
// encoding/json matched case-insensitively and let the last key win, so a document holding
// two conflicting payloads decoded to one of them with no error — and astral-py rejects the
// same document, so the two SDKs disagreed on the result. structValue already refuses the
// shape one level down; the envelope now matches it.
func TestJSONEnvelope_RejectsCaseCollision(t *testing.T) {
	tests := []struct {
		name     string
		envelope string
	}{
		{"two Object keys", `{"Type":"astral.blueprint.slice_spec","Object":{"Type":"uint32"},"OBJECT":{"Type":"string8"}}`},
		{"two Type keys", `{"Type":"astral.blueprint.slice_spec","TYPE":"astral.blueprint.ref_spec","Object":{"Type":"uint32"}}`},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			doc := `{"Name":"x","Spec":` + test.envelope + `}`

			var field Field
			err := json.Unmarshal([]byte(doc), &field)
			if err == nil {
				t.Fatalf("resolved a case-colliding envelope, got spec %#v", field.Spec)
			}
			if !strings.Contains(err.Error(), "duplicate fields due to case insensitivity") {
				t.Fatalf("error: want a case-collision rejection, got %v", err)
			}
		})
	}
}

// TestJSONEnvelope_AcceptsConforming pins the strictness costing nothing the spec allows.
// Key names fold case-insensitively (topics/json-encoding.md), key order is not
// significant, and an envelope with no payload key stays acceptable — the spec fixes the
// container's shape, not the decoder's tolerance for an absent optional payload.
func TestJSONEnvelope_AcceptsConforming(t *testing.T) {
	tests := []struct {
		name     string
		envelope string
		want     string
	}{
		{"canonical", `{"Type":"astral.blueprint.slice_spec","Object":{"Type":"uint32"}}`, "uint32"},
		{"lowercase keys", `{"type":"astral.blueprint.slice_spec","object":{"Type":"uint32"}}`, "uint32"},
		{"uppercase keys", `{"TYPE":"astral.blueprint.slice_spec","OBJECT":{"Type":"uint32"}}`, "uint32"},
		{"reversed key order", `{"Object":{"Type":"uint32"},"Type":"astral.blueprint.slice_spec"}`, "uint32"},
		{"no payload key", `{"Type":"astral.blueprint.slice_spec"}`, ""},
		{"null payload", `{"Type":"astral.blueprint.slice_spec","Object":null}`, ""},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			doc := `{"Name":"x","Spec":` + test.envelope + `}`

			var field Field
			if err := json.Unmarshal([]byte(doc), &field); err != nil {
				t.Fatalf("Unmarshal returned error: %v (from %s)", err, doc)
			}
			spec, ok := field.Spec.(*SliceSpec)
			if !ok {
				t.Fatalf("Spec: want *SliceSpec, got %#v", field.Spec)
			}
			if spec.Type.String() != test.want {
				t.Fatalf("SliceSpec.Type: want %q, got %q", test.want, spec.Type)
			}
		})
	}
}

// TestJSONEnvelope_EveryDecodeRouteIsStrict pins the check covering every envelope decode
// in the module, not only Field.Spec. All four routes funnel through JSONAdapter, which is
// why the check lives on that type rather than at each slot.
func TestJSONEnvelope_EveryDecodeRouteIsStrict(t *testing.T) {
	t.Run("interfaceValue via Field.Spec", func(t *testing.T) {
		var field Field
		err := json.Unmarshal([]byte(`{"Name":"x","Spec":{"Type":"astral.blueprint.slice_spec","Obejct":{"Type":"uint32"}}}`), &field)
		assertEnvelopeRejected(t, err, field.Spec)
	})

	t.Run("interfaceValue via ErrUnexpectedObject.Object", func(t *testing.T) {
		var e ErrUnexpectedObject
		err := json.Unmarshal([]byte(`{"Object":{"Type":"uint32","Obejct":42}}`), &e)
		assertEnvelopeRejected(t, err, e.Object)
	})

	t.Run("Bundle element", func(t *testing.T) {
		bundle := NewBundle()
		err := json.Unmarshal([]byte(`[{"Type":"uint32","Obejct":42}]`), bundle)
		assertEnvelopeRejected(t, err, bundle.Objects())
	})

	t.Run("RuntimeObject ObjectSpec field", func(t *testing.T) {
		bp := NewBlueprint("test.envelope_route", Field{Name: "Slot", Spec: &ObjectSpec{}})
		ro, err := NewRuntimeObject(bp)
		if err != nil {
			t.Fatalf("NewRuntimeObject returned error: %v", err)
		}
		err = json.Unmarshal([]byte(`{"Slot":{"Type":"uint32","Obejct":42}}`), ro)
		assertEnvelopeRejected(t, err, ro)
	})
}

func assertEnvelopeRejected(t *testing.T, err error, decoded any) {
	t.Helper()
	if err == nil {
		t.Fatalf("decoded a malformed envelope without error, got %#v", decoded)
	}
	if !strings.Contains(err.Error(), "excess fields in json envelope") {
		t.Fatalf("error: want an excess-fields rejection, got %v", err)
	}
}

// TestJSONEnvelope_EmissionUnchanged pins the wire bytes literally. The strictness is a
// decode-side change only: every carrier still emits both keys, and the empty payload emits
// "Object":{} rather than omitting the key — so nothing this module emits is refused by the
// stricter decoder, and neither is anything astral-py or astral-js emits.
func TestJSONEnvelope_EmissionUnchanged(t *testing.T) {
	tests := []struct {
		spec Spec
		want string
	}{
		{&PrimitiveSpec{PrimitiveType: "uint32"}, `{"Name":"f","Spec":{"Type":"astral.blueprint.primitive_spec","Object":{"PrimitiveType":"uint32"}}}`},
		{&SliceSpec{Type: "uint8"}, `{"Name":"f","Spec":{"Type":"astral.blueprint.slice_spec","Object":{"Type":"uint8"}}}`},
		{&ArraySpec{Type: "uint8", Length: 4}, `{"Name":"f","Spec":{"Type":"astral.blueprint.array_spec","Object":{"Length":4,"Type":"uint8"}}}`},
		{&ObjectSpec{}, `{"Name":"f","Spec":{"Type":"astral.blueprint.object_spec","Object":{}}}`},
	}

	for _, test := range tests {
		t.Run(test.spec.ObjectType(), func(t *testing.T) {
			encoded, err := json.Marshal(&Field{Name: "f", Spec: test.spec})
			if err != nil {
				t.Fatalf("Marshal returned error: %v", err)
			}
			if string(encoded) != test.want {
				t.Fatalf("wire form:\n want %s\n  got %s", test.want, encoded)
			}

			var decoded Field
			if err = json.Unmarshal(encoded, &decoded); err != nil {
				t.Fatalf("own emission refused by the stricter decoder: %v", err)
			}
		})
	}
}
