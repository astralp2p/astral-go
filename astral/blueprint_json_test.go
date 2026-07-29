package astral

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"
)

func TestBlueprint_JSONRoundTrip(t *testing.T) {
	cases := []*Blueprint{
		NewBlueprint("test.json.empty"),
		NewBlueprintAlias("test.json.alias", "uint8"),
		NewBlueprint("test.json.full",
			Field{Name: "prim", Spec: &PrimitiveSpec{PrimitiveType: "string16"}},
			Field{Name: "ref", Spec: &RefSpec{Type: "some.thing"}},
			Field{Name: "slice", Spec: &SliceSpec{Type: "uint32"}},
			Field{Name: "hetero", Spec: &SliceSpec{Type: ""}},
			Field{Name: "array", Spec: &ArraySpec{Type: "uint8", Length: 4}},
			Field{Name: "map", Spec: &MapSpec{KeyType: "string16", ValueType: "uint32"}},
			Field{Name: "ptr", Spec: &PtrSpec{Type: "some.thing"}},
			Field{Name: "object", Spec: &ObjectSpec{}},
		),
	}

	for _, src := range cases {
		t.Run(string(src.Type), func(t *testing.T) {
			b, err := json.Marshal(src)
			if err != nil {
				t.Fatal(err)
			}

			dst := &Blueprint{}
			if err := json.Unmarshal(b, dst); err != nil {
				t.Fatal(err)
			}
			// nil-vs-empty Fields is the pre-existing Objectify slice contract,
			// not part of this codec: normalize before comparing.
			if len(src.Fields) == 0 && len(dst.Fields) == 0 {
				dst.Fields = src.Fields
			}
			if !reflect.DeepEqual(src, dst) {
				t.Fatalf("round-trip mismatch:\nwant %#v\ngot  %#v", src, dst)
			}
		})
	}
}

// The Spec slot is interface-typed, so it marshals as a nested {"Type","Object"} container
// naming the carrier — the discriminator the decoder dispatches on (topics/blueprints.md#json).
func TestBlueprint_JSONSpecEnvelope(t *testing.T) {
	src := NewBlueprint("test.json.envelope",
		Field{Name: "prim", Spec: &PrimitiveSpec{PrimitiveType: "identity"}},
	)

	b, err := json.Marshal(src)
	if err != nil {
		t.Fatal(err)
	}
	want := `"Spec":{"Type":"astral.blueprint.primitive_spec","Object":{"PrimitiveType":"identity"}}`
	if !strings.Contains(string(b), want) {
		t.Fatalf("marshal lacks the Spec container:\nwant substring %s\ngot %s", want, b)
	}
}

// RefSpec and PtrSpec marshal identical payloads ({"Type":…}); only the container's type tag
// separates them. A decoded Spec must come back as the carrier that went in.
func TestBlueprint_JSONSpecDiscrimination(t *testing.T) {
	src := NewBlueprint("test.json.discrim",
		Field{Name: "ref", Spec: &RefSpec{Type: "some.thing"}},
		Field{Name: "ptr", Spec: &PtrSpec{Type: "some.thing"}},
	)

	b, err := json.Marshal(src)
	if err != nil {
		t.Fatal(err)
	}
	dst := &Blueprint{}
	if err := json.Unmarshal(b, dst); err != nil {
		t.Fatal(err)
	}
	if _, ok := dst.Fields[0].Spec.(*RefSpec); !ok {
		t.Fatalf("field 0: want *RefSpec, got %T", dst.Fields[0].Spec)
	}
	if _, ok := dst.Fields[1].Spec.(*PtrSpec); !ok {
		t.Fatalf("field 1: want *PtrSpec, got %T", dst.Fields[1].Spec)
	}
}

// The JSON channel receiver's exact path: New() the registered prototype, then unmarshal the
// payload into it. Before Blueprint had UnmarshalJSON this failed on the first field.
func TestBlueprint_JSONViaRegistry(t *testing.T) {
	payload := `{"Type":"test.json.registry","Fields":[{"Name":"Body","Spec":{"Type":"astral.blueprint.primitive_spec","Object":{"PrimitiveType":"string16"}}}],"Underlying":""}`

	o := New("astral.blueprint")
	if o == nil {
		t.Fatal("astral.blueprint not registered")
	}
	if err := json.Unmarshal([]byte(payload), o); err != nil {
		t.Fatal(err)
	}

	bp, ok := o.(*Blueprint)
	if !ok {
		t.Fatalf("want *Blueprint, got %T", o)
	}
	if bp.Type != "test.json.registry" || len(bp.Fields) != 1 || bp.Fields[0].Name != "Body" {
		t.Fatalf("unexpected blueprint: %+v", bp)
	}
	spec, ok := bp.Fields[0].Spec.(*PrimitiveSpec)
	if !ok || spec.PrimitiveType != "string16" {
		t.Fatalf("unexpected spec: %+v", bp.Fields[0].Spec)
	}
}

func TestField_JSONStandalone(t *testing.T) {
	src := &Field{Name: "n", Spec: &SliceSpec{Type: "uint64"}}

	b, err := json.Marshal(src)
	if err != nil {
		t.Fatal(err)
	}
	dst := &Field{}
	if err := json.Unmarshal(b, dst); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(src, dst) {
		t.Fatalf("round-trip mismatch:\nwant %+v\ngot  %+v", src, dst)
	}
}

// A Spec envelope naming a registered type that does not implement Spec must fail the
// decode, not panic reflect.Value.Convert — JSON channels make the slot network-reachable
// (objects.register_blueprint -in json is open to peers).
func TestBlueprint_JSONHostileSpecType(t *testing.T) {
	if _, err := Register(NewBlueprint("test.json.hostile.rt")); err != nil {
		t.Fatal(err)
	}

	payloads := map[string]string{
		"primitive": `{"Type":"t","Fields":[{"Name":"x","Spec":{"Type":"uint8","Object":5}}],"Underlying":""}`,
		"runtime":   `{"Type":"t","Fields":[{"Name":"x","Spec":{"Type":"test.json.hostile.rt","Object":{}}}],"Underlying":""}`,
	}
	for name, payload := range payloads {
		t.Run(name, func(t *testing.T) {
			dst := &Blueprint{}
			if err := json.Unmarshal([]byte(payload), dst); err == nil {
				t.Fatal("decoded a non-Spec type into the Spec slot")
			}
		})
	}
}

// A null Spec decodes to nil rather than erroring; validation rejects it at register time.
func TestBlueprint_JSONNullSpec(t *testing.T) {
	payload := `{"Type":"test.json.nullspec","Fields":[{"Name":"n","Spec":null}],"Underlying":""}`

	dst := &Blueprint{}
	if err := json.Unmarshal([]byte(payload), dst); err != nil {
		t.Fatal(err)
	}
	if dst.Fields[0].Spec != nil {
		t.Fatalf("want nil Spec, got %+v", dst.Fields[0].Spec)
	}
	if err := validateBlueprint(dst); err == nil {
		t.Fatal("validateBlueprint accepted a nil Spec")
	}
}
