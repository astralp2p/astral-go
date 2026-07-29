package routing

import (
	"strings"
	"testing"

	"github.com/astralp2p/astral-go/lib/query"
)

// OpSpecView renders an OpSpec for a human reader. The output carries theme
// escape sequences, so these assertions cover the content and the structure,
// never the exact bytes.

func TestOpSpecView_RendersNameAndParentheses(t *testing.T) {
	view := OpSpecView{OpSpec: &OpSpec{Name: "objects.read"}}

	out := view.Render()

	if !strings.Contains(out, "objects.read") {
		t.Fatalf("want the op name in %q", out)
	}
	if !strings.Contains(out, "(") || !strings.Contains(out, ")") {
		t.Fatalf("want the parameter list delimiters in %q", out)
	}
}

func TestOpSpecView_RendersEveryParameter(t *testing.T) {
	view := OpSpecView{OpSpec: &OpSpec{
		Name: "objects.read",
		Parameters: []query.FieldSpec{
			{Name: "id", Type: "object_id", Required: true},
			{Name: "offset", Type: "uint64"},
		},
	}}

	out := view.Render()

	for _, want := range []string{"id", "object_id", "offset", "uint64"} {
		if !strings.Contains(out, want) {
			t.Fatalf("want %q in %q", want, out)
		}
	}

	// two parameters are separated
	if !strings.Contains(out, ", ") {
		t.Fatalf("want a separator between parameters in %q", out)
	}
}

// A required parameter carries the "*" marker; an optional one does not
// introduce a second marker.
func TestOpSpecView_MarksRequiredParameters(t *testing.T) {
	required := OpSpecView{OpSpec: &OpSpec{
		Name:       "op",
		Parameters: []query.FieldSpec{{Name: "id", Type: "object_id", Required: true}},
	}}.Render()

	optional := OpSpecView{OpSpec: &OpSpec{
		Name:       "op",
		Parameters: []query.FieldSpec{{Name: "id", Type: "object_id"}},
	}}.Render()

	if !strings.Contains(required, "*") {
		t.Fatalf("want the required marker in %q", required)
	}
	if strings.Contains(optional, "*") {
		t.Fatalf("want no required marker in %q", optional)
	}
}

// An op with no parameters renders an empty list rather than a stray separator.
func TestOpSpecView_RendersAnEmptyParameterList(t *testing.T) {
	out := OpSpecView{OpSpec: &OpSpec{Name: "ping"}}.Render()

	if strings.Contains(out, ",") {
		t.Fatalf("want no separator for a parameterless op, got %q", out)
	}
}
