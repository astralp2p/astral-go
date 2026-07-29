package query

import (
	"encoding/base64"
	"reflect"
	"testing"

	"github.com/astralp2p/astral-go/astral"
)

// FieldEditor converts one struct field to and from its text form, and reports
// the astral type the field presents to the ".spec" op.

// fieldOf builds an editor over the single field of a one-field struct.
func fieldOf(t *testing.T, structPtr any, name string) *FieldEditor {
	t.Helper()

	editor, err := Edit(structPtr).Field(name)
	if err != nil {
		t.Fatalf("Field(%q): %v", name, err)
	}
	return editor
}

// ObjectType maps a Go kind to the astral type an app author sees in a spec.
// Every integer width maps to its own astral type; the sized types are not
// collapsed.
//
// reflect.Float64 is absent from this table on purpose: field_editor.go:62
// maps it to astral.Float32, although astral.Float64 exists and reports
// "float64". That is a defect, filed as its own task — asserting the current
// value here would cement it.
func TestFieldEditor_ObjectType(t *testing.T) {
	type kinds struct {
		Str     string
		I8      int8
		I16     int16
		I32     int32
		I64     int64
		I       int
		U8      uint8
		U16     uint16
		U32     uint32
		U64     uint64
		U       uint
		F32     float32
		Boolean bool
		Bytes   []byte
	}

	cases := []struct {
		field string
		want  string
	}{
		{"str", astral.String8("").ObjectType()},
		{"i8", astral.Int8(0).ObjectType()},
		{"i16", astral.Int16(0).ObjectType()},
		{"i32", astral.Int32(0).ObjectType()},
		{"i64", astral.Int64(0).ObjectType()},
		{"i", astral.Int64(0).ObjectType()},
		{"u8", astral.Uint8(0).ObjectType()},
		{"u16", astral.Uint16(0).ObjectType()},
		{"u32", astral.Uint32(0).ObjectType()},
		{"u64", astral.Uint64(0).ObjectType()},
		{"u", astral.Uint64(0).ObjectType()},
		{"f32", astral.Float32(0).ObjectType()},
		{"boolean", astral.Bool(false).ObjectType()},
		{"bytes", astral.Bytes32{}.ObjectType()},
	}

	for _, c := range cases {
		t.Run(c.field, func(t *testing.T) {
			got := fieldOf(t, &kinds{}, c.field).ObjectType()
			if got != c.want {
				t.Fatalf("want %q, got %q", c.want, got)
			}
		})
	}
}

// A field that is itself an astral.Object reports its own type, not the type
// derived from its kind.
func TestFieldEditor_ObjectTypeOfAnAstralObject(t *testing.T) {
	type args struct {
		ID *astral.Identity
	}

	t.Run("populated", func(t *testing.T) {
		got := fieldOf(t, &args{ID: astral.GenerateIdentity()}, "id").ObjectType()
		if got != astral.GenerateIdentity().ObjectType() {
			t.Fatalf("want %q, got %q", astral.GenerateIdentity().ObjectType(), got)
		}
	})

	// a nil pointer still reports the type: the editor allocates a zero value
	// solely to ask it
	t.Run("nil pointer", func(t *testing.T) {
		got := fieldOf(t, &args{}, "id").ObjectType()
		if got != astral.GenerateIdentity().ObjectType() {
			t.Fatalf("want %q, got %q", astral.GenerateIdentity().ObjectType(), got)
		}
	})
}

// A kind outside the mapped set reports the empty string, which is how Spec
// decides to omit the field.
func TestFieldEditor_ObjectTypeOfAnUnmappedKind(t *testing.T) {
	type args struct {
		Nested  struct{ A int }
		Strings []string
	}

	for _, name := range []string{"nested", "strings"} {
		t.Run(name, func(t *testing.T) {
			if got := fieldOf(t, &args{}, name).ObjectType(); got != "" {
				t.Fatalf("want empty, got %q", got)
			}
		})
	}
}

func TestFieldEditor_TextRoundTrip(t *testing.T) {
	type kinds struct {
		Str     string
		I64     int64
		U64     uint64
		F64     float64
		Boolean bool
	}

	cases := []struct {
		field string
		text  string
	}{
		{"str", "hello"},
		{"str", ""},
		{"i64", "42"},
		{"i64", "-42"},
		{"u64", "42"},
		{"f64", "1.5"},
		{"boolean", "true"},
		{"boolean", "false"},
	}

	for _, c := range cases {
		t.Run(c.field+"="+c.text, func(t *testing.T) {
			editor := fieldOf(t, &kinds{}, c.field)

			if err := editor.Set(c.text); err != nil {
				t.Fatalf("Set(%q): unexpected error: %v", c.text, err)
			}
			if got := editor.Get(); got != c.text {
				t.Fatalf("want %q, got %q", c.text, got)
			}
		})
	}
}

// Bool accepts the forms strconv.ParseBool accepts, and normalises on the way
// back out.
func TestFieldEditor_BoolAcceptsShortForms(t *testing.T) {
	type args struct{ Enabled bool }

	cases := []struct {
		text string
		want string
	}{
		{"1", "true"},
		{"t", "true"},
		{"TRUE", "true"},
		{"0", "false"},
		{"f", "false"},
	}

	for _, c := range cases {
		t.Run(c.text, func(t *testing.T) {
			editor := fieldOf(t, &args{}, "enabled")

			if err := editor.Set(c.text); err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got := editor.Get(); got != c.want {
				t.Fatalf("want %q, got %q", c.want, got)
			}
		})
	}
}

// A []byte field crosses the query string as standard base64.
func TestFieldEditor_BytesUseBase64(t *testing.T) {
	type args struct{ Data []byte }

	var a args
	editor := fieldOf(t, &a, "data")

	payload := []byte{0x00, 0x01, 0xfe, 0xff}
	encoded := base64.StdEncoding.EncodeToString(payload)

	if err := editor.Set(encoded); err != nil {
		t.Fatalf("Set: unexpected error: %v", err)
	}
	if !reflect.DeepEqual(a.Data, payload) {
		t.Fatalf("want %v, got %v", payload, a.Data)
	}
	if got := editor.Get(); got != encoded {
		t.Fatalf("want %q, got %q", encoded, got)
	}
}

// A nil pointer field marshals to no text at all, which is how MarshalQuery
// omits it from the query string rather than writing an empty value.
func TestFieldEditor_NilPointerMarshalsToNothing(t *testing.T) {
	type args struct {
		Optional *string
	}

	text, err := fieldOf(t, &args{}, "optional").MarshalText()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if text != nil {
		t.Fatalf("want nil, got %q", string(text))
	}
}

// An omitted optional field leaves the query string clean.
func TestEditor_MarshalQueryOmitsNilPointers(t *testing.T) {
	type args struct {
		Name     string
		Optional *string
	}

	text, err := Edit(&args{Name: "bob"}).MarshalQuery()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	want := "name=bob"
	if string(text) != want {
		t.Fatalf("want %q, got %q", want, string(text))
	}
}

// Setting a nil pointer field allocates it.
func TestFieldEditor_SetAllocatesANilPointer(t *testing.T) {
	type args struct {
		Optional *string
	}

	var a args
	if err := fieldOf(t, &a, "optional").Set("value"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if a.Optional == nil {
		t.Fatal("want the pointer allocated, got nil")
	}
	if *a.Optional != "value" {
		t.Fatalf("want %q, got %q", "value", *a.Optional)
	}
}

// A field implementing TextMarshaler/TextUnmarshaler uses its own codec rather
// than the kind table. astral.Identity is the case app authors hit.
func TestFieldEditor_TextCodecTakesPrecedence(t *testing.T) {
	type args struct {
		ID *astral.Identity
	}

	id := astral.GenerateIdentity()
	text, err := id.MarshalText()
	if err != nil {
		t.Fatalf("MarshalText: %v", err)
	}

	var a args
	editor := fieldOf(t, &a, "id")

	if err := editor.Set(string(text)); err != nil {
		t.Fatalf("Set: unexpected error: %v", err)
	}
	if a.ID == nil {
		t.Fatal("want the identity set, got nil")
	}
	if !a.ID.IsEqual(id) {
		t.Fatalf("want %v, got %v", id, a.ID)
	}
	if got := editor.Get(); got != string(text) {
		t.Fatalf("want %q, got %q", string(text), got)
	}
}

// A malformed value is reported, not silently coerced to the zero value.
func TestFieldEditor_UnmarshalTextRejectsMalformedValues(t *testing.T) {
	type kinds struct {
		I64     int64
		U64     uint64
		F64     float64
		Boolean bool
		Data    []byte
	}

	cases := []struct {
		field string
		text  string
	}{
		{"i64", "not-a-number"},
		{"i64", "1.5"},
		{"u64", "-1"},
		{"f64", "not-a-number"},
		{"boolean", "yes"},
		{"data", "not!base64"},
	}

	for _, c := range cases {
		t.Run(c.field+"="+c.text, func(t *testing.T) {
			err := fieldOf(t, &kinds{}, c.field).Set(c.text)
			if err == nil {
				t.Fatalf("want an error for %q, got nil", c.text)
			}
		})
	}
}

// A field of an unmapped kind cannot cross the text boundary in either
// direction.
func TestFieldEditor_UnmappedKindFailsBothWays(t *testing.T) {
	type args struct {
		Nested struct{ A int }
	}

	editor := fieldOf(t, &args{}, "nested")

	if _, err := editor.MarshalText(); err == nil {
		t.Fatal("MarshalText: want an error, got nil")
	}
	if err := editor.UnmarshalText([]byte("x")); err == nil {
		t.Fatal("UnmarshalText: want an error, got nil")
	}
}

func TestFieldEditor_TagIsExposed(t *testing.T) {
	editor := fieldOf(t, &taggedArgs{}, "needed")

	if !editor.Tag().Required {
		t.Fatal("Tag().Required: want true, got false")
	}
}

func TestFieldEditor_StringMatchesGet(t *testing.T) {
	type args struct{ Name string }

	editor := fieldOf(t, &args{Name: "bob"}, "name")

	if editor.String() != editor.Get() {
		t.Fatalf("String()=%q, Get()=%q", editor.String(), editor.Get())
	}
	if editor.String() != "bob" {
		t.Fatalf("want %q, got %q", "bob", editor.String())
	}
}
