package pub_test

import (
	"encoding/json"
	"reflect"
	"sort"
	"testing"

	_ "github.com/astralp2p/astral-go"
	"github.com/astralp2p/astral-go/astral"
)

// An empty list marshals to JSON [] and never to null: astral-rs rejects null
// where it expects a list, and astral's slice codec writes [] on purpose
// (astral/slice_value.go). A type left on encoding/json's default writes a nil
// slice as null, so every registered type is marshalled and its output walked.
//
// Each type is filled to slice depth k and left nil below it, for k up to
// maxListDepth, so a nil list inside a list element is reached too.
//
// Byte strings are not lists, and a nil map is null in astral's own map codec
// (astral/map_value.go), so neither is checked.
//
// This lives at the repo root because it needs the blank import above to
// register the whole api surface.

const maxListDepth = 3

func TestEmptyListsMarshalAsArrays(t *testing.T) {
	names := astral.DefaultBlueprints().OrderedBlueprints()
	sort.Strings(names)

	for k := 0; k <= maxListDepth; k++ {
		for _, name := range names {
			o := astral.New(name)
			if o == nil {
				continue
			}
			fillLists(reflect.ValueOf(o), 0, k)

			b, err := json.Marshal(o)
			if err != nil {
				continue
			}
			var doc any
			if err := json.Unmarshal(b, &doc); err != nil {
				continue
			}
			for _, path := range nullLists(name, doc, reflect.TypeOf(o)) {
				t.Errorf("%s marshals an empty list as null: %s", path, b)
			}
		}
	}
}

// fillLists allocates pointers and gives each list one element, down to list
// depth maxDepth; lists below it stay nil.
func fillLists(v reflect.Value, depth, maxDepth int) {
	switch v.Kind() {
	case reflect.Pointer:
		if v.IsNil() && v.CanSet() {
			v.Set(reflect.New(v.Type().Elem()))
		}
		if !v.IsNil() {
			fillLists(v.Elem(), depth, maxDepth)
		}
	case reflect.Struct:
		for i := 0; i < v.NumField(); i++ {
			if v.Type().Field(i).IsExported() {
				fillLists(v.Field(i), depth, maxDepth)
			}
		}
	case reflect.Slice:
		if isList(v.Type()) && v.CanSet() && depth < maxDepth {
			s := reflect.MakeSlice(v.Type(), 1, 1)
			fillLists(s.Index(0), depth+1, maxDepth)
			v.Set(s)
		}
	}
}

// nullLists answers the paths at which doc holds null for a list of t.
func nullLists(path string, doc any, t reflect.Type) (out []string) {
	for t.Kind() == reflect.Pointer {
		t = t.Elem()
	}
	switch d := doc.(type) {
	case nil:
		if isList(t) {
			out = append(out, path)
		}
	case []any:
		if t.Kind() == reflect.Slice {
			for _, e := range d {
				out = append(out, nullLists(path+"[]", e, t.Elem())...)
			}
		}
	case map[string]any:
		if t.Kind() != reflect.Struct {
			return
		}
		for key, v := range d {
			if f, ok := t.FieldByName(key); ok {
				out = append(out, nullLists(path+"."+key, v, f.Type)...)
			}
		}
	}
	return
}

func isList(t reflect.Type) bool {
	return t.Kind() == reflect.Slice && t.Elem().Kind() != reflect.Uint8
}
