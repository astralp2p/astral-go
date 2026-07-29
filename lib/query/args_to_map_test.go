package query

import (
	"reflect"
	"testing"
)

// ArgsToMap is the flag parser on the App.Run path. A "-name value" pair
// becomes a named key; anything else becomes the positional argument under
// DefaultArgKey.

func TestArgsToMap(t *testing.T) {
	cases := []struct {
		name string
		args []string
		want map[string]string
	}{
		{
			name: "empty",
			args: []string{},
			want: map[string]string{},
		},
		{
			name: "one flag pair",
			args: []string{"-id", "abc"},
			want: map[string]string{"id": "abc"},
		},
		{
			name: "several flag pairs",
			args: []string{"-id", "abc", "-message", "hello"},
			want: map[string]string{"id": "abc", "message": "hello"},
		},
		{
			name: "positional lands under DefaultArgKey",
			args: []string{"value"},
			want: map[string]string{DefaultArgKey: "value"},
		},
		{
			name: "positional mixed with a flag pair",
			args: []string{"value", "-id", "abc"},
			want: map[string]string{DefaultArgKey: "value", "id": "abc"},
		},
		{
			name: "trailing flag without a value maps to empty",
			args: []string{"-id"},
			want: map[string]string{"id": ""},
		},
		{
			name: "a flag value may itself look like a value",
			args: []string{"-id", "-1"},
			want: map[string]string{"id": "-1"},
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := ArgsToMap(c.args)
			if !reflect.DeepEqual(got, c.want) {
				t.Fatalf("want %v, got %v", c.want, got)
			}
		})
	}
}

// Only one positional argument survives: each overwrites the last, because they
// share the single DefaultArgKey slot.
func TestArgsToMap_RepeatedPositionalsCollapse(t *testing.T) {
	got := ArgsToMap([]string{"first", "second", "third"})

	want := map[string]string{DefaultArgKey: "third"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("want %v, got %v", want, got)
	}
}

// A trailing flag with no value stops the walk, so anything the caller expected
// after it is dropped. Pinned because the early return is easy to miss.
func TestArgsToMap_TrailingValuelessFlagStopsTheWalk(t *testing.T) {
	got := ArgsToMap([]string{"-a", "1", "-b"})

	want := map[string]string{"a": "1", "b": ""}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("want %v, got %v", want, got)
	}
}
