package main

import (
	"encoding/json"
	"flag"
	"github.com/ReconfigureIO/goblin"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
)

// Assuming you build with `make`, this variable will be filled in automatically
// (leaning on -ldflags -X).
var version string = "unspecified"

func main() {
	versionFlag := flag.Bool("v", false, "display goblin version")
	builtinDumpFlag := flag.Bool("builtin-dump", false, "use go/ast to dump the file, not JSON")
	panicFlag := flag.Bool("panic", false, "use panic() rather than JSON on error conditions")
	fileFlag := flag.String("file", "", "file to parse")
	stmtFlag := flag.String("stmt", "", "statement to parse")
	exprFlag := flag.String("expr", "", "expression to parse")

	flag.Parse()
	// Create the AST by parsing src.
	fset := token.NewFileSet() // positions are relative to fset

	if *panicFlag {
		goblin.ShouldPanic = true
	}

	if *versionFlag {
		println(version)
		return
	} else if *fileFlag != "" {
		file, err := os.Open(*fileFlag)
		if err != nil {
			goblin.Perish(goblin.TOPLEVEL_POSITION, "path_error", err.Error())
		}
		info, err := file.Stat()
		if err != nil {
			goblin.Perish(goblin.TOPLEVEL_POSITION, "path_error", err.Error())
		}

		size := info.Size()
		file.Close()

		fset.AddFile(*fileFlag, -1, int(size))

		f, err := parser.ParseFile(fset, *fileFlag, nil, parser.ParseComments)
		if err != nil {
			goblin.Perish(goblin.INVALID_POSITION, "positionless_syntax_error", err.Error())
		}

		if *builtinDumpFlag {
			ast.Print(fset, f)
		} else {
			val, _ := goblin.DumpFile(f, fset)
			os.Stdout.Write(val)
		}
	} else if *exprFlag != "" {
		val, _ := json.Marshal(goblin.TestExpr(*exprFlag))
		os.Stdout.Write(val)
	} else if *stmtFlag != "" {
		val := goblin.TestStmt(*stmtFlag)
		os.Stdout.Write(val)
	} else {
		flag.PrintDefaults()
	}
}
