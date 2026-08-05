package codecguard

import (
	"fmt"
	"go/token"
	"sort"
	"testing"
)

// key is what a test asserts on: which rule fired, on which method, over which field.
// Line numbers are left out so a comment added to testdata does not fail the test.
func key(f Finding) string {
	return fmt.Sprintf("%s %s.%s %s", f.Rule, f.Type, f.Method, f.Field)
}

func keys(found []Finding) []string {
	out := make([]string, 0, len(found))
	for _, f := range found {
		out = append(out, key(f))
	}
	sort.Strings(out)
	return out
}

func check(t *testing.T, dir string) []Finding {
	t.Helper()

	found, err := CheckDir(token.NewFileSet(), dir)
	if err != nil {
		t.Fatalf("CheckDir(%s): %v", dir, err)
	}
	return found
}

// Every want below but lateGuard is a defect this module shipped. The rule exists
// because those four reached users, and a fifth of the same shape would otherwise reach
// them too; lateGuard pins that a guard placed after the load does not count.
func TestShippedDefectsAreReported(t *testing.T) {
	want := []string{
		"discarded-decode endpointMapping.ReadFrom ",
		"unguarded-load lateGuard.WriteTo Identity",
		"unguarded-load nodeInfo.ReadFrom Identity",
		"unguarded-load nodeInfo.WriteTo Identity",
		"unguarded-load runtimeSlice.ReadFrom ptr",
		"unguarded-load runtimeSlice.WriteTo ptr",
		"unguarded-load stringView.ReadFrom str",
		"unguarded-load stringView.WriteTo str",
	}

	got := keys(check(t, "testdata/defects"))
	if len(got) != len(want) {
		t.Fatalf("want %d findings, got %d: %v", len(want), len(got), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("finding %d: want %q, got %q", i, want[i], got[i])
		}
	}
}

// A linter that fires on correct code gets turned off. Every shape here is one the rule
// was measured against, including the reflective carriers that made the naive
// value-receiver rule useless.
func TestCorrectCodecsAreNotReported(t *testing.T) {
	for _, f := range check(t, "testdata/clean") {
		t.Errorf("%v", f)
	}
}

// The module itself is the standing regression pin: `go test ./...` fails the moment a
// codec of either shape lands, with no CI step required.
func TestModuleIsClean(t *testing.T) {
	found, err := CheckTree(token.NewFileSet(), "../..")
	if err != nil {
		t.Fatalf("CheckTree: %v", err)
	}
	for _, f := range found {
		t.Errorf("%v", f)
	}
}
