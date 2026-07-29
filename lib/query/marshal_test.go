package query

import (
	"errors"
	"testing"
)

// Marshal turns the accepted argument shapes into a query string. A string
// passes through untouched; everything else is encoded through url.Values,
// which sorts keys, so the output is deterministic.

type stringerArg struct{}

func (stringerArg) String() string { return "from-stringer" }

type textArg struct{}

func (textArg) MarshalText() ([]byte, error) { return []byte("from-text"), nil }

// bothArg implements TextMarshaler and Stringer; TextMarshaler is checked first.
type bothArg struct{}

func (bothArg) MarshalText() ([]byte, error) { return []byte("from-text"), nil }
func (bothArg) String() string               { return "from-stringer" }

var errMarshalText = errors.New("marshal text failed")

type failingTextArg struct{}

func (failingTextArg) MarshalText() ([]byte, error) { return nil, errMarshalText }

type marshalStruct struct {
	UserName string
	Count    int
}

func TestMarshal(t *testing.T) {
	cases := []struct {
		name   string
		params any
		want   string
	}{
		{"nil", nil, ""},
		{"string passes through unencoded", "a=1&b=2", "a=1&b=2"},
		{"string is not validated", "not a query string", "not a query string"},
		{"empty string", "", ""},
		{"map of strings", map[string]string{"b": "2", "a": "1"}, "a=1&b=2"},
		{"empty map of strings", map[string]string{}, ""},
		{"map of any, string value", map[string]any{"a": "1"}, "a=1"},
		{"map of any, Stringer value", map[string]any{"a": stringerArg{}}, "a=from-stringer"},
		{"map of any, TextMarshaler value", map[string]any{"a": textArg{}}, "a=from-text"},
		{"map of any, TextMarshaler beats Stringer", map[string]any{"a": bothArg{}}, "a=from-text"},
		{"map of any, int falls back to %v", map[string]any{"a": 7}, "a=7"},
		{"map of any, bool falls back to %v", map[string]any{"a": true}, "a=true"},
		{"Args alias", Args{"a": "1"}, "a=1"},
		{"values are escaped", map[string]string{"msg": "hello world"}, "msg=hello+world"},
		{"struct", marshalStruct{UserName: "bob", Count: 3}, "count=3&user_name=bob"},
		{"pointer to struct", &marshalStruct{UserName: "bob", Count: 3}, "count=3&user_name=bob"},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, err := Marshal(c.params)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != c.want {
				t.Fatalf("want %q, got %q", c.want, got)
			}
		})
	}
}

// A non-addressable struct is copied into a fresh addressable value rather than
// rejected, so passing a struct by value works.
func TestMarshal_NonAddressableStruct(t *testing.T) {
	got, err := Marshal(marshalStruct{UserName: "bob", Count: 3})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	want := "count=3&user_name=bob"
	if got != want {
		t.Fatalf("want %q, got %q", want, got)
	}
}

// A TextMarshaler that fails aborts the whole marshal rather than skipping the
// field.
func TestMarshal_TextMarshalerErrorPropagates(t *testing.T) {
	_, err := Marshal(map[string]any{"a": failingTextArg{}})

	if !errors.Is(err, errMarshalText) {
		t.Fatalf("want %v, got %v", errMarshalText, err)
	}
}

// Types outside the accepted set are rejected by kind.
func TestMarshal_UnsupportedType(t *testing.T) {
	cases := []struct {
		name   string
		params any
	}{
		{"int", 42},
		{"slice", []string{"a"}},
		{"map with non-string keys", map[int]string{1: "a"}},
		{"pointer to a non-struct", new(int)},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			_, err := Marshal(c.params)
			if err == nil {
				t.Fatalf("want an error for %T, got nil", c.params)
			}
		})
	}
}
