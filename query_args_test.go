package pub_test

import (
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"testing"
)

// An op binds its arguments by snake_cased, lower-cased field name, and silently
// skips a key that matches nothing. A capitalised key therefore reaches the wire
// verbatim, matches nothing, and is dropped without complaint — the op runs with
// zero values and reports success. Nothing at run time reports it, so the check is
// static.
var snakeCaseKey = regexp.MustCompile(`^[a-z][a-z0-9_]*$`)

func TestQueryArgsKeysAreSnakeCase(t *testing.T) {
	fset := token.NewFileSet()
	var offenders []string

	err := filepath.WalkDir(".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			if d.Name() == ".git" || d.Name() == ".ai" {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(path, ".go") {
			return nil
		}

		file, perr := parser.ParseFile(fset, path, nil, 0)
		if perr != nil {
			return perr
		}

		ast.Inspect(file, func(n ast.Node) bool {
			lit, ok := n.(*ast.CompositeLit)
			if !ok || !isQueryArgs(lit.Type) {
				return true
			}

			for _, elt := range lit.Elts {
				kv, ok := elt.(*ast.KeyValueExpr)
				if !ok {
					continue
				}
				key, ok := kv.Key.(*ast.BasicLit)
				if !ok || key.Kind != token.STRING {
					continue
				}
				name, uerr := strconv.Unquote(key.Value)
				if uerr != nil || snakeCaseKey.MatchString(name) {
					continue
				}
				offenders = append(offenders,
					fset.Position(key.Pos()).String()+": query.Args key "+strconv.Quote(name))
			}
			return true
		})

		return nil
	})
	if err != nil {
		t.Fatalf("walk: %v", err)
	}

	if len(offenders) > 0 {
		t.Errorf("query.Args keys must be snake_case; the node drops the rest silently:\n  %s",
			strings.Join(offenders, "\n  "))
	}
}

// isQueryArgs matches the composite literal type query.Args, however the package is
// named at the use site.
func isQueryArgs(expr ast.Expr) bool {
	sel, ok := expr.(*ast.SelectorExpr)
	if !ok {
		return false
	}
	return sel.Sel.Name == "Args"
}
