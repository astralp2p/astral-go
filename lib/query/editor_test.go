package query

import (
	"errors"
	"reflect"
	"slices"
	"strings"
	"testing"
)

// Editor reflects over a struct and exposes its fields by query-argument name.
// Field naming is the contract app authors depend on: Edit snake_cases, and a
// key tag overrides.

type editorArgs struct {
	UserName string
	Count    int
	Enabled  bool
}

type taggedArgs struct {
	Renamed  string `query:"key:custom"`
	Hidden   string `query:"skip"`
	Needed   string `query:"required"`
	Ordinary string
	hidden   string //nolint:unused // unexported fields are excluded by the editor
}

func TestEdit_SnakeCasesFieldNames(t *testing.T) {
	editor := Edit(&editorArgs{})

	for _, name := range []string{"user_name", "count", "enabled"} {
		if _, err := editor.Field(name); err != nil {
			t.Fatalf("Field(%q): want found, got %v", name, err)
		}
	}

	if _, err := editor.Field("UserName"); !errors.Is(err, ErrFieldNotFound) {
		t.Fatalf("Field(%q): want ErrFieldNotFound, got %v", "UserName", err)
	}
}

func TestEditCamel_PreservesFieldNameCasing(t *testing.T) {
	editor := EditCamel(&editorArgs{})

	if _, err := editor.Field("UserName"); err != nil {
		t.Fatalf("Field(%q): want found, got %v", "UserName", err)
	}
	if _, err := editor.Field("user_name"); !errors.Is(err, ErrFieldNotFound) {
		t.Fatalf("Field(%q): want ErrFieldNotFound, got %v", "user_name", err)
	}
}

// A key tag overrides the derived name in both naming modes.
func TestEdit_KeyTagOverridesTheDerivedName(t *testing.T) {
	for _, mode := range []struct {
		name string
		edit func(any) *Editor
	}{
		{"Edit", Edit},
		{"EditCamel", EditCamel},
	} {
		t.Run(mode.name, func(t *testing.T) {
			editor := mode.edit(&taggedArgs{})

			if _, err := editor.Field("custom"); err != nil {
				t.Fatalf("Field(%q): want found, got %v", "custom", err)
			}
			if _, err := editor.Field("renamed"); !errors.Is(err, ErrFieldNotFound) {
				t.Fatalf("Field(%q): want ErrFieldNotFound, got %v", "renamed", err)
			}
		})
	}
}

func TestEdit_SkipTagExcludesTheField(t *testing.T) {
	editor := Edit(&taggedArgs{})

	if _, err := editor.Field("hidden"); !errors.Is(err, ErrFieldNotFound) {
		t.Fatalf("Field(%q): want ErrFieldNotFound, got %v", "hidden", err)
	}
}

func TestEdit_UnexportedFieldsAreExcluded(t *testing.T) {
	editor := Edit(&taggedArgs{})

	// the struct declares an unexported "hidden" field alongside the
	// skip-tagged "Hidden"; neither is addressable by name
	for _, field := range editor.fields {
		if field.name == "hidden" {
			t.Fatal("an unexported field must not be exposed")
		}
	}
}

func TestEdit_PanicsOnInvalidArgument(t *testing.T) {
	cases := []struct {
		name string
		arg  any
	}{
		{"non-pointer struct", editorArgs{}},
		{"nil typed pointer", (*editorArgs)(nil)},
		{"pointer to a non-struct", new(int)},
		{"plain value", 42},
		{"nil", nil},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			defer func() {
				if recover() == nil {
					t.Fatalf("want a panic for %T, got none", c.arg)
				}
			}()
			Edit(c.arg)
		})
	}
}

// A pointer to a nil struct pointer is allocated in place, so the caller's
// pointer is populated rather than left nil.
func TestEdit_AllocatesThroughAPointerToPointer(t *testing.T) {
	var args *editorArgs

	editor := Edit(&args)

	if args == nil {
		t.Fatal("want the nil pointer allocated, got nil")
	}

	if err := editor.Set("user_name", "bob"); err != nil {
		t.Fatalf("Set: unexpected error: %v", err)
	}
	if args.UserName != "bob" {
		t.Fatalf("UserName: want %q, got %q", "bob", args.UserName)
	}
}

func TestEditor_SetAndGetRoundTrip(t *testing.T) {
	var args editorArgs
	editor := Edit(&args)

	cases := []struct {
		field string
		value string
	}{
		{"user_name", "bob"},
		{"count", "7"},
		{"enabled", "true"},
	}

	for _, c := range cases {
		t.Run(c.field, func(t *testing.T) {
			if err := editor.Set(c.field, c.value); err != nil {
				t.Fatalf("Set(%q): unexpected error: %v", c.field, err)
			}

			got, err := editor.Get(c.field)
			if err != nil {
				t.Fatalf("Get(%q): unexpected error: %v", c.field, err)
			}
			if got != c.value {
				t.Fatalf("Get(%q): want %q, got %q", c.field, c.value, got)
			}
		})
	}

	if args.UserName != "bob" || args.Count != 7 || !args.Enabled {
		t.Fatalf("want the struct updated, got %+v", args)
	}
}

func TestEditor_SetAndGetReportUnknownFields(t *testing.T) {
	editor := Edit(&editorArgs{})

	if err := editor.Set("nope", "x"); !errors.Is(err, ErrFieldNotFound) {
		t.Fatalf("Set: want ErrFieldNotFound, got %v", err)
	}
	if _, err := editor.Get("nope"); !errors.Is(err, ErrFieldNotFound) {
		t.Fatalf("Get: want ErrFieldNotFound, got %v", err)
	}
}

// Set wraps the cause, so the field name reaches the caller alongside the
// sentinel.
func TestEditor_SetNamesTheFieldInTheError(t *testing.T) {
	editor := Edit(&editorArgs{})

	err := editor.Set("nope", "x")
	if err == nil {
		t.Fatal("want an error, got nil")
	}
	if !strings.Contains(err.Error(), "nope") {
		t.Fatalf("want the field name in %q", err.Error())
	}
}

// SetMany is the query-parameter entry point: an unknown key is skipped so a
// stray argument cannot fail an op, but a bad value is reported.
func TestEditor_SetManySkipsUnknownKeys(t *testing.T) {
	var args editorArgs
	editor := Edit(&args)

	err := editor.SetMany(map[string]string{
		"user_name": "bob",
		"unknown":   "ignored",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if args.UserName != "bob" {
		t.Fatalf("UserName: want %q, got %q", "bob", args.UserName)
	}
}

func TestEditor_SetManyReportsConversionErrors(t *testing.T) {
	editor := Edit(&editorArgs{})

	err := editor.SetMany(map[string]string{"count": "not-a-number"})
	if err == nil {
		t.Fatal("want an error, got nil")
	}
	if errors.Is(err, ErrFieldNotFound) {
		t.Fatalf("want a conversion error, got ErrFieldNotFound: %v", err)
	}
}

func TestEditor_MarshalQueryRoundTrip(t *testing.T) {
	source := editorArgs{UserName: "bob", Count: 7, Enabled: true}

	text, err := Edit(&source).MarshalQuery()
	if err != nil {
		t.Fatalf("MarshalQuery: unexpected error: %v", err)
	}

	var target editorArgs
	if err := Edit(&target).UnmarshalQuery(text); err != nil {
		t.Fatalf("UnmarshalQuery: unexpected error: %v", err)
	}

	if target != source {
		t.Fatalf("want %+v, got %+v", source, target)
	}
}

func TestEditor_MarshalQuerySortsKeys(t *testing.T) {
	text, err := Edit(&editorArgs{UserName: "bob", Count: 7}).MarshalQuery()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	want := "count=7&enabled=false&user_name=bob"
	if string(text) != want {
		t.Fatalf("want %q, got %q", want, string(text))
	}
}

// UnmarshalQuery rejects an unknown key, unlike SetMany which skips it.
func TestEditor_UnmarshalQueryRejectsUnknownKeys(t *testing.T) {
	err := Edit(&editorArgs{}).UnmarshalQuery([]byte("unknown=1"))

	if !errors.Is(err, ErrFieldNotFound) {
		t.Fatalf("want ErrFieldNotFound, got %v", err)
	}
}

func TestEditor_UnmarshalQueryRejectsMalformedInput(t *testing.T) {
	err := Edit(&editorArgs{}).UnmarshalQuery([]byte("a=%zz"))

	if err == nil {
		t.Fatal("want an error, got nil")
	}
}

func TestEditor_StringIsTheMarshalledQuery(t *testing.T) {
	got := Edit(&editorArgs{UserName: "bob", Count: 7}).String()

	want := "count=7&enabled=false&user_name=bob"
	if got != want {
		t.Fatalf("want %q, got %q", want, got)
	}
}

// Spec drives the ".spec" op. It reports the name, the astral type, and whether
// the argument is required.
func TestEditor_Spec(t *testing.T) {
	specs := Edit(&taggedArgs{}).Spec()

	byName := map[string]FieldSpec{}
	for _, spec := range specs {
		byName[spec.Name] = spec
	}

	if _, found := byName["hidden"]; found {
		t.Fatal("a skip-tagged field must not appear in the spec")
	}

	needed, found := byName["needed"]
	if !found {
		t.Fatalf("want %q in the spec, got %v", "needed", byName)
	}
	if !needed.Required {
		t.Fatal("needed.Required: want true, got false")
	}

	ordinary, found := byName["ordinary"]
	if !found {
		t.Fatalf("want %q in the spec, got %v", "ordinary", byName)
	}
	if ordinary.Required {
		t.Fatal("ordinary.Required: want false, got true")
	}
}

// A field whose kind maps to no astral type is omitted from the spec entirely,
// even though it remains settable through the editor.
func TestEditor_SpecOmitsFieldsWithoutAnObjectType(t *testing.T) {
	type unmappedArgs struct {
		Name    string
		Unknown struct{ A int }
	}

	specs := Edit(&unmappedArgs{}).Spec()

	if len(specs) != 1 {
		t.Fatalf("want 1 spec, got %d (%v)", len(specs), specs)
	}
	if specs[0].Name != "name" {
		t.Fatalf("want %q, got %q", "name", specs[0].Name)
	}
}

func TestEditor_JSONRoundTrip(t *testing.T) {
	source := editorArgs{UserName: "bob", Count: 7, Enabled: true}

	data, err := Edit(&source).MarshalJSON()
	if err != nil {
		t.Fatalf("MarshalJSON: unexpected error: %v", err)
	}

	var target editorArgs
	if err := Edit(&target).UnmarshalJSON(data); err != nil {
		t.Fatalf("UnmarshalJSON: unexpected error: %v", err)
	}

	if target != source {
		t.Fatalf("want %+v, got %+v", source, target)
	}
}

// EditValue is the entry point for a caller that already holds a reflected
// pointer; it must agree with Edit.
func TestEditValue_MatchesEdit(t *testing.T) {
	args := editorArgs{UserName: "bob"}

	got, err := EditValue(reflect.ValueOf(&args)).Get("user_name")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "bob" {
		t.Fatalf("want %q, got %q", "bob", got)
	}
}

// TestEditor_SetArgs pins the flag-parsing contract, including termination.
//
// The non-flag branch previously appended args[i] and re-entered the loop without
// advancing i, so the same element was appended forever and the process spun until
// the allocator gave out. Every case here reaches its assertion only if SetArgs
// terminates; a regression hangs the package until the test timeout kills it.
func TestEditor_SetArgs(t *testing.T) {
	type target struct {
		UserName string
		Count    uint32
	}

	tests := []struct {
		name     string
		args     []string
		wantName string
		wantUnpa []string
	}{
		{
			name:     "flag pair",
			args:     []string{"-user_name", "bob"},
			wantName: "bob",
		},
		{
			name:     "positional only",
			args:     []string{"positional"},
			wantUnpa: []string{"positional"},
		},
		{
			name:     "positional before a flag pair",
			args:     []string{"positional", "-user_name", "bob"},
			wantName: "bob",
			wantUnpa: []string{"positional"},
		},
		{
			name:     "positional after a flag pair",
			args:     []string{"-user_name", "bob", "positional"},
			wantName: "bob",
			wantUnpa: []string{"positional"},
		},
		{
			name:     "several positionals keep their order",
			args:     []string{"one", "two", "three"},
			wantUnpa: []string{"one", "two", "three"},
		},
		{
			name:     "trailing flag with no value is unparsed, prefix intact",
			args:     []string{"-user_name"},
			wantUnpa: []string{"-user_name"},
		},
		{
			name: "empty slice",
			args: []string{},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var got target

			unparsed, err := Edit(&got).SetArgs(test.args)
			if err != nil {
				t.Fatalf("SetArgs returned error: %v", err)
			}
			if got.UserName != test.wantName {
				t.Fatalf("UserName: want %q, got %q", test.wantName, got.UserName)
			}
			if !slices.Equal(unparsed, test.wantUnpa) {
				t.Fatalf("unparsed: want %v, got %v", test.wantUnpa, unparsed)
			}
		})
	}
}

// TestEditor_SetArgs_UnknownFlag pins an unrecognised flag being reported rather than
// collected: it is a caller mistake, not a positional argument.
func TestEditor_SetArgs_UnknownFlag(t *testing.T) {
	var got struct{ UserName string }

	if _, err := Edit(&got).SetArgs([]string{"-nope", "value"}); err == nil {
		t.Fatal("SetArgs accepted an unknown flag, want an error")
	}
}

// TestEditor_SetArgs_ConversionError pins a value that does not fit its field being
// reported, not skipped.
func TestEditor_SetArgs_ConversionError(t *testing.T) {
	var got struct{ Count uint32 }

	if _, err := Edit(&got).SetArgs([]string{"-count", "not-a-number"}); err == nil {
		t.Fatal("SetArgs accepted a non-numeric value for a uint32 field, want an error")
	}
}
