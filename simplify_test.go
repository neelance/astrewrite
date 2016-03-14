package astrewrite

import (
	"bytes"
	"go/ast"
	"go/parser"
	"go/printer"
	"go/token"
	"go/types"
	"testing"
)

func TestSimplify(t *testing.T) {
	simplifyAndCompare(t, "-a()", "_1 := a(); -_1")
	simplifyAndCompare(t, "a() + b()", "_1 := a(); _2 := b(); _1 + _2")
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
	simplifyAndCompare(t, "func() { a()() }", "func() { _1 := a(); _1() }")

	simplifyAndCompare(t, "a() && b", "_1 := a(); _1 && b")
	simplifyAndCompare(t, "a && b()", "_1 := a; if _1 { _1 = b() }; _1")
	simplifyAndCompare(t, "a() && b()", "_1 := a(); if _1 { _1 = b() }; _1")

	simplifyAndCompare(t, "a() || b", "_1 := a(); _1 || b")
	simplifyAndCompare(t, "a || b()", "_1 := a; if !_1 { _1 = b() }; _1")
	simplifyAndCompare(t, "a() || b()", "_1 := a(); if !_1 { _1 = b() }; _1")

	simplifyAndCompare(t, "a && (b || c())", "_1 := a; if(_1) { _2 := b; if(!_2) { _2 = c() }; _1 = (_2) }; _1")

	simplifyAndCompare(t, "if a() { b }", "_1 := a(); if _1 { b }")
	simplifyAndCompare(t, "if a := b(); a { c }", "{ a := b(); if a { c } }")
	simplifyAndCompare(t, "if a { b()() }", "if a { _1 := b(); _1() }")
	simplifyAndCompare(t, "if a { b } else { c()() }", "if a { b } else { _1 := c(); _1() }")
	simplifyAndCompare(t, "if a { b } else if c { d()() }", "if a { b } else if c { _1 := d(); _1() }")
	simplifyAndCompare(t, "if a { b } else if c() { d }", "if a { b } else { _1 := c(); if _1 { d } }")
	simplifyAndCompare(t, "if a { b } else if c := d(); c { e }", "if a { b } else { c := d(); if c { e } }")

	simplifyAndCompare(t, "l: switch a { case b, c: d()() }", "l: switch { default: _1 := a; if _1 == (b) || _1 == (c) { _2 := d(); _2() } }")
	simplifyAndCompare(t, "switch a() { case b: c }", "switch { default: _1 := a(); if _1 == (b) { c } }")
	simplifyAndCompare(t, "switch x := a(); x { case b, c: d }", "switch { default: x := a(); _1 := x; if _1 == (b) || _1 == (c) { d } }")
	simplifyAndCompare(t, "switch a() { case b: c; default: e; case c: d }", "switch { default: _1 := a(); if _1 == (b) { c } else if _1 == (c) { d } else { e } }")
	simplifyAndCompare(t, "switch a { case b(): c }", "switch { default: _1 := a; _2 := b(); if _1 == (_2) { c } }")
	simplifyAndCompare(t, "switch a { default: d; fallthrough; case b: c }", "switch { default: _1 := a; if _1 == (b) { c } else { d; c } }")

	simplifyAndCompare(t, "switch a().(type) { case b, c: d }", "_1 := a(); switch _1.(type) { case b, c: d }")
	simplifyAndCompare(t, "switch x := a(); x.(type) { case b: c }", "{ x := a(); switch x.(type) { case b: c } }")
	simplifyAndCompare(t, "switch a := b().(type) { case c: d }", "_1 := b(); switch a := _1.(type) { case c: d }")
	simplifyAndCompare(t, "switch a.(type) { case b, c: d()() }", "switch a.(type) { case b, c: _1 := d(); _1() }")

	simplifyAndCompare(t, "for a { b()() }", "for a { _1 := b(); _1() }")
	// simplifyAndCompare(t, "for a() { b() }", "for { _1 := a(); if !_1 { break }; b() }")

	simplifyAndCompare(t, "select { case <-a: b()(); default: c()() }", "select { case <-a: _1 := b(); _1(); default: _2 := c(); _2() }")
	simplifyAndCompare(t, "select { case <-a(): b; case <-c(): d }", "_1 := a(); _2 := c(); select { case <-_1: b; case <-_2: d }")
	simplifyAndCompare(t, "var d int; select { case a().f = <-b(): c; case d = <-e(): f }", "var d int; _3 := b(); _4 := e(); select { case _1 := <-_3: _2 := a(); _2.f = _1; c; case d = <-_4: f }")
	simplifyAndCompare(t, "select { case a() <- b(): c; case d() <- e(): f }", "_1 := a(); _2 := b(); _3 := d(); _4 := e(); select { case _1 <- _2: c; case _3 <- _4: f }")

	simplifyAndCompare(t, "a().f++", "_1 := a(); _1.f++")
	simplifyAndCompare(t, "go a()()", "_1 := a(); go _1()")
	simplifyAndCompare(t, "defer a()()", "_1 := a(); defer _1()")
	simplifyAndCompare(t, "a() <- b", "_1 := a(); _1 <- b")
	simplifyAndCompare(t, "a <- b()", "_1 := b(); a <- _1")

	simplifyAndCompare2(t, "f(g())", "_1, _2 := g(); f(_1, _2)", func(stmts []ast.Stmt) *types.Info {
		g := stmts[0].(*ast.ExprStmt).X.(*ast.CallExpr).Args[0].(*ast.CallExpr)
		return &types.Info{
			Types: map[ast.Expr]types.TypeAndValue{
				g: types.TypeAndValue{Type: types.NewTuple(
					types.NewParam(0, nil, "x", nil),
					types.NewParam(0, nil, "y", nil),
				)},
			},
		}
	})
}

func simplifyAndCompare(t *testing.T, in, out string) {
	emptyTypes := func(stmts []ast.Stmt) *types.Info { return &types.Info{} }
	simplifyAndCompare2(t, in, out, emptyTypes)
	simplifyAndCompare2(t, out, out, emptyTypes)
}

func simplifyAndCompare2(t *testing.T, in, out string, mockTypes func([]ast.Stmt) *types.Info) {
	fset := token.NewFileSet()

	expected := fprint(t, fset, parse(t, fset, out))

	file := parse(t, fset, in)
	f := file.Decls[0].(*ast.FuncDecl)
	f.Body.List = Simplify(f.Body.List, mockTypes(f.Body.List), true)
	got := fprint(t, fset, file)

	if got != expected {
		t.Errorf("\n--- input:\n%s\n--- expected output:\n%s--- got:\n%s", in, expected, got)
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

func TestContainsCall(t *testing.T) {
	testContainsCall(t, "a", false)
	testContainsCall(t, "a()", true)
	testContainsCall(t, "T{a, b}", false)
	testContainsCall(t, "T{a, b()}", true)
	testContainsCall(t, "T{a: a, b: b()}", true)
	testContainsCall(t, "(a())", true)
	testContainsCall(t, "a().f", true)
	testContainsCall(t, "a()[b]", true)
	testContainsCall(t, "a[b()]", true)
	testContainsCall(t, "a()[:]", true)
	testContainsCall(t, "a[b():]", true)
	testContainsCall(t, "a[:b()]", true)
	testContainsCall(t, "a[:b:c()]", true)
	testContainsCall(t, "a().(T)", true)
	testContainsCall(t, "*a()", true)
	testContainsCall(t, "-a()", true)
	testContainsCall(t, "&a()", true)
	testContainsCall(t, "&a()", true)
	testContainsCall(t, "a() + b", true)
	testContainsCall(t, "a + b()", true)
}

func testContainsCall(t *testing.T, in string, expected bool) {
	x, err := parser.ParseExpr(in)
	if err != nil {
		t.Fatal(err)
	}
	if got := ContainsCall(x); got != expected {
		t.Errorf("ContainsCall(%s): expected %t, got %t", in, expected, got)
	}
}
