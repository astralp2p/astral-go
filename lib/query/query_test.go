package query

import (
	"reflect"
	"testing"

	"github.com/astralp2p/astral-go/astral"
)

// Parse splits a query string into the op path and its parameters. It is the
// server-side counterpart of New.

func TestParse(t *testing.T) {
	cases := []struct {
		name       string
		query      string
		wantPath   string
		wantParams map[string]string
	}{
		{
			name:       "path only",
			query:      "objects.read",
			wantPath:   "objects.read",
			wantParams: map[string]string{},
		},
		{
			name:       "path with one parameter",
			query:      "objects.read?id=abc",
			wantPath:   "objects.read",
			wantParams: map[string]string{"id": "abc"},
		},
		{
			name:       "path with several parameters",
			query:      "op?a=1&b=2",
			wantPath:   "op",
			wantParams: map[string]string{"a": "1", "b": "2"},
		},
		{
			name:       "empty query",
			query:      "",
			wantPath:   "",
			wantParams: map[string]string{},
		},
		{
			name:       "path with an empty parameter section",
			query:      "op?",
			wantPath:   "op",
			wantParams: map[string]string{},
		},
		{
			name:       "percent-encoded value is decoded",
			query:      "op?msg=hello+world",
			wantPath:   "op",
			wantParams: map[string]string{"msg": "hello world"},
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			path, params := Parse(c.query)

			if path != c.wantPath {
				t.Fatalf("path: want %q, got %q", c.wantPath, path)
			}
			if !reflect.DeepEqual(params, c.wantParams) {
				t.Fatalf("params: want %v, got %v", c.wantParams, params)
			}
		})
	}
}

// A repeated key keeps the first value; url.ParseQuery preserves both, and
// Parse takes index 0.
func TestParse_RepeatedKeyKeepsTheFirstValue(t *testing.T) {
	_, params := Parse("op?a=1&a=2")

	if params["a"] != "1" {
		t.Fatalf("params[a]: want %q, got %q", "1", params["a"])
	}
}

// A bare key becomes a named parameter with an empty value — it does NOT
// become the positional argument. The DefaultArgKey branch at query.go:63 is
// unreachable: url.ParseQuery("verbose") yields {"verbose": [""]}, so len(v)
// is 1 and the else branch never runs.
//
// This diverges from ArgsToMap, which does route a bare token to DefaultArgKey.
// The two argument paths disagree on positional arguments; the divergence is
// recorded on the task, and this test pins the current behaviour so that
// closing the gap is a deliberate change.
func TestParse_BareKeyIsNamedNotPositional(t *testing.T) {
	_, params := Parse("op?verbose")

	value, found := params["verbose"]
	if !found {
		t.Fatalf("params: want the bare key present as a name, got %v", params)
	}
	if value != "" {
		t.Fatalf("params[verbose]: want empty, got %q", value)
	}

	if _, found := params[DefaultArgKey]; found {
		t.Fatalf("params: want no %q entry, got %v", DefaultArgKey, params)
	}
}

// A malformed escape makes url.ParseQuery fail. Parse swallows the error and
// returns the path with empty parameters, so a caller cannot tell a malformed
// query from an argument-free one.
func TestParse_MalformedEscapeYieldsEmptyParams(t *testing.T) {
	path, params := Parse("op?a=%zz")

	if path != "op" {
		t.Fatalf("path: want %q, got %q", "op", path)
	}
	if len(params) != 0 {
		t.Fatalf("params: want empty, got %v", params)
	}
}

func TestSplitPathParams(t *testing.T) {
	cases := []struct {
		name       string
		query      string
		wantPath   string
		wantParams string
	}{
		{"no separator", "op", "op", ""},
		{"with separator", "op?a=1", "op", "a=1"},
		{"trailing separator", "op?", "op", ""},
		{"leading separator", "?a=1", "", "a=1"},
		{"only the first separator splits", "op?a=1?b=2", "op", "a=1?b=2"},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			path, params := splitPathParams(c.query)

			if path != c.wantPath {
				t.Fatalf("path: want %q, got %q", c.wantPath, path)
			}
			if params != c.wantParams {
				t.Fatalf("params: want %q, got %q", c.wantParams, params)
			}
		})
	}
}

// New builds a Query, appending marshalled arguments to the path.

func TestNew_NilArgsLeavesThePathBare(t *testing.T) {
	caller, target := astral.GenerateIdentity(), astral.GenerateIdentity()

	q := New(caller, target, "op", nil)

	if q.QueryString != "op" {
		t.Fatalf("QueryString: want %q, got %q", "op", q.QueryString)
	}
	if !q.Caller.IsEqual(caller) {
		t.Fatal("Caller: want the supplied caller")
	}
	if !q.Target.IsEqual(target) {
		t.Fatal("Target: want the supplied target")
	}
}

func TestNew_AppendsMarshalledArgs(t *testing.T) {
	id := astral.GenerateIdentity()

	cases := []struct {
		name string
		args any
		want string
	}{
		{"string args", "a=1&b=2", "op?a=1&b=2"},
		{"map args", map[string]string{"id": "abc"}, "op?id=abc"},
		{"Args alias", Args{"id": "abc"}, "op?id=abc"},
		{"empty string args", "", "op"},
		{"empty map args", map[string]string{}, "op"},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			q := New(id, id, "op", c.args)

			if q.QueryString != c.want {
				t.Fatalf("QueryString: want %q, got %q", c.want, q.QueryString)
			}
		})
	}
}

// New has no error return, so a marshal failure is indistinguishable from
// having no arguments: the query is built with a bare path and dispatched.
func TestNew_MarshalFailureDropsArgsSilently(t *testing.T) {
	id := astral.GenerateIdentity()

	// an int is not a supported args type — Marshal returns an error
	q := New(id, id, "op", 42)

	if q.QueryString != "op" {
		t.Fatalf("QueryString: want %q, got %q", "op", q.QueryString)
	}
}

// Every query carries a nonce, so two queries built alike remain distinct.
func TestNew_AssignsADistinctNonce(t *testing.T) {
	id := astral.GenerateIdentity()

	a := New(id, id, "op", nil)
	b := New(id, id, "op", nil)

	if a.Nonce == b.Nonce {
		t.Fatalf("want distinct nonces, got %v twice", a.Nonce)
	}
}
