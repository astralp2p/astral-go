package codecguard

import (
	"go/ast"
	"go/token"
)

// hazard says how a field's zero value fails when a codec loads through it.
type hazard int

const (
	// hazardNil is a pointer field: loading through it while nil panics.
	hazardNil hazard = iota
	// hazardInvalid is a reflect.Value field: the zero Value has no type, so every
	// accessor on it panics. This is the shape the three Runtime* carriers shipped.
	hazardInvalid
)

// facts is what the rule needs from one package: which types declare ObjectType, and
// which of their fields carry a hazard.
type facts struct {
	objectTypes map[string]bool
	fields      map[string]map[string]hazard
}

func collect(files []*ast.File) *facts {
	f := &facts{
		objectTypes: map[string]bool{},
		fields:      map[string]map[string]hazard{},
	}

	for _, file := range files {
		for _, decl := range file.Decls {
			switch d := decl.(type) {
			case *ast.GenDecl:
				f.addTypes(d)
			case *ast.FuncDecl:
				f.addObjectType(d)
			}
		}
	}

	return f
}

func (f *facts) addTypes(decl *ast.GenDecl) {
	if decl.Tok != token.TYPE {
		return
	}

	for _, spec := range decl.Specs {
		ts, ok := spec.(*ast.TypeSpec)
		if !ok {
			continue
		}
		st, ok := ts.Type.(*ast.StructType)
		if !ok {
			continue
		}
		if h := hazardFields(st); len(h) > 0 {
			f.fields[ts.Name.Name] = h
		}
	}
}

// hazardFields names the struct's fields whose zero value panics on load, including an
// embedded `*T`, whose field name is the pointed-to type's name.
func hazardFields(st *ast.StructType) map[string]hazard {
	out := map[string]hazard{}

	for _, field := range st.Fields.List {
		h, ok := fieldHazard(field.Type)
		if !ok {
			continue
		}
		if len(field.Names) == 0 {
			if name := embeddedName(field.Type); name != "" {
				out[name] = h
			}
			continue
		}
		for _, n := range field.Names {
			out[n.Name] = h
		}
	}

	return out
}

func fieldHazard(e ast.Expr) (hazard, bool) {
	if _, ok := e.(*ast.StarExpr); ok {
		return hazardNil, true
	}
	if isReflectValue(e) {
		return hazardInvalid, true
	}
	return 0, false
}

// isReflectValue matches the `reflect.Value` type by name. The package alias is assumed
// to be the import default; a renamed `reflect` import makes the rule silent, never
// wrong.
func isReflectValue(e ast.Expr) bool {
	sel, ok := e.(*ast.SelectorExpr)
	if !ok {
		return false
	}
	pkg, ok := sel.X.(*ast.Ident)
	return ok && pkg.Name == "reflect" && sel.Sel.Name == "Value"
}

// embeddedName is the field name Go gives an anonymous field.
func embeddedName(e ast.Expr) string {
	if star, ok := e.(*ast.StarExpr); ok {
		e = star.X
	}
	switch t := e.(type) {
	case *ast.Ident:
		return t.Name
	case *ast.SelectorExpr:
		return t.Sel.Name
	}
	return ""
}

func (f *facts) addObjectType(fn *ast.FuncDecl) {
	if fn.Name.Name != "ObjectType" || fn.Recv == nil {
		return
	}
	if name := receiverType(fn); name != "" {
		f.objectTypes[name] = true
	}
}

// receiverType names the type a method is declared on, with the pointer and any type
// parameters stripped.
func receiverType(fn *ast.FuncDecl) string {
	if fn.Recv == nil || len(fn.Recv.List) != 1 {
		return ""
	}

	e := fn.Recv.List[0].Type
	if star, ok := e.(*ast.StarExpr); ok {
		e = star.X
	}
	switch t := e.(type) {
	case *ast.Ident:
		return t.Name
	case *ast.IndexExpr:
		return exprIdent(t.X)
	case *ast.IndexListExpr:
		return exprIdent(t.X)
	}

	return ""
}

func exprIdent(e ast.Expr) string {
	if id, ok := e.(*ast.Ident); ok {
		return id.Name
	}
	return ""
}

// receiverName is the identifier the body uses for the receiver. An unnamed or blank
// receiver reaches no field and returns "".
func receiverName(fn *ast.FuncDecl) string {
	if fn.Recv == nil || len(fn.Recv.List) != 1 || len(fn.Recv.List[0].Names) != 1 {
		return ""
	}
	if name := fn.Recv.List[0].Names[0].Name; name != "_" {
		return name
	}
	return ""
}

// isCodecMethod matches the `WriteTo`/`ReadFrom` shape astral.Object requires: one
// parameter, two results.
func isCodecMethod(fn *ast.FuncDecl) bool {
	if fn.Recv == nil || fn.Body == nil {
		return false
	}
	if fn.Name.Name != "WriteTo" && fn.Name.Name != "ReadFrom" {
		return false
	}

	t := fn.Type
	return t.Params != nil && len(t.Params.List) == 1 &&
		t.Results != nil && len(t.Results.List) == 2
}

// hasValueReceiver reports whether the method takes its receiver by value, so anything
// it writes into the receiver dies at the return.
func hasValueReceiver(fn *ast.FuncDecl) bool {
	if fn.Recv == nil || len(fn.Recv.List) != 1 {
		return false
	}
	_, isPointer := fn.Recv.List[0].Type.(*ast.StarExpr)
	return !isPointer
}
