package main

import ("encoding/json"
	"flag"
	"github.com/Reconfigureio/goblin"
	"go/parser"
        "go/token"
        "os")

// Assuming you build with `make`, this variable will be filled in automatically
// (leaning on -ldflags -X).
var version string = "unspecified"

func main() {
	versionFlag := flag.Bool("v", false, "display goblin version")
	fileFlag    := flag.String("file", "", "file to parse")
	stmtFlag    := flag.String("stmt", "", "statement to parse")
	exprFlag    := flag.String("expr", "", "expression to parse")

	flag.Parse()
	// Create the AST by parsing src.
	fset := token.NewFileSet() // positions are relative to fset

	if (*versionFlag) {
		println(version)
		return
	} else if *fileFlag != "" {
		f, err := parser.ParseFile(fset, *fileFlag, nil, 0)
		if err != nil {
			panic(err)
		}

		// Inspect the AST and print all identifiers and literals.
		val, _ := goblin.DumpFile(f, fset)
		os.Stdout.Write(val)
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
