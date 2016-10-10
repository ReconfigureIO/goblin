package main

import (
	"go/ast"
	"go/parser"
	"go/token"
	"encoding/json"
	"os"
	"strings"
)

// this file is like a paean to the problems with imperative languages

type ASTDump interface {
	Dump(interface{}, *token.FileSet) ([]byte, error)
}

func DumpIdent(i *ast.Ident, fset *token.FileSet) map[string]string {
	if i == nil {
		return nil
	}

	return map[string]string {
		"kind": "ident",
		"value": i.Name,
	}
}

func DumpArray(a *ast.ArrayType, fset *token.FileSet) interface{} {
	return map[string]interface{} {
		"kind": "array-type",
		"length": DumpExpr(a.Len, fset),
		"element": DumpExpr(a.Elt, fset),
	}
}

func DumpExpr(n ast.Expr, fset *token.FileSet) interface{} {
	if n, ok := n.(*ast.Ident); ok {
		return DumpIdent(n, fset)
	}

	if n, ok := n.(*ast.ArrayType); ok {
		return DumpArray(n, fset)
	}

	if n, ok := n.(*ast.BasicLit); ok {
		return DumpBasicLit(n, fset)
	}

	if n, ok := n.(*ast.BinaryExpr); ok {
		return DumpBinaryExpr(n, fset)
	}

	return nil
}

func DumpBinaryExpr(b *ast.BinaryExpr, fset *token.FileSet) map[string]interface{} {
	return map[string]interface{} {
		"kind": "binary",
		"left": DumpExpr(b.X, fset),
		"right": DumpExpr(b.Y, fset),
		"operation": b.Op.String(),
	}
}

func DumpBasicLit(l *ast.BasicLit, fset *token.FileSet) map[string]string {
	return map[string]string {
		"kind": "literal",
		"token-kind": l.Kind.String(),
		"value": l.Value,
	}
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

	if res, ok := t.Type.(*ast.Ident); ok {
		contained = DumpIdent(res, fset)
	} else if res, ok := t.Type.(*ast.ArrayType); ok {
		contained = DumpArray(res, fset)
	}

	return map[string]interface{} {
		"kind": "type",
		"name": DumpIdent(t.Name, fset),
		"contained": contained,
		"comments": DumpCommentGroup(t.Comment, fset),
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
		"type": DumpExpr(spec.Type, fset),
		"values": processedValues,
		"comments": DumpCommentGroup(spec.Doc, fset),
	}
}

func DumpDecl(n interface{}, fset *token.FileSet) interface{} {
 	if decl, ok := n.(*ast.GenDecl); ok {
		results := make([]map[string]interface{}, len(decl.Specs))
//		print(decl.Tok.String())
		switch decl.Tok {
		case token.IMPORT:
			for i, v := range decl.Specs {
				results[i] = DumpImport(v.(*ast.ImportSpec), fset)
			}
			return results

		case token.TYPE:
			for i, v := range decl.Specs {
				results[i] = DumpType(v.(*ast.TypeSpec), fset)
			}
			return results

		case token.CONST:
			for i, v := range decl.Specs {
				results[i] = DumpValue("const", v.(*ast.ValueSpec), fset)
			}
			return results

		case token.VAR:
			for i, v := range decl.Specs {
				results[i] = DumpValue("var", v.(*ast.ValueSpec), fset)
			}
			return results

		}
	}

	return nil
}

func DumpFile(f *ast.File, fset *token.FileSet) ([]byte, error) {
	decls := []interface{} {}
	if f.Decls != nil {
		decls = make([]interface{}, len(f.Decls))
		for i, v := range f.Decls {
			decls[i] = DumpDecl(v, fset)
		}
	}

	// if f.Unresolved != nil {
	// 	os.Stderr.Write([]byte("Warning: unresolved identifiers present in file"))
	// }

	return json.Marshal(map[string]interface{} {
		"kind": "file",
		"name": DumpIdent(f.Name, fset),
		"comments": DumpCommentGroup(f.Doc, fset),
		"declarations": decls,
	})
}

func main() {
	// src is the input for which we want to inspect the AST.
	src := `
package p
import "go/ast"
type MyArray [16]int8
const c = 1.0
var X = f(3.14)*2 + c
`

	// Create the AST by parsing src.
	fset := token.NewFileSet() // positions are relative to fset
	f, err := parser.ParseFile(fset, "src.go", src, 0)
	if err != nil {
		panic(err)
	}

	// Inspect the AST and print all identifiers and literals.
	val, _ := DumpFile(f, fset)
	os.Stdout.Write(val)

	// ast.Print(fset, f)

}
