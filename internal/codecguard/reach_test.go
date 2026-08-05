package codecguard

import (
	"go/ast"
	"go/token"
	"io/fs"
	"path/filepath"
	"strings"
	"testing"
)

// A rule that matches nothing is quiet for the wrong reason, and a green module proves
// nothing without a count of what the rule actually read. Break receiverType, or the
// ObjectType filter, and the module still passes TestModuleIsClean — but not this.
//
// The floor is well under the 126 methods measured when the rule was written; it exists
// to catch a rule that stopped matching, not to pin the module's size.
const minCodecMethodsWithHazard = 50

func TestRuleReachesTheModulesCodecs(t *testing.T) {
	fset := token.NewFileSet()
	reached := 0

	err := filepath.WalkDir("../..", func(path string, d fs.DirEntry, err error) error {
		if err != nil || !d.IsDir() {
			return err
		}
		if path != "../.." && (skipNames[d.Name()] || strings.HasPrefix(d.Name(), ".")) {
			return fs.SkipDir
		}
		reached += hazardMethods(fset, t, path)
		return nil
	})
	if err != nil {
		t.Fatalf("walk: %v", err)
	}

	if reached < minCodecMethodsWithHazard {
		t.Errorf("the rule read %d codec methods carrying a field it can report on, want at least %d: "+
			"a filter is matching less than it did", reached, minCodecMethodsWithHazard)
	}
}

// hazardMethods counts the WriteTo/ReadFrom methods in one package that the rule both
// recognizes as a codec and finds a reportable field on.
func hazardMethods(fset *token.FileSet, t *testing.T, dir string) int {
	files, err := parseDir(fset, dir)
	if err != nil {
		t.Fatalf("parse %s: %v", dir, err)
	}

	f := collect(files)

	count := 0
	for _, file := range files {
		for _, decl := range file.Decls {
			fn, ok := decl.(*ast.FuncDecl)
			if !ok || !isCodecMethod(fn) {
				continue
			}
			name := receiverType(fn)
			if f.objectTypes[name] && len(f.fields[name]) > 0 {
				count++
			}
		}
	}

	return count
}
