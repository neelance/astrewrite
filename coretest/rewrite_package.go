package main

import (
	"go/ast"
	"go/build"
	"go/importer"
	"go/parser"
	"go/printer"
	"go/token"
	"go/types"
	"os"
	"path/filepath"

	"github.com/neelance/astrewrite"
)

func main() {
	importPath := os.Args[1]

	pkg, err := build.Import(importPath, "", 0)
	if err != nil {
		panic(err)
	}

	fset := token.NewFileSet()
	files := make([]*ast.File, len(pkg.GoFiles))
	for i, name := range pkg.GoFiles {
		file, err := parser.ParseFile(fset, filepath.Join(pkg.Dir, name), nil, parser.ParseComments)
		if err != nil {
			panic(err)
		}
		files[i] = file
	}

	typesInfo := &types.Info{
		Types: make(map[ast.Expr]types.TypeAndValue),
	}
	config := &types.Config{
		Importer: importer.Default(),
	}
	if _, err := config.Check(importPath, fset, files, typesInfo); err != nil {
		panic(err)
	}

	for i, file := range files {
		for _, decl := range file.Decls {
			if f, ok := decl.(*ast.FuncDecl); ok && f.Body != nil {
				f.Body.List = astrewrite.Simplify(f.Body.List, typesInfo, false)
			}
		}
		out, err := os.Create(filepath.Join("goroot", "src", importPath, pkg.GoFiles[i]))
		if err != nil {
			panic(err)
		}
		if err := printer.Fprint(out, fset, file); err != nil {
			panic(err)
		}
	}
}
