// Package codecguard reports a hand-written astral codec that loads through a receiver
// field the zero value leaves unusable, or that decodes into a copy it discards.
//
// why: astral primitives declare WriteTo on the value receiver, so Go's synthesized
// pointer method dereferences before the body runs — a nil `*Identity` handed to a
// codec panics instead of erroring. Every registered type is materialized at its zero
// value by `objects.new`, which any anonymous local query can reach, so a zero-value
// field that no constructor would produce is still on the wire path. Four shipped
// defects had one of these two shapes.
//
// The rule is narrow on purpose. It reads only `WriteTo` and `ReadFrom`, only on types
// declaring `ObjectType`, and only fields of the receiver itself. A wider rule — "a
// value-receiver `ReadFrom` is an error" — measured 20 false positives out of 23 on
// this module, because the reflective carriers wrap a `reflect.Value` for which a value
// receiver is correct. That wider question belongs to a runtime sweep over registered
// types, where it is exact; see `count_audit_test.go`.
//
// A static rule is what reaches the rest. `styles.StringView` is never `astral.Add`-ed,
// so no runtime enumeration materializes it — which is exactly how it shipped a
// byte-identical copy of the `NodeInfo` crash and no test noticed.
package codecguard

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// Rule names the check that produced a Finding.
type Rule string

const (
	// RuleUnguardedLoad: a codec loads through a receiver field whose zero value panics.
	RuleUnguardedLoad Rule = "unguarded-load"
	// RuleDiscardedDecode: a value-receiver ReadFrom decodes into its own copy.
	RuleDiscardedDecode Rule = "discarded-decode"
)

// Finding is one reported defect.
type Finding struct {
	Pos    token.Position
	Rule   Rule
	Type   string
	Method string
	Field  string // the receiver field at fault; empty when the whole method is
	Detail string
}

func (f Finding) String() string {
	return fmt.Sprintf("%s: [%s] %s.%s %s", f.Pos, f.Rule, f.Type, f.Method, f.Detail)
}

// skipNames are directories the tree walk never descends into. `testdata` holds the
// deliberate defects this package's own tests assert on, and Go ignores it too.
var skipNames = map[string]bool{"testdata": true, "vendor": true, "node_modules": true}

// CheckTree reports findings for every Go package under root. Dot-directories are
// skipped, which excludes `.ai/system` — the astral-docs submodule is not our source.
func CheckTree(fset *token.FileSet, root string) ([]Finding, error) {
	var found []Finding

	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		switch {
		case err != nil:
			return err
		case !d.IsDir(), path == root:
		case skipNames[d.Name()], strings.HasPrefix(d.Name(), "."):
			return fs.SkipDir
		}
		if !d.IsDir() {
			return nil
		}

		f, err := CheckDir(fset, path)
		found = append(found, f...)
		return err
	})

	return found, err
}

// CheckDir reports findings for the Go package in dir, ignoring subdirectories. Test
// files are excluded: a codec lives in the file that declares its type, and a test may
// construct a broken value on purpose.
func CheckDir(fset *token.FileSet, dir string) ([]Finding, error) {
	files, err := parseDir(fset, dir)
	if err != nil {
		return nil, err
	}

	facts := collect(files)

	var found []Finding
	for _, file := range files {
		for _, decl := range file.Decls {
			fn, ok := decl.(*ast.FuncDecl)
			if !ok {
				continue
			}
			found = append(found, checkMethod(fset, facts, fn)...)
		}
	}

	sort.Slice(found, func(i, j int) bool {
		if found[i].Pos.Filename != found[j].Pos.Filename {
			return found[i].Pos.Filename < found[j].Pos.Filename
		}
		return found[i].Pos.Offset < found[j].Pos.Offset
	})

	return found, nil
}

func parseDir(fset *token.FileSet, dir string) ([]*ast.File, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	var files []*ast.File
	for _, e := range entries {
		name := e.Name()
		if e.IsDir() || !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			continue
		}

		file, err := parser.ParseFile(fset, filepath.Join(dir, name), nil, parser.SkipObjectResolution)
		if err != nil {
			return nil, err
		}
		files = append(files, file)
	}

	return files, nil
}
