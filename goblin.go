package goblin

import (
	"encoding/json"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"reflect"
	"strings"
)

/* TODO: add something like this to catch nils:

func NilNotAllowed(f interface {}) interface {
    if (f == nil) panic();
    return nil
}

and perhaps a corresponding NilAllowed? mumble mumble million-dollar mistake mumble

*/

func DumpPosition(p token.Position) map[string]interface{} {
	return map[string]interface{}{
		// We need these float64 conversions or our test cases will fail.
		// Believe me, I'm as angry about this as you are.
		"filename": p.Filename,
		"line":     float64(p.Line),
		"offset":   float64(p.Offset),
		"column":   float64(p.Column),
	}
}

func DumpIdent(i *ast.Ident, fset *token.FileSet) map[string]interface{} {
	if i == nil {
		return nil
	}

	asLiteral := map[string]interface{}{
		"kind":     "literal",
		"type":     "BOOL",
		"position": DumpPosition(fset.Position(i.Pos())),
	}

	switch i.Name {
	case "true":
		asLiteral["value"] = "true"
		return asLiteral

	case "false":
		asLiteral["value"] = "false"
		return asLiteral

	case "iota":
		asLiteral["type"] = "IOTA"
		return asLiteral

	}

	return map[string]interface{}{
		"kind":     "ident",
		"value":    i.Name,
		"position": DumpPosition(fset.Position(i.Pos())),
	}
}

func DumpArray(a *ast.ArrayType, fset *token.FileSet) map[string]interface{} {
	return map[string]interface{}{
		"kind":     "array",
		"length":   DumpExpr(a.Len, fset),
		"element":  DumpExprAsType(a.Elt, fset),
		"position": DumpPosition(fset.Position(a.Pos())),
	}
}

func AttemptExprAsType(e ast.Expr, fset *token.FileSet) map[string]interface{} {
	if e == nil {
		return nil
	}

	if n, ok := e.(*ast.ParenExpr); ok {
		return AttemptExprAsType(n.X, fset)
	}

	if n, ok := e.(*ast.Ident); ok {
		return map[string]interface{}{
			"kind":     "type",
			"type":     "identifier",
			"value":    DumpIdent(n, fset),
			"position": DumpPosition(fset.Position(e.Pos())),
		}
	}

	if n, ok := e.(*ast.SelectorExpr); ok {
		lhs := DumpExpr(n.X, fset)

		if lhs["type"] == "identifier" && lhs["qualifier"] == nil {
			return map[string]interface{}{
				"kind":      "type",
				"type":      "identifier",
				"qualifier": lhs["value"],
				"value":     DumpIdent(n.Sel, fset),
				"position":  DumpPosition(fset.Position(e.Pos())),
			}
		}
	}

	if n, ok := e.(*ast.ArrayType); ok {
		if n.Len == nil {
			return map[string]interface{}{
				"kind":     "type",
				"type":     "slice",
				"element":  DumpExprAsType(n.Elt, fset),
				"position": DumpPosition(fset.Position(e.Pos())),
			}
		} else {
			return map[string]interface{}{
				"kind":     "type",
				"type":     "array",
				"element":  DumpExprAsType(n.Elt, fset),
				"length":   DumpExpr(n.Len, fset),
				"position": DumpPosition(fset.Position(e.Pos())),
			}
		}
	}

	if n, ok := e.(*ast.StarExpr); ok {
		return map[string]interface{}{
			"kind":      "type",
			"type":      "pointer",
			"contained": DumpExprAsType(n.X, fset),
			"position":  DumpPosition(fset.Position(e.Pos())),
		}
	}

	if n, ok := e.(*ast.InterfaceType); ok {
		return map[string]interface{}{
			"kind":       "type",
			"type":       "interface",
			"incomplete": n.Incomplete,
			"methods":    DumpFields(n.Methods, fset),
			"position":   DumpPosition(fset.Position(e.Pos())),
		}
	}

	if n, ok := e.(*ast.MapType); ok {
		return map[string]interface{}{
			"kind":     "type",
			"type":     "map",
			"key":      DumpExprAsType(n.Key, fset),
			"value":    DumpExprAsType(n.Value, fset),
			"position": DumpPosition(fset.Position(e.Pos())),
		}
	}

	if n, ok := e.(*ast.ChanType); ok {
		return map[string]interface{}{
			"kind":      "type",
			"type":      "chan",
			"direction": DumpChanDir(n.Dir),
			"value":     DumpExprAsType(n.Value, fset),
			"position":  DumpPosition(fset.Position(e.Pos())),
		}
	}

	if n, ok := e.(*ast.StructType); ok {
		return map[string]interface{}{
			"kind":     "type",
			"type":     "struct",
			"fields":   DumpFields(n.Fields, fset),
			"position": DumpPosition(fset.Position(e.Pos())),
		}
	}

	if n, ok := e.(*ast.FuncType); ok {
		return map[string]interface{}{
			"kind":     "type",
			"type":     "function",
			"params":   DumpFields(n.Params, fset),
			"results":  DumpFields(n.Results, fset),
			"position": DumpPosition(fset.Position(e.Pos())),
		}
	}

	return nil
}

func DumpExprAsType(e ast.Expr, fset *token.FileSet) map[string]interface{} {
	result := AttemptExprAsType(e, fset)

	if result != nil {
		return result
	}

	// bail out
	gotten := reflect.TypeOf(e).String()
	pos := fset.PositionFor(e.Pos(), true).String()
	panic("Unrecognized type " + gotten + " in expr-as-type at " + pos)
}

func DumpChanDir(d ast.ChanDir) string {
	switch d {
	case ast.SEND:
		return "send"

	case ast.RECV:
		return "recv"

	case ast.SEND | ast.RECV:
		return "both"
	}

	panic("Unrecognized ChanDir value " + string(d))
}

func DumpExpr(e ast.Expr, fset *token.FileSet) map[string]interface{} {
	if e == nil {
		return nil
	}

	if _, ok := e.(*ast.ArrayType); ok {
		return DumpExprAsType(e, fset)
	}

	if n, ok := e.(*ast.Ident); ok {

		val := DumpIdent(n, fset)

		if val["type"] == "BOOL" {
			return val
		}

		return map[string]interface{}{
			"kind":     "expression",
			"type":     "identifier",
			"value":    val,
			"position": DumpPosition(fset.Position(e.Pos())),
		}
	}

	if n, ok := e.(*ast.Ellipsis); ok {
		return map[string]interface{}{
			"type":  "ellipsis",
			"kind":  "type",
			"value": DumpExpr(n.Elt, fset),
		}
	}

	// is this the right place??
	if n, ok := e.(*ast.FuncLit); ok {
		return map[string]interface{}{
			"kind":     "literal",
			"type":     "function",
			"params":   DumpFields(n.Type.Params, fset),
			"results":  DumpFields(n.Type.Results, fset),
			"body":     DumpBlock(n.Body, fset),
			"position": DumpPosition(fset.Position(e.Pos())),
		}
	}

	if n, ok := e.(*ast.BasicLit); ok {
		return DumpBasicLit(n, fset)
	}

	if n, ok := e.(*ast.CompositeLit); ok {
		return map[string]interface{}{
			"kind":     "literal",
			"type":     "composite",
			"declared": DumpExprAsType(n.Type, fset),
			"values":   DumpExprs(n.Elts, fset),
			"position": DumpPosition(fset.Position(e.Pos())),
		}
	}

	if n, ok := e.(*ast.BinaryExpr); ok {
		return DumpBinaryExpr(n, fset)
	}

	if n, ok := e.(*ast.IndexExpr); ok {
		return map[string]interface{}{
			"kind":     "expression",
			"type":     "index",
			"target":   DumpExpr(n.X, fset),
			"index":    DumpExpr(n.Index, fset),
			"position": DumpPosition(fset.Position(e.Pos())),
		}
	}

	if n, ok := e.(*ast.StarExpr); ok {
		return map[string]interface{}{
			"kind":   "expression",
			"type":   "star",
			"target": DumpExpr(n.X, fset),
		}
	}

	if n, ok := e.(*ast.CallExpr); ok {

		return DumpCall(n, fset)
	}

	if n, ok := e.(*ast.ParenExpr); ok {
		return map[string]interface{}{
			"kind":     "expression",
			"type":     "paren",
			"target":   DumpExpr(n.X, fset),
			"position": DumpPosition(fset.Position(e.Pos())),
		}
	}

	if n, ok := e.(*ast.SelectorExpr); ok {
		lhs := DumpExpr(n.X, fset)
		// If the left hand side is just an identifier without a further qualifier,
		// assume that this is a qualified expression rather than a method call.
		// this is not correct in all cases, but ensuring correctness is outside
		// of the scope of a lowly parser such as goblin.
		if lhs["type"] == "identifier" && lhs["qualifier"] == nil {
			return map[string]interface{}{
				"kind":      "expression",
				"type":      "identifier",
				"qualifier": lhs["value"],
				"value":     DumpIdent(n.Sel, fset),
				"position":  DumpPosition(fset.Position(e.Pos())),
			}
		}

		return map[string]interface{}{
			"kind":     "expression",
			"type":     "selector",
			"target":   lhs,
			"field":    DumpIdent(n.Sel, fset),
			"position": DumpPosition(fset.Position(e.Pos())),
		}
	}

	if n, ok := e.(*ast.TypeAssertExpr); ok {
		return map[string]interface{}{
			"kind":     "expression",
			"type":     "type-assert",
			"target":   DumpExpr(n.X, fset),
			"asserted": DumpExprAsType(n.Type, fset),
			"position": DumpPosition(fset.Position(e.Pos())),
		}
	}

	if n, ok := e.(*ast.UnaryExpr); ok {
		return map[string]interface{}{
			"kind":     "unary",
			"target":   DumpExpr(n.X, fset),
			"operator": n.Op.String(),
			"position": DumpPosition(fset.Position(n.Pos())),
		}
	}

	if n, ok := e.(*ast.SliceExpr); ok {
		return map[string]interface{}{
			"kind":     "expression",
			"type":     "slice",
			"target":   DumpExpr(n.X, fset),
			"low":      DumpExpr(n.Low, fset),
			"high":     DumpExpr(n.High, fset),
			"max":      DumpExpr(n.Max, fset),
			"three":    n.Slice3,
			"position": DumpPosition(fset.Position(e.Pos())),
		}
	}

	if n, ok := e.(*ast.KeyValueExpr); ok {
		return map[string]interface{}{
			"kind":  "expression",
			"type":  "key-value",
			"key":   DumpExpr(n.Key, fset),
			"value": DumpExpr(n.Value, fset),
		}
	}

	if n, ok := e.(*ast.BadExpr); ok {
		pos := fset.PositionFor(n.From, true).String()
		panic("Encountered BadExpr at " + pos + "; bailing out")
	}

	typ := reflect.TypeOf(e).String()
	panic("Encountered unexpected " + typ + " node while processing an expression; bailing out")
}

func DumpExprs(exprs []ast.Expr, fset *token.FileSet) []interface{} {
	values := make([]interface{}, len(exprs))
	for i, v := range exprs {
		values[i] = DumpExpr(v, fset)
	}

	return values
}

func DumpBinaryExpr(b *ast.BinaryExpr, fset *token.FileSet) map[string]interface{} {
	return map[string]interface{}{
		"type":     "expression",
		"kind":     "binary",
		"left":     DumpExpr(b.X, fset),
		"right":    DumpExpr(b.Y, fset),
		"operator": b.Op.String(),
		"position": DumpPosition(fset.Position(b.Pos())),
	}
}

func DumpBasicLit(l *ast.BasicLit, fset *token.FileSet) map[string]interface{} {
	if l == nil {
		return nil
	}

	return map[string]interface{}{
		"kind":     "literal",
		"type":     l.Kind.String(),
		"value":    l.Value,
		"position": DumpPosition(fset.Position(l.Pos())),
	}
}

func DumpField(f *ast.Field, fset *token.FileSet) map[string]interface{} {

	nameCount := 0
	if f.Names != nil {
		nameCount = len(f.Names)
	}

	names := make([]interface{}, nameCount)
	if f.Names != nil {
		for i, v := range f.Names {
			names[i] = DumpIdent(v, fset)
		}
	}

	return map[string]interface{}{
		"kind":          "field",
		"names":         names,
		"declared-type": DumpExprAsType(f.Type, fset),
		"tag":           DumpBasicLit(f.Tag, fset),
	}
}

func DumpFields(fs *ast.FieldList, fset *token.FileSet) []map[string]interface{} {
	if fs == nil {
		return nil
	}

	results := make([]map[string]interface{}, len(fs.List))
	for i, v := range fs.List {
		results[i] = DumpField(v, fset)
	}

	return results
}

func DumpCommentGroup(g *ast.CommentGroup, fset *token.FileSet) []string {
	if g == nil {
		return []string{}
	}

	result := make([]string, len(g.List))
	for i, v := range g.List {
		result[i] = v.Text
	}

	return result
}

func DumpTypeAlias(t *ast.TypeSpec, fset *token.FileSet) map[string]interface{} {
	return map[string]interface{}{
		"kind":     "decl",
		"type":     "type-alias",
		"name":     DumpIdent(t.Name, fset),
		"value":    DumpExprAsType(t.Type, fset),
		"comments": DumpCommentGroup(t.Comment, fset),
		"position": DumpPosition(fset.Position(t.Pos())),
	}
}

func DumpCall(c *ast.CallExpr, fset *token.FileSet) map[string]interface{} {
	if callee, ok := c.Fun.(*ast.Ident); ok {
		if callee.Name == "new" {
			return map[string]interface{}{
				"kind":     "expression",
				"type":     "new",
				"argument": DumpExprAsType(c.Args[0], fset),
				"position": DumpPosition(fset.Position(c.Pos())),
			}
		}

		if callee.Name == "make" {
			return map[string]interface{}{
				"kind":     "expression",
				"type":     "make",
				"argument": DumpExprAsType(c.Args[0], fset),
				"rest":     DumpExprs(c.Args[1:], fset),
				"position": DumpPosition(fset.Position(c.Pos())),
			}
		}
	}

	// try to parse the LHS as a type. if it succeeds and is *not* an identifier name,
	// it's a cast. currently, we don't have any heuristics for determining whether an
	// identifier is a typename (we don't even do the obvious cases like int8, float64
	// et cetera). such heuristics can't be perfectly accurate due to cross-module type
	// declarations, so it's probably more morally-correct, if less helpful, to treat them
	// as function calls and disambiguate them at a further stage.
	callee := AttemptExprAsType(c.Fun, fset)

	if callee != nil && callee["type"] != "identifier" {
		return map[string]interface{}{
			"kind":       "expression",
			"type":       "cast",
			"target":     DumpExpr(c.Args[0], fset),
			"coerced-to": callee,
			"position":   DumpPosition(fset.Position(c.Pos())),
		}
	}

	callee = DumpExpr(c.Fun, fset)

	return map[string]interface{}{
		"kind":      "expression",
		"type":      "call",
		"function":  callee,
		"arguments": DumpExprs(c.Args, fset),
		"ellipsis":  c.Ellipsis != token.NoPos,
		"position":  DumpPosition(fset.Position(c.Pos())),
	}
}

func DumpImport(spec *ast.ImportSpec, fset *token.FileSet) map[string]interface{} {
	res := map[string]interface{}{
		"type":     "import",
		"doc":      DumpCommentGroup(spec.Doc, fset),
		"comments": DumpCommentGroup(spec.Comment, fset),
		"name":     DumpIdent(spec.Name, fset),
		"path":     strings.Trim(spec.Path.Value, "\""),
		"position": DumpPosition(fset.Position(spec.Pos())),
	}

	return res
}

func DumpValue(kind string, spec *ast.ValueSpec, fset *token.FileSet) map[string]interface{} {
	givenValues := []ast.Expr{}
	if spec.Values != nil {
		givenValues = spec.Values
	}

	processedValues := make([]interface{}, len(givenValues))
	for i, v := range givenValues {
		processedValues[i] = DumpExpr(v, fset)
	}

	processedNames := make([]interface{}, len(spec.Names))
	for i, v := range spec.Names {
		processedNames[i] = DumpIdent(v, fset)
	}

	return map[string]interface{}{
		"kind":          "spec",
		"type":          kind,
		"names":         processedNames,
		"declared-type": AttemptExprAsType(spec.Type, fset),
		"values":        processedValues,
		"comments":      DumpCommentGroup(spec.Comment, fset),
		"position":      DumpPosition(fset.Position(spec.Pos())),
	}
}

func DumpGenDecl(decl *ast.GenDecl, fset *token.FileSet) map[string]interface{} {
	prettyToken := ""
	results := make([]interface{}, len(decl.Specs))
	switch decl.Tok {
	case token.TYPE:
		if len(decl.Specs) != 1 {
			pos := fset.PositionFor(decl.Pos(), true).String()
			panic("Unexpected number of tokens (" + string(len(decl.Specs)) + " in type alias at " + pos)
		}
		// EARLY RETURN
		return DumpTypeAlias(decl.Specs[0].(*ast.TypeSpec), fset)

	case token.IMPORT:
		prettyToken = "import"
		for i, v := range decl.Specs {
			results[i] = DumpImport(v.(*ast.ImportSpec), fset)
		}
	case token.CONST:
		prettyToken = "const"
		for i, v := range decl.Specs {
			results[i] = DumpValue("const", v.(*ast.ValueSpec), fset)
		}

	case token.VAR:
		prettyToken = "var"
		for i, v := range decl.Specs {
			results[i] = DumpValue("var", v.(*ast.ValueSpec), fset)
		}
	default:
		pos := fset.PositionFor(decl.Pos(), true).String()
		panic("Unrecognized token " + decl.Tok.String() + " in GenDecl at " + pos)
	}

	return map[string]interface{}{
		"kind":     "decl",
		"type":     prettyToken,
		"specs":    results,
		"position": DumpPosition(fset.Position(decl.Pos())),
	}
}

func DumpStmt(s ast.Stmt, fset *token.FileSet) interface{} {
	if s == nil {
		return nil
	}

	if n, ok := s.(*ast.ReturnStmt); ok {
		return map[string]interface{}{
			"kind":     "statement",
			"type":     "return",
			"values":   DumpExprs(n.Results, fset),
			"position": DumpPosition(fset.Position(n.Pos())),
		}
	}

	if n, ok := s.(*ast.AssignStmt); ok {
		if n.Tok == token.ASSIGN {
			return map[string]interface{}{
				"kind":     "statement",
				"type":     "assign",
				"left":     DumpExprs(n.Lhs, fset),
				"right":    DumpExprs(n.Rhs, fset),
				"position": DumpPosition(fset.Position(n.Pos())),
			}

		} else if n.Tok == token.DEFINE {
			return map[string]interface{}{
				"kind":     "statement",
				"type":     "define",
				"left":     DumpExprs(n.Lhs, fset),
				"right":    DumpExprs(n.Rhs, fset),
				"position": DumpPosition(fset.Position(n.Pos())),
			}
		} else {
			tok := n.Tok.String()
			return map[string]interface{}{
				"kind":     "statement",
				"type":     "assign-operator",
				"operator": tok[0 : len(tok)-1],
				"left":     DumpExprs(n.Lhs, fset),
				"right":    DumpExprs(n.Rhs, fset),
				"position": DumpPosition(fset.Position(n.Pos())),
			}
		}

	}

	if n, ok := s.(*ast.EmptyStmt); ok {
		return map[string]interface{}{
			"kind":     "statement",
			"type":     "empty",
			"position": DumpPosition(fset.Position(n.Pos())),
		}
	}

	if n, ok := s.(*ast.ExprStmt); ok {
		return map[string]interface{}{
			"kind":  "statement",
			"type":  "expression",
			"value": DumpExpr(n.X, fset),
		}
	}

	if n, ok := s.(*ast.LabeledStmt); ok {
		return map[string]interface{}{
			"kind":      "statement",
			"type":      "labeled",
			"label":     DumpIdent(n.Label, fset),
			"statement": DumpStmt(n.Stmt, fset),
			"position":  DumpPosition(fset.Position(n.Pos())),
		}
	}

	if n, ok := s.(*ast.BranchStmt); ok {
		result := map[string]interface{}{
			"kind":     "statement",
			"position": DumpPosition(fset.Position(n.Pos())),
		}

		switch n.Tok {
		case token.BREAK:
			result["type"] = "break"
			result["label"] = DumpIdent(n.Label, fset)

		case token.CONTINUE:
			result["type"] = "continue"
			result["label"] = DumpIdent(n.Label, fset)

		case token.GOTO:
			result["type"] = "goto"
			result["label"] = DumpIdent(n.Label, fset)

		case token.FALLTHROUGH:
			result["type"] = "fallthrough"

		}
		return result
	}

	if n, ok := s.(*ast.RangeStmt); ok {
		return map[string]interface{}{
			"kind":      "statement",
			"type":      "range",
			"key":       DumpExpr(n.Key, fset),
			"value":     DumpExpr(n.Value, fset),
			"target":    DumpExpr(n.X, fset),
			"is-assign": n.Tok == token.DEFINE,
			"body":      DumpBlock(n.Body, fset),
			"position":  DumpPosition(fset.Position(n.Pos())),
		}
	}
	if n, ok := s.(*ast.DeclStmt); ok {
		return map[string]interface{}{
			"kind":     "statement",
			"type":     "declaration",
			"target":   DumpDecl(n.Decl, fset),
			"position": DumpPosition(fset.Position(n.Pos())),
		}
	}

	if n, ok := s.(*ast.DeferStmt); ok {
		return map[string]interface{}{
			"kind":     "statement",
			"type":     "defer",
			"target":   DumpCall(n.Call, fset),
			"position": DumpPosition(fset.Position(n.Pos())),
		}
	}

	if n, ok := s.(*ast.IfStmt); ok {
		return map[string]interface{}{
			"kind":      "statement",
			"type":      "if",
			"init":      DumpStmt(n.Init, fset),
			"condition": DumpExpr(n.Cond, fset),
			"body":      DumpBlock(n.Body, fset),
			"else":      DumpStmt(n.Else, fset),
			"position":  DumpPosition(fset.Position(n.Pos())),
		}
	}

	if n, ok := s.(*ast.BlockStmt); ok {
		return DumpBlockAsStmt(n, fset)
	}

	if n, ok := s.(*ast.ForStmt); ok {
		return map[string]interface{}{
			"kind":      "statement",
			"type":      "for",
			"init":      DumpStmt(n.Init, fset),
			"condition": DumpExpr(n.Cond, fset),
			"post":      DumpStmt(n.Post, fset),
			"body":      DumpBlock(n.Body, fset),
			"position":  DumpPosition(fset.Position(n.Pos())),
		}
	}

	if n, ok := s.(*ast.GoStmt); ok {
		return map[string]interface{}{
			"kind":     "statement",
			"type":     "go",
			"target":   DumpCall(n.Call, fset),
			"position": DumpPosition(fset.Position(n.Pos())),
		}
	}

	if n, ok := s.(*ast.SendStmt); ok {
		return map[string]interface{}{
			"kind":     "statement",
			"type":     "send",
			"channel":  DumpExpr(n.Chan, fset),
			"value":    DumpExpr(n.Value, fset),
			"position": DumpPosition(fset.Position(n.Pos())),
		}
	}

	if n, ok := s.(*ast.SelectStmt); ok {
		return map[string]interface{}{
			"kind":     "statement",
			"type":     "select",
			"body":     DumpBlock(n.Body, fset),
			"position": DumpPosition(fset.Position(n.Pos())),
		}
	}

	if n, ok := s.(*ast.IncDecStmt); ok {
		return map[string]interface{}{
			"kind":      "statement",
			"type":      "crement",
			"target":    DumpExpr(n.X, fset),
			"operation": n.Tok.String(),
			"position":  DumpPosition(fset.Position(n.Pos())),
		}
	}

	if n, ok := s.(*ast.SwitchStmt); ok {
		return map[string]interface{}{
			"kind":      "statement",
			"type":      "switch",
			"init":      DumpStmt(n.Init, fset),
			"condition": DumpExpr(n.Tag, fset),
			"body":      DumpBlock(n.Body, fset),
			"position":  DumpPosition(fset.Position(n.Pos())),
		}
	}

	if n, ok := s.(*ast.TypeSwitchStmt); ok {
		return map[string]interface{}{
			"kind":     "statement",
			"type":     "type-switch",
			"init":     DumpStmt(n.Init, fset),
			"assign":   DumpStmt(n.Assign, fset),
			"body":     DumpBlock(n.Body, fset),
			"position": DumpPosition(fset.Position(n.Pos())),
		}
	}

	if n, ok := s.(*ast.CommClause); ok {
		stmts := make([]interface{}, len(n.Body))
		for i, v := range n.Body {
			stmts[i] = DumpStmt(v, fset)
		}

		return map[string]interface{}{
			"kind":      "statement",
			"type":      "select-clause",
			"statement": DumpStmt(n.Comm, fset),
			"body":      stmts,
			"position":  DumpPosition(fset.Position(n.Pos())),
		}

	}

	if n, ok := s.(*ast.CaseClause); ok {
		exprs := make([]interface{}, len(n.Body))
		for i, v := range n.Body {
			exprs[i] = DumpStmt(v, fset)
		}

		return map[string]interface{}{
			"kind":        "statement",
			"type":        "case-clause",
			"expressions": DumpExprs(n.List, fset),
			"body":        exprs,
			"position":    DumpPosition(fset.Position(n.Pos())),
		}
	}

	if n, ok := s.(*ast.BadStmt); ok {
		pos := fset.PositionFor(n.From, true).String()
		panic("Encountered BadStmt at " + pos + "; bailing out")
	}

	typ := reflect.TypeOf(s).String()
	pos := fset.PositionFor(s.Pos(), true).String()
	panic("Encountered unexpected " + typ + " node at " +
		pos + "while processing an statement; bailing out")
}

func DumpBlock(b *ast.BlockStmt, fset *token.FileSet) []interface{} {
	results := make([]interface{}, len(b.List))
	for i, v := range b.List {
		results[i] = DumpStmt(v, fset)
	}

	return results
}

func DumpBlockAsStmt(b *ast.BlockStmt, fset *token.FileSet) map[string]interface{} {
	return map[string]interface{}{
		"kind":     "statement",
		"type":     "block",
		"body":     DumpBlock(b, fset),
		"position": DumpPosition(fset.Position(b.Pos())),
	}
}

func DumpFuncDecl(f *ast.FuncDecl, fset *token.FileSet) map[string]interface{} {
	return map[string]interface{}{
		"kind":     "decl",
		"type":     "function",
		"name":     DumpIdent(f.Name, fset),
		"body":     DumpBlock(f.Body, fset),
		"params":   DumpFields(f.Type.Params, fset),
		"results":  DumpFields(f.Type.Results, fset),
		"comments": DumpCommentGroup(f.Doc, fset),
		"position": DumpPosition(fset.Position(f.Pos())),
	}
}

func DumpMethodDecl(f *ast.FuncDecl, fset *token.FileSet) map[string]interface{} {
	return map[string]interface{}{
		"kind":     "decl",
		"type":     "method",
		"receiver": DumpField(f.Recv.List[0], fset),
		"name":     DumpIdent(f.Name, fset),
		"body":     DumpBlock(f.Body, fset),
		"params":   DumpFields(f.Type.Params, fset),
		"results":  DumpFields(f.Type.Results, fset),
		"comments": DumpCommentGroup(f.Doc, fset),
		"position": DumpPosition(fset.Position(f.Pos())),
	}
}

func DumpDecl(n ast.Decl, fset *token.FileSet) map[string]interface{} {
	if decl, ok := n.(*ast.GenDecl); ok {
		return DumpGenDecl(decl, fset)
	}

	if decl, ok := n.(*ast.FuncDecl); ok {
		if decl.Recv == nil {
			return DumpFuncDecl(decl, fset)
		} else {
			return DumpMethodDecl(decl, fset)
		}
	}

	if decl, ok := n.(*ast.BadDecl); ok {
		pos := fset.PositionFor(decl.From, true).String()
		panic("Encountered BadDecl at " + pos + "; bailing out")
	}

	typ := reflect.TypeOf(n).String()
	pos := fset.PositionFor(n.Pos(), true).String()
	panic("Encountered unexpected " + typ + " node at " +
		pos + "while processing an expression; bailing out")
}

func IsImport(d ast.Decl) bool {
	if decl, ok := d.(*ast.GenDecl); ok {
		return decl.Tok == token.IMPORT
	}

	return false
}

func DumpFile(f *ast.File, fset *token.FileSet) ([]byte, error) {
	decls := []interface{}{}
	imps := []interface{}{}
	if f.Decls != nil {
		var ii int
		for ii = 0; ii < len(f.Decls); ii++ {
			if !IsImport(f.Decls[ii]) {
				break
			}
		}

		imports := f.Decls[0:ii]

		decls = make([]interface{}, len(f.Decls))
		for i, v := range f.Decls {
			decls[i] = DumpDecl(v, fset)
		}

		imps = make([]interface{}, len(imports))
		for i, v := range imports {
			imps[i] = DumpDecl(v, fset)
		}
	}

	allComments := make([][]string, len(f.Comments))
	for i, v := range f.Comments {
		allComments[i] = DumpCommentGroup(v, fset)
	}

	return json.Marshal(map[string]interface{}{
		"kind":         "file",
		"name":         DumpIdent(f.Name, fset),
		"comments":     DumpCommentGroup(f.Doc, fset),
		"all-comments": allComments,
		"declarations": decls,
		"imports":      imps,
	})
}

func TestExpr(s string) map[string]interface{} {
	fset := token.NewFileSet() // positions are relative to fset

	f, err := parser.ParseExpr(s)
	if err != nil {
		panic(err.Error())
	}

	// Inspect the AST and print all identifiers and literals.
	return DumpExpr(f, fset)
}

func TestFile(p string) []byte {
	fset := token.NewFileSet()

	file, err := os.Open(p)
	if err != nil {
		return nil
	}
	info, err := file.Stat()
	if err != nil {
		return nil
	}

	size := info.Size()
	file.Close()

	fset.AddFile(p, -1, int(size))

	f, err := parser.ParseFile(fset, p, nil, 0)

	if err != nil {
		panic(err.Error())
	}

	// Inspect the AST and print all identifiers and literals.
	res, err := DumpFile(f, fset)

	if err != nil {
		panic(err.Error())
	}

	return res
}

func TestStmt(s string) []byte {
	fset := token.NewFileSet() // positions are relative to fset

	f, err := parser.ParseFile(fset, "stdin", "package p; func blah(foo int, bar float64) string { "+s+"}", 0)
	if err != nil {
		panic(err.Error())
	}

	// Inspect the AST and print all identifiers and literals.
	res, err := DumpFile(f, fset)

	if err != nil {
		panic(err.Error())
	}

	return res
}
