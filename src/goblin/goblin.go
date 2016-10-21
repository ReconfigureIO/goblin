package goblin

import (
	"go/ast"
	"go/parser"
	"go/token"
	"encoding/json"
	"strings"
	"reflect"
)

// this file is like a paean to the problems with imperative languages

func DumpIdent(i *ast.Ident, fset *token.FileSet) map[string]interface{} {
	if i == nil {
		return nil
	}

	asLiteral := map[string]interface{} {
		"kind": "literal",
		"type": "BOOL",
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

	return map[string]interface{} {
		"kind": "ident",
		"value": i.Name,
	}
}

func DumpArray(a *ast.ArrayType, fset *token.FileSet) map[string]interface{} {
	return map[string]interface{} {
		"kind": "array",
		"length": DumpExpr(a.Len, fset),
		"element": DumpExprAsType(a.Elt, fset),
	}
}

// This is a weird hack to work around the fact that a ton of Type nodes have an Expr
// rather than a Type as their associated, well, type
func DumpExprAsType(e ast.Expr, fset *token.FileSet) map[string]interface{} {
	var contained interface{} = nil
	var typ string = ""

	if e == nil {
		return nil
	}

	if n, ok := e.(*ast.Ident); ok {
		contained = DumpIdent(n, fset)
		typ = "type-name"
	}

	if n, ok := e.(*ast.ArrayType); ok {
		contained = DumpArray(n, fset)
		typ = "array"
	}

	if n, ok := e.(*ast.StarExpr); ok {
		contained = DumpExprAsType(n.X, fset)
		typ = "pointer"
	}

	if n, ok := e.(*ast.MapType); ok {
		contained = map[string]interface{} {
			"key": DumpExprAsType(n.Key, fset),
			"value": DumpExprAsType(n.Value, fset),
		}
		typ = "map"
	}

	if n, ok := e.(*ast.ChanType); ok {
		contained = map[string]interface{} {
			"direction": DumpChanDir(n.Dir),
			"value": DumpExprAsType(n.Value, fset),
		}
		typ = "chan"
	}

	if n, ok := e.(*ast.StructType); ok {
		contained = DumpFields(n.Fields, fset)
		typ = "struct"
	}

	if typ == "" {
		gotten := reflect.TypeOf(e).String()
		pos := fset.PositionFor(e.Pos(), true).String()
		panic("Unrecognized type " + gotten + " in expr-as-type at " + pos)
	}

	return map[string]interface{} {
		"kind": "type",
		"type": typ,
		"contained": contained,
	}
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
		return map[string]interface{} {
			"kind": "expression",
			"type": "identifier",
			"value": DumpIdent(n, fset),
		}
	}

	if n, ok := e.(*ast.Ellipsis); ok {
		return map[string]interface{} {
			"kind": "ellipsis",
			"value": DumpExpr(n.Elt, fset),
		}
	}

	// is this the right place??
	if n, ok := e.(*ast.FuncLit); ok {
		return map[string]interface{} {
			"kind": "literal",
			"type": "function",
			"params": DumpFields(n.Type.Params, fset),
			"results": DumpFields(n.Type.Results, fset),
		}
	}

	if n, ok := e.(*ast.BasicLit); ok {
		return DumpBasicLit(n, fset)
	}

	if n, ok := e.(*ast.CompositeLit); ok {
		return map[string]interface{} {
			"kind": "literal",
			"type": "composite",
			"declared": DumpExprAsType(n.Type, fset),
			"values": DumpExprs(n.Elts, fset),
		}
	}

	if n, ok := e.(*ast.BinaryExpr); ok {
		return DumpBinaryExpr(n, fset)
	}

	if n, ok := e.(*ast.IndexExpr); ok {
		return map[string]interface{} {
			"kind": "expression",
			"type": "index",
			"target": DumpExpr(n.X, fset),
			"index": DumpExpr(n.Index, fset),
		}
	}

	if n, ok := e.(*ast.StarExpr); ok {
		return map[string]interface{} {
			"kind": "expression",
			"type": "star",
			"target": DumpExpr(n.X, fset),
		}
	}

	if n, ok := e.(*ast.CallExpr); ok {

		return DumpCall(n, fset)
	}

	if n, ok := e.(*ast.ParenExpr); ok {
		return map[string]interface{} {
			"kind": "expression",
			"type": "paren",
			"target": DumpExpr(n.X, fset),
		}
	}

	if n, ok := e.(*ast.SelectorExpr); ok {
		return map[string]interface{} {
			"kind": "expression",
			"type": "selector",
			"target": DumpExpr(n.X, fset),
			"field": DumpIdent(n.Sel, fset),
		}
	}

	if n, ok := e.(*ast.TypeAssertExpr); ok {
		return map[string]interface{} {
			"kind": "expression",
			"type": "type-assert",
			"target": DumpExpr(n.X, fset),
			"asserted": DumpExprAsType(n.Type, fset),
		}
	}

	if n, ok := e.(*ast.UnaryExpr); ok {
		return map[string]interface{} {
			"kind": "unary",
			"target": DumpExpr(n.X, fset),
			"operator": n.Op.String(),
		}
	}

	if n, ok := e.(*ast.SliceExpr); ok {
		return map[string]interface{} {
			"kind": "expression",
			"type": "slice",
			"target": DumpExpr(n.X, fset),
			"low": DumpExpr(n.Low, fset),
			"high": DumpExpr(n.High, fset),
			"max": DumpExpr(n.Max, fset),
			"three": n.Slice3,
		}
	}

	if n, ok := e.(*ast.KeyValueExpr); ok {
		return map[string]interface{} {
			"kind": "expression",
			"type": "key-value",
			"key": DumpExpr(n.Key, fset),
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
	return map[string]interface{} {
		"kind": "binary",
		"left": DumpExpr(b.X, fset),
		"right": DumpExpr(b.Y, fset),
		"operator": b.Op.String(),
	}
}

func DumpBasicLit(l *ast.BasicLit, fset *token.FileSet) map[string]interface{} {
	if l == nil {
		return nil
	}

	return map[string]interface{} {
		"kind": "literal",
		"type": l.Kind.String(),
		"value": l.Value,
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


	return map[string]interface{} {
		"kind": "field",
		"names": names,
		"declared-type": DumpExpr(f.Type, fset),
		"tag": DumpBasicLit(f.Tag, fset),
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
		return nil
	}

	result := make([]string, len(g.List))
	for i, v := range g.List {
		result[i] = v.Text
	}

	return result
}

func DumpType(t *ast.TypeSpec, fset *token.FileSet) map[string]interface{} {
	var contained interface{} = nil
	var typ string = ""

	if res, ok := t.Type.(*ast.Ident); ok {
		typ = "type-name"
		contained = DumpIdent(res, fset)
	}

	if res, ok := t.Type.(*ast.ArrayType); ok {
		typ = "array"
		contained = DumpArray(res, fset)
	}

	if res, ok := t.Type.(*ast.MapType); ok {
		typ = "map"
		contained = map[string]interface{} {
			"key": DumpExpr(res.Key, fset),
			"value": DumpExpr(res.Value, fset),
		}
	}

	if res, ok := t.Type.(*ast.InterfaceType); ok {
		typ = "interface"
		contained = map[string]interface{} {
			"methods": DumpFields(res.Methods, fset),
			"incomplete": res.Incomplete,
		}
	}

	if res, ok := t.Type.(*ast.ChanType); ok {
		typ = "chan"
		contained = map[string]interface{} {
			"direction": res.Dir,
			"value": DumpExpr(res.Value, fset),
		}
	}

	if typ == "" {
		pos := fset.PositionFor(t.Pos(), true).String()
		panic("Unrecognized Type " + t.Name.Name + " in Type at " + pos)
	}

	return map[string]interface{} {
		"kind": "type",
		"type": typ,
		"name": DumpIdent(t.Name, fset),
		"value": contained,
		"comments": DumpCommentGroup(t.Comment, fset),
	}
}

func DumpCall(c *ast.CallExpr, fset *token.FileSet) map[string]interface{} {
	if callee, ok := c.Fun.(*ast.Ident); ok {
		if callee.Name == "new" {
			return map[string]interface{} {
				"kind": "expression",
				"type": "new",
				"argument": DumpExprAsType(c.Args[0], fset),
			}
		}

		if callee.Name == "make" {
			return map[string]interface{} {
				"kind": "expression",
				"type": "make",
				"argument": DumpExprAsType(c.Args[0], fset),
				"rest": DumpExprs(c.Args[1:], fset),
			}
		}
	}

	callee := DumpExpr(c.Fun, fset)

	if callee["kind"].(string) == "type" {
		return map[string]interface{} {
			"kind": "expression",
			"type": "cast",
			"target": DumpExpr(c.Args[0], fset),
			"coerced-to": callee,
		}
	}

	return map[string]interface{} {
		"kind": "expression",
		"type": "call",
		"function": callee,
		"arguments": DumpExprs(c.Args, fset),
		"ellipsis": c.Ellipsis != token.NoPos,
	}
}

func DumpImport(spec *ast.ImportSpec, fset *token.FileSet) map[string]interface{} {
	res := map[string]interface{} {
		"type": "import",
		"comments": DumpCommentGroup(spec.Doc, fset),
		"name": DumpIdent(spec.Name, fset),
		"path": strings.Trim(spec.Path.Value, "\""),
	}

	return res
}

func DumpValue(kind string, spec *ast.ValueSpec, fset *token.FileSet) map[string]interface{} {
	givenValues := []ast.Expr {}
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

	return map[string]interface{} {
		"kind": kind,
		"names": processedNames,
		"type": DumpExprAsType(spec.Type, fset),
		"values": processedValues,
		"comments": DumpCommentGroup(spec.Doc, fset),
	}
}

func DumpGenDecl(decl *ast.GenDecl, fset *token.FileSet) []map[string]interface{} {
	results := make([]map[string]interface{}, len(decl.Specs))
	switch decl.Tok {
	case token.IMPORT:
		for i, v := range decl.Specs {
			results[i] = DumpImport(v.(*ast.ImportSpec), fset)
		}

	case token.TYPE:
		for i, v := range decl.Specs {
			results[i] = DumpType(v.(*ast.TypeSpec), fset)
		}

	case token.CONST:
		for i, v := range decl.Specs {
			results[i] = DumpValue("const", v.(*ast.ValueSpec), fset)
		}

	case token.VAR:
		for i, v := range decl.Specs {
			results[i] = DumpValue("var", v.(*ast.ValueSpec), fset)
		}
	default:
		pos := fset.PositionFor(decl.Pos(), true).String()
		panic("Unrecognized token " + decl.Tok.String() + " in GenDecl at " + pos)
	}


	return results
}

func DumpStmt(s ast.Stmt, fset *token.FileSet) interface{} {
	if s == nil {
		return nil
	}

	if n, ok := s.(*ast.ReturnStmt); ok {
		return map[string]interface{} {
			"kind": "statement",
			"type": "return",
			"values": DumpExprs(n.Results, fset),
		}
	}

	if n, ok := s.(*ast.AssignStmt); ok {
		if n.Tok == token.ASSIGN {
			return map[string]interface{} {
				"kind": "statement",
				"type": "assign",
				"left": DumpExprs(n.Lhs, fset),
				"right": DumpExprs(n.Rhs, fset),
			}

		} else if n.Tok == token.DEFINE {
			return map[string]interface{} {
				"kind": "statement",
				"type": "define",
				"left": DumpExprs(n.Lhs, fset),
				"right": DumpExprs(n.Rhs, fset),
			}
	        }  else {
			tok := n.Tok.String()
			return map[string]interface{} {
				"kind": "statement",
				"type": "assign-operator",
				"operator": tok[0:len(tok)-1],
				"left": DumpExprs(n.Lhs, fset),
				"right": DumpExprs(n.Rhs, fset),
			}
		}

	}

	if _, ok := s.(*ast.EmptyStmt); ok {
		return map[string]interface{} {
			"kind": "statement",
			"type": "empty",
		}
	}

	if n, ok := s.(*ast.ExprStmt); ok {
		return map[string]interface{} {
			"kind": "statement",
			"type": "expression",
			"value": DumpExpr(n.X, fset),
		}
	}

	if n, ok := s.(*ast.LabeledStmt); ok {
		return map[string]interface{} {
			"kind": "statement",
			"type": "labeled",
			"label": DumpIdent(n.Label, fset),
			"statement": DumpStmt(n.Stmt, fset),
		}
	}

	if n, ok := s.(*ast.BranchStmt); ok {
		result := map[string]interface{} {
			"kind": "statement",
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
		return map[string]interface{} {
			"kind": "statement",
			"type": "range",
			"key": DumpExpr(n.Key, fset),
			"value": DumpExpr(n.Value, fset),
			"target": DumpExpr(n.X, fset),
			"body": DumpBlock(n.Body, fset),
		}
	}

	if n, ok := s.(*ast.DeclStmt); ok {
		return map[string]interface{} {
			"kind": "statement",
			"type": "declaration",
			"target": DumpDecl(n.Decl, fset),
		}
	}

	if n, ok := s.(*ast.DeferStmt); ok {
		return map[string]interface{} {
			"kind": "statement",
			"type": "defer",
			"target": DumpCall(n.Call, fset),
		}
	}

	if n, ok := s.(*ast.IfStmt); ok {
		return map[string]interface{} {
			"kind": "statement",
			"type": "if",
			"init": DumpStmt(n.Init, fset),
			"condition": DumpExpr(n.Cond, fset),
			"body": DumpBlock(n.Body, fset),
			"else": DumpStmt(n.Else, fset),
		}
	}

	if n, ok := s.(*ast.BlockStmt); ok {
		return DumpBlock(n, fset)
	}

	if n, ok := s.(*ast.ForStmt); ok {
		return map[string]interface{} {
			"kind": "statement",
			"type": "for",
			"init": DumpStmt(n.Init, fset),
			"condition": DumpExpr(n.Cond, fset),
			"post": DumpStmt(n.Post, fset),
			"body": DumpBlock(n.Body, fset),
		}
	}

	if n, ok := s.(*ast.GoStmt); ok {
		return map[string]interface{} {
			"kind": "statement",
			"type": "go",
			"target": DumpCall(n.Call, fset),
		}
	}

	if n, ok := s.(*ast.SendStmt); ok {
		return map[string]interface{} {
			"kind": "statement",
			"type": "send",
			"channel": DumpExpr(n.Chan, fset),
			"value": DumpExpr(n.Value, fset),
		}
	}

	if n, ok := s.(*ast.SelectStmt); ok {
		return map[string]interface{} {
			"kind": "statement",
			"type": "select",
			"body": DumpBlock(n.Body, fset),
		}
	}

	if n, ok := s.(*ast.IncDecStmt); ok {
		return map[string]interface{} {
			"kind": "statement",
			"type": "crement",
			"target": DumpExpr(n.X, fset),
			"operation": n.Tok.String(),
		}
	}

	if n, ok := s.(*ast.TypeSwitchStmt); ok {
		return map[string]interface{} {
			"kind": "statement",
			"type": "type-switch",
			"init": DumpStmt(n.Init, fset),
			"assign": DumpStmt(n.Assign, fset),
			"body": DumpBlock(n.Body, fset),
		}
	}

	if n, ok := s.(*ast.CaseClause); ok {
		exprs := make([]interface{}, len(n.Body))
		for i, v := range n.Body {
			exprs[i] = DumpStmt(v, fset)
		}

		return map[string]interface{} {
			"kind": "statement",
			"type": "case-clause",
			"expressions": DumpExprs(n.List, fset),
			"body": exprs,
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

func DumpFuncDecl(f *ast.FuncDecl, fset *token.FileSet) map[string]interface{} {
	return map[string]interface{} {
		"kind": "decl",
		"type": "function",
		"name": DumpIdent(f.Name, fset),
		"body": DumpBlock(f.Body, fset),
		"params": DumpFields(f.Type.Params, fset),
		"results": DumpFields(f.Type.Results, fset),
	}
}

func DumpDecl(n ast.Decl, fset *token.FileSet) interface{} {
 	if decl, ok := n.(*ast.GenDecl); ok {
		return DumpGenDecl(decl, fset)
	}

	if decl, ok := n.(*ast.FuncDecl); ok {
		return DumpFuncDecl(decl, fset)
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

func DumpFile(f *ast.File, fset *token.FileSet) ([]byte, error) {
	decls := []interface{} {}
	if f.Decls != nil {
		decls = make([]interface{}, len(f.Decls))
		for i, v := range f.Decls {
			decls[i] = DumpDecl(v, fset)
		}
	}

	return json.Marshal(map[string]interface{} {
		"kind": "file",
		"name": DumpIdent(f.Name, fset),
		"comments": DumpCommentGroup(f.Doc, fset),
		"declarations": decls,
	})
}

func TestExpr(s string) map[string]interface{} {
	fset := token.NewFileSet() // positions are relative to fset

	f, err := parser.ParseExpr(s)
	if (err != nil) {
		panic(err.Error())
	}

	// Inspect the AST and print all identifiers and literals.
	return DumpExpr(f, fset)
}

func TestStmt(s string) []byte {
	fset := token.NewFileSet() // positions are relative to fset

	f, err := parser.ParseFile(fset, "stdin", "package p; func blah() { " + s + "}", 0)
	if (err != nil) {
		panic(err.Error())
	}

	// Inspect the AST and print all identifiers and literals.
	res, err := DumpFile(f, fset)

	if (err != nil) {
		panic(err.Error())
	}

	return res
}
