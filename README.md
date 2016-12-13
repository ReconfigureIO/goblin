Goblin
======

[![Build Status](https://travis-ci.org/ReconfigureIO/goblin.svg?branch=master)](https://travis-ci.org/ReconfigureIO/goblin)
[![codecov.io](https://codecov.io/github/ReconfigureIO/goblin/branch/master/graph/badge.svg)](https://codecov.io/github/ReconfigureIO/goblin)

`goblin` is an executable that uses Go's `ast`, `parser`, and `token` modules to dump a Go expression, statement, or file to JSON. It is small, fast, self-contained, and incurs no dependencies.

## Usage

`goblin --file [FILENAME]` dumps a given file.
`goblin --expr EXPR` dumps an expression.
`goblin --stmt STMT` dumps a statement—due to a quirk in the Go AST API, this statement will be surrounded by a dummy function.

## Format

Every node is a JSON object containing at least two guaranteed keys:

* `kind` (string): this corresponds to the data type of the given node. Expressions (`Prim` and `Expr`) are `"expression"`, statements (`Statement` and `Simp`) are `"statement"`, binary and unary expressions are `"unary"` and `"binary"` respectively.
* `type` (string): this corresponds to the data constructor associated with the node. Casts have kind `"expression""` and type `"cast"`. Floats have kind `"literal"` and type `"FLOAT"`. Pointer types have kind `"type"` and type `"pointer"`.

I apologize for the semantic overlap associated with the vagueness of the words "kind" and "type". Suggestions as to better nomenclature are welcomed.

## FAQ's

**Why not use the `ast.Visitor` interface instead of recursing manually into every node?** Because `Visitor` is inherently side-effectual: it declares no return type, so it is not possible to use it to express an algebra (which is all this program really is).

## Licensing

`goblin` is open-source software © Reconfigure.io, released to the public under the terms of the Apache 2.0 license. A copy can be found under the LICENSE file in the project root.

## Contributing

Pull requests are enthusiastically accepted!

By participating in this project you agree to follow the [Contributor Code of Conduct][coc].

## TODO

* Use JSON Schema to ensure well-formedness of the AST.
* Pull in github.com/stretchr/testify for assertions and glog for logging.

## Known Issues

* The built-in `make` and `new` functions can be shadowed. Since goblin expects `make` and `new` to take types as arguments, it will reject a shadowing as a syntax error. The chances of this happening in real code are pretty low, as shadowing built-in functions is discouraged in real-world code.

[coc]: http://contributor-covenant.org/version/1/4/
