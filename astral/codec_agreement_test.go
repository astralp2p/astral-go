package astral

import (
	"bytes"
	"strings"
	"testing"
)

// The two codecs must agree byte for byte. Where they do not, the same logical object
// has two encodings and two content addresses, and no SDK can be written against
// either one.

type nilHolder struct {
	V Object
}

func (nilHolder) ObjectType() string { return "test.nil_holder" }

// runtimeNilHolder builds the same shape as a runtime Blueprint, whose spec-zero for a
// polymorphic field is &Nil{}.
func runtimeNilHolder(t *testing.T) *RuntimeObject {
	t.Helper()

	// why: the same type name as the Go struct, so ResolveObjectID hashes the same
	// Stamp and type header for both and any difference is the payload alone.
	bp := NewBlueprint("test.nil_holder", Field{Name: "V", Spec: &ObjectSpec{}})

	ro, err := NewRuntimeObject(bp)
	if err != nil {
		t.Fatalf("NewRuntimeObject: %v", err)
	}
	return ro
}

func encode(t *testing.T, o Object) []byte {
	t.Helper()

	var buf bytes.Buffer
	if _, err := o.WriteTo(&buf); err != nil {
		t.Fatalf("WriteTo: %v", err)
	}
	return buf.Bytes()
}

// An absent polymorphic slot is the zero-length type tag, whichever codec wrote it and
// whichever of the three equivalent Go forms the slot holds.
func TestAbsentPolymorphicSlot_HasOneBinarySpelling(t *testing.T) {
	want := []byte{0x00}

	cases := map[string][]byte{
		"reflection, interface nil": encode(t, Objectify(&nilHolder{})),
		"reflection, &Nil{}":        encode(t, Objectify(&nilHolder{V: &Nil{}})),
		"runtime, spec zero":        encode(t, runtimeNilHolder(t)),
	}

	for name, got := range cases {
		if !bytes.Equal(got, want) {
			t.Errorf("%s: want % x, got % x", name, want, got)
		}
	}
}

func TestAbsentPolymorphicSlot_HasOneJSONSpelling(t *testing.T) {
	cases := map[string]Object{
		"reflection, interface nil": Objectify(&nilHolder{}),
		"reflection, &Nil{}":        Objectify(&nilHolder{V: &Nil{}}),
		"runtime, spec zero":        runtimeNilHolder(t),
	}

	for name, o := range cases {
		got, err := o.(interface{ MarshalJSON() ([]byte, error) }).MarshalJSON()
		if err != nil {
			t.Fatalf("%s: MarshalJSON: %v", name, err)
		}
		if !strings.Contains(string(got), "null") || strings.Contains(string(got), `"nil"`) {
			t.Errorf("%s: want an absent slot as null, got %s", name, got)
		}
	}
}

// Each codec must read what the other writes.
func TestAbsentPolymorphicSlot_CrossDecodes(t *testing.T) {
	reflectionBytes := encode(t, Objectify(&nilHolder{}))
	runtimeBytes := encode(t, runtimeNilHolder(t))

	t.Run("reflection output into the runtime reader", func(t *testing.T) {
		ro := runtimeNilHolder(t)
		if _, err := ro.ReadFrom(bytes.NewReader(reflectionBytes)); err != nil {
			t.Fatalf("ReadFrom: %v", err)
		}
	})

	t.Run("runtime output into the reflection reader", func(t *testing.T) {
		var holder nilHolder
		if _, err := Objectify(&holder).ReadFrom(bytes.NewReader(runtimeBytes)); err != nil {
			t.Fatalf("ReadFrom: %v", err)
		}
		if holder.V != nil {
			t.Errorf("want an absent slot to decode to nil, got %T", holder.V)
		}
	})
}

// Peers written before the spelling was settled send the Nil type's own name.
func TestAbsentPolymorphicSlot_AcceptsTheLegacySpelling(t *testing.T) {
	var legacy bytes.Buffer
	if _, err := String8(nilTypeName).WriteTo(&legacy); err != nil {
		t.Fatalf("encode legacy tag: %v", err)
	}

	var holder nilHolder
	if _, err := Objectify(&holder).ReadFrom(bytes.NewReader(legacy.Bytes())); err != nil {
		t.Fatalf("reflection reader: %v", err)
	}

	ro := runtimeNilHolder(t)
	if _, err := ro.ReadFrom(bytes.NewReader(legacy.Bytes())); err != nil {
		t.Fatalf("runtime reader: %v", err)
	}
}

// The test that would have caught this: two encodings mean two content addresses.
func TestAbsentPolymorphicSlot_ResolvesToOneObjectID(t *testing.T) {
	viaReflection, err := ResolveObjectID(Objectify(&nilHolder{}))
	if err != nil {
		t.Fatalf("ResolveObjectID: %v", err)
	}

	viaRuntime, err := ResolveObjectID(runtimeNilHolder(t))
	if err != nil {
		t.Fatalf("ResolveObjectID: %v", err)
	}

	if !viaReflection.IsEqual(viaRuntime) {
		t.Errorf("the same absent slot resolves to two content addresses:\n  reflection %s\n  runtime    %s",
			viaReflection, viaRuntime)
	}
}
