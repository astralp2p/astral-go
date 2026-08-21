package codecguard

import (
	"fmt"
	"go/ast"
	"go/token"
)

func checkMethod(fset *token.FileSet, f *facts, fn *ast.FuncDecl) []Finding {
	typeName := receiverType(fn)
	if !isCodecMethod(fn) || !f.objectTypes[typeName] {
		return nil
	}

	found := checkDiscardedDecode(fset, typeName, fn)
	return append(found, checkUnguardedLoad(fset, f.fields[typeName], typeName, fn)...)
}

// checkDiscardedDecode reports a ReadFrom that decodes into the address of a value
// receiver. The receiver is a copy, so every byte read lands in a value the return
// discards; `astral.Objectify` additionally panics on a non-pointer, which made
// `kcp.EndpointLocalMapping.ReadFrom` fail on every call it ever received.
func checkDiscardedDecode(fset *token.FileSet, typeName string, fn *ast.FuncDecl) []Finding {
	recv := receiverName(fn)
	if fn.Name.Name != "ReadFrom" || !hasValueReceiver(fn) || recv == "" {
		return nil
	}

	var found []Finding
	ast.Inspect(fn.Body, func(n ast.Node) bool {
		u, ok := n.(*ast.UnaryExpr)
		if !ok || u.Op != token.AND || exprIdent(u.X) != recv {
			return true
		}
		found = append(found, Finding{
			Pos:    fset.Position(u.Pos()),
			Rule:   RuleDiscardedDecode,
			Type:   typeName,
			Method: fn.Name.Name,
			Detail: fmt.Sprintf("decodes into &%s on a value receiver, so the "+
				"decoded value dies at the return", recv),
		})
		return true
	})

	return found
}

// loadCheck holds the state the unguarded-load rule accumulates over one method body.
type loadCheck struct {
	recv    string
	fields  map[string]hazard
	aliases map[string]string      // local identifier -> receiver field it copies
	guards  map[string][]token.Pos // field -> positions from which a load is proven safe
	pass    map[ast.Node]bool      // nodes that copy or prove a field, never load through it
}

// checkUnguardedLoad reports each receiver field the method loads through before
// anything proves the field usable.
//
// Proof takes three forms, all measured against the shapes already in this module: a
// nil comparison, an `IsValid` call, and an assignment to the field — an initialisation
// is as good as a check. A local copy `x := recv.f` is followed, so proving `x` proves
// the field; that is how `NodeInfo.WriteTo` reads, and treating the copy as a load put
// it on the hit list of the first draft.
func checkUnguardedLoad(fset *token.FileSet, fields map[string]hazard, typeName string, fn *ast.FuncDecl) []Finding {
	recv := receiverName(fn)
	if recv == "" || len(fields) == 0 {
		return nil
	}

	c := &loadCheck{
		recv:    recv,
		fields:  fields,
		aliases: map[string]string{},
		guards:  map[string][]token.Pos{},
		pass:    map[ast.Node]bool{},
	}
	c.bindAliases(fn.Body)
	c.scanGuards(fn.Body)

	return c.loads(fset, typeName, fn)
}

// bindAliases records `x := recv.f`, which copies the field rather than loading through
// it. Both sides are marked so neither counts as a load.
func (c *loadCheck) bindAliases(body *ast.BlockStmt) {
	ast.Inspect(body, func(n ast.Node) bool {
		assign, ok := n.(*ast.AssignStmt)
		if !ok || len(assign.Lhs) != len(assign.Rhs) {
			return true
		}

		for i, rhs := range assign.Rhs {
			name := exprIdent(assign.Lhs[i])
			field := c.field(rhs)
			if name == "" || name == "_" || field == "" {
				continue
			}
			c.aliases[name] = field
			c.pass[assign.Lhs[i]], c.pass[rhs] = true, true
		}
		return true
	})
}

// scanGuards records every position from which a field is proven usable, and marks the
// expressions that do the proving so they are not read back as loads. Taking a field's
// address is marked too: `&recv.f` reads the slot, not the value behind it.
func (c *loadCheck) scanGuards(body *ast.BlockStmt) {
	ast.Inspect(body, func(n ast.Node) bool {
		switch e := n.(type) {
		case *ast.AssignStmt:
			c.guardAssign(e)
		case *ast.BinaryExpr:
			c.guardNilCompare(e)
		case *ast.CallExpr:
			c.guardIsValid(e)
		case *ast.UnaryExpr:
			if e.Op == token.AND && c.target(e.X) != "" {
				c.pass[e.X] = true
			}
		}
		return true
	})
}

// guardAssign treats an assignment to a field, or to a copy of one, as proof from the
// end of the statement onwards.
func (c *loadCheck) guardAssign(assign *ast.AssignStmt) {
	for _, lhs := range assign.Lhs {
		if c.pass[lhs] {
			continue // the alias-creating assignment proves nothing about the field
		}
		if field := c.target(lhs); field != "" {
			c.pass[lhs] = true
			c.guards[field] = append(c.guards[field], assign.End())
		}
	}
}

func (c *loadCheck) guardNilCompare(e *ast.BinaryExpr) {
	if e.Op != token.EQL && e.Op != token.NEQ {
		return
	}

	sides := [2][2]ast.Expr{{e.X, e.Y}, {e.Y, e.X}}
	for _, s := range sides {
		side, other := s[0], s[1]
		if exprIdent(other) != "nil" {
			continue
		}
		if field := c.target(side); field != "" {
			c.pass[side] = true
			c.guards[field] = append(c.guards[field], e.End())
		}
	}
}

// guardIsValid accepts `f.IsValid()` as proof for a reflect.Value field, the only
// question that can be asked of one before its type is known.
func (c *loadCheck) guardIsValid(call *ast.CallExpr) {
	sel, ok := call.Fun.(*ast.SelectorExpr)
	if !ok || sel.Sel.Name != "IsValid" {
		return
	}

	if field := c.target(sel.X); field != "" {
		c.pass[sel.X] = true
		c.guards[field] = append(c.guards[field], call.End())
	}
}

// loads walks the body once more and reports every remaining mention of a field, or of
// a copy of one, that no earlier position proves.
func (c *loadCheck) loads(fset *token.FileSet, typeName string, fn *ast.FuncDecl) []Finding {
	skip := selectorNames(fn.Body)

	var found []Finding
	ast.Inspect(fn.Body, func(n ast.Node) bool {
		e, ok := n.(ast.Expr)
		if !ok || c.pass[n] || skip[n] {
			return true
		}
		field := c.target(e)
		if field == "" || c.proven(field, e.Pos()) {
			return true
		}
		found = append(found, Finding{
			Pos:    fset.Position(e.Pos()),
			Rule:   RuleUnguardedLoad,
			Type:   typeName,
			Method: fn.Name.Name,
			Field:  field,
			Detail: c.detail(field),
		})
		return true
	})

	return found
}

func (c *loadCheck) detail(field string) string {
	if c.fields[field] == hazardInvalid {
		return fmt.Sprintf("loads %s, a reflect.Value field, with no IsValid guard: "+
			"the zero Value has no type and every accessor on it panics", field)
	}
	return fmt.Sprintf("loads %s, a pointer field, with no nil guard: an astral "+
		"primitive declares WriteTo on the value receiver, so a nil pointer panics", field)
}

func (c *loadCheck) proven(field string, pos token.Pos) bool {
	for _, at := range c.guards[field] {
		if at <= pos {
			return true
		}
	}
	return false
}

// target names the receiver field an expression stands for: the field itself, or a
// local copy of it.
func (c *loadCheck) target(e ast.Expr) string {
	if field := c.field(e); field != "" {
		return field
	}
	return c.aliases[exprIdent(e)]
}

// field names the receiver field a selector reads, or "" for anything else.
func (c *loadCheck) field(e ast.Expr) string {
	sel, ok := e.(*ast.SelectorExpr)
	if !ok || exprIdent(sel.X) != c.recv {
		return ""
	}
	if _, ok := c.fields[sel.Sel.Name]; !ok {
		return ""
	}
	return sel.Sel.Name
}

// selectorNames collects the identifiers that name a member rather than reference a
// variable. A field and a local copy of it often share a name, and without this the
// walk reads `v.str` a second time through its own `Sel`.
func selectorNames(body *ast.BlockStmt) map[ast.Node]bool {
	skip := map[ast.Node]bool{}

	ast.Inspect(body, func(n ast.Node) bool {
		switch e := n.(type) {
		case *ast.SelectorExpr:
			skip[e.Sel] = true
		case *ast.KeyValueExpr:
			if _, ok := e.Key.(*ast.Ident); ok {
				skip[e.Key] = true
			}
		}
		return true
	})

	return skip
}
