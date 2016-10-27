package main

import ("github.com/Nerabus/goblin"
        "go/token"
	"go/parser"
        "encoding/json"
        "os")

func main() {

	// Create the AST by parsing src.
	fset := token.NewFileSet() // positions are relative to fset

	if len(os.Args) != 3 {
		println("Usage: ./goblin [--expr EXPR] [--file FILE]")
		return
	}

	if os.Args[1] == "--expr" {
		val, _ := json.Marshal(goblin.TestExpr(os.Args[2]))
		os.Stdout.Write(val)
	} else if os.Args[1] == "--stmt" {
		val := goblin.TestStmt(os.Args[2])
		os.Stdout.Write(val)
	} else if os.Args[1] == "--file" {
		f, err := parser.ParseFile(fset, os.Args[2], nil, 0)
		if err != nil {
			panic(err)
		}

		// Inspect the AST and print all identifiers and literals.
		val, _ := goblin.DumpFile(f, fset)
		os.Stdout.Write(val)
	}

}
