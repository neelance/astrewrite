package astutil

import (
	"bytes"
	"go/ast"
	"go/parser"
	"go/printer"
	"go/token"
	"testing"
)

func TestSimplify(t *testing.T) {
	simplifyAndCompare(t, "-a", "-a")
	simplifyAndCompare(t, "-a()", "_1 := a(); -_1")
	simplifyAndCompare(t, "a() + b()", "_1 := a(); _2 := b(); _1 + _2")
	simplifyAndCompare(t, "a() && b()", "_1 := a(); if(_1) { _1 = b() }; _1")
	simplifyAndCompare(t, "a() || b()", "_1 := a(); if(!_1) { _1 = b() }; _1")
	simplifyAndCompare(t, "a() && (b() || c)", "_1 := a(); if(_1) { _2 := b(); if(!_2) { _2 = c }; _1 = (_2) }; _1")
	simplifyAndCompare(t, "f(g(), h())", "_1 := g(); _2 := h(); f(_1, _2)")
	simplifyAndCompare(t, "f().x", "_1 := f(); _1.x")
	simplifyAndCompare(t, "f()()", "_1 := f(); _1()")
	simplifyAndCompare(t, "x.f()", "x.f()")
	simplifyAndCompare(t, "f()[g()]", "_1 := f(); _2 := g(); _1[_2]")
	simplifyAndCompare(t, "f()[g():h()]", "_1 := f(); _2 := g(); _3 := h(); _1[_2:_3]")
	simplifyAndCompare(t, "f()[g():h():i()]", "_1 := f(); _2 := g(); _3 := h(); _4 := i(); _1[_2:_3:_4]")
	simplifyAndCompare(t, "*f()", "_1 := f(); *_1")
	simplifyAndCompare(t, "f().(t)", "_1 := f(); _1.(t)")
	simplifyAndCompare(t, "func() { -a() }", "func() { _1 := a(); -_1 }")
	simplifyAndCompare(t, "T{a(), b()}", "_1 := a(); _2 := b(); T{_1, _2}")
	simplifyAndCompare(t, "T{A: a(), B: b()}", "_1 := a(); _2 := b(); T{A: _1, B: _2}")
}

func simplifyAndCompare(t *testing.T, in, out string) {
	fset := token.NewFileSet()

	expected := fprint(t, fset, parse(t, fset, out))

	file := parse(t, fset, in)
	f := file.Decls[0].(*ast.FuncDecl)
	f.Body.List = Simplify(f.Body.List)
	got := fprint(t, fset, file)

	if got != expected {
		t.Errorf("\n--- expected:\n%s--- got:\n%s", expected, got)
	}
}

func parse(t *testing.T, fset *token.FileSet, body string) *ast.File {
	file, err := parser.ParseFile(fset, "", "package main; func main() { "+body+" }", 0)
	if err != nil {
		t.Fatal(err)
	}
	return file
}

func fprint(t *testing.T, fset *token.FileSet, file *ast.File) string {
	var buf bytes.Buffer
	if err := printer.Fprint(&buf, fset, file); err != nil {
		t.Fatal(err)
	}
	return buf.String()
}
