package astutil

import (
	"fmt"
	"go/ast"
	"go/token"
)

func Simplify(stmts []ast.Stmt) []ast.Stmt {
	c := &simplifyContext{}
	return c.simplifyStmtList(stmts)
}

func (c *simplifyContext) simplifyStmtList(stmts []ast.Stmt) []ast.Stmt {
	var newStmts []ast.Stmt
	for _, s := range stmts {
		switch s := s.(type) {
		case *ast.ExprStmt:
			newStmts = append(newStmts, &ast.ExprStmt{
				X: c.simplifyExpr(&newStmts, s.X),
			})
		default:
			newStmts = append(newStmts, s)
		}
	}
	return newStmts
}

type simplifyContext struct {
	varCounter int
}

func (c *simplifyContext) simplifyExpr(stmts *[]ast.Stmt, x ast.Expr) ast.Expr {
	switch x := x.(type) {
	case *ast.FuncLit:
		return &ast.FuncLit{
			Type: x.Type,
			Body: &ast.BlockStmt{
				List: c.simplifyStmtList(x.Body.List),
			},
		}

	case *ast.CompositeLit:
		elts := make([]ast.Expr, len(x.Elts))
		for i, elt := range x.Elts {
			if kv, ok := elt.(*ast.KeyValueExpr); ok {
				elts[i] = &ast.KeyValueExpr{
					Key:   kv.Key,
					Colon: kv.Colon,
					Value: c.newVar(stmts, kv.Value),
				}
				continue
			}
			elts[i] = c.newVar(stmts, elt)
		}
		return &ast.CompositeLit{
			Type:   x.Type,
			Lbrace: x.Lbrace,
			Elts:   elts,
			Rbrace: x.Rbrace,
		}

	case *ast.ParenExpr:
		return &ast.ParenExpr{
			Lparen: x.Lparen,
			X:      c.simplifyExpr(stmts, x.X),
			Rparen: x.Rparen,
		}

	case *ast.SelectorExpr:
		return &ast.SelectorExpr{
			X:   c.newVar(stmts, x.X),
			Sel: x.Sel,
		}

	case *ast.IndexExpr:
		return &ast.IndexExpr{
			X:      c.newVar(stmts, x.X),
			Lbrack: x.Lbrack,
			Index:  c.newVar(stmts, x.Index),
			Rbrack: x.Rbrack,
		}

	case *ast.SliceExpr:
		return &ast.SliceExpr{
			X:      c.newVar(stmts, x.X),
			Lbrack: x.Lbrack,
			Low:    c.newVar(stmts, x.Low),
			High:   c.newVar(stmts, x.High),
			Max:    c.newVar(stmts, x.Max),
			Slice3: x.Slice3,
			Rbrack: x.Rbrack,
		}

	case *ast.TypeAssertExpr:
		return &ast.TypeAssertExpr{
			X:      c.newVar(stmts, x.X),
			Lparen: x.Lparen,
			Type:   x.Type,
			Rparen: x.Rparen,
		}

	case *ast.CallExpr:
		fun := x.Fun
		if _, ok := fun.(*ast.SelectorExpr); !ok {
			fun = c.newVar(stmts, fun)
		}
		args := make([]ast.Expr, len(x.Args))
		for i, arg := range x.Args {
			args[i] = c.newVar(stmts, arg)
		}
		return &ast.CallExpr{
			Fun:      fun,
			Lparen:   x.Lparen,
			Args:     args,
			Ellipsis: x.Ellipsis,
			Rparen:   x.Rparen,
		}

	case *ast.StarExpr:
		return &ast.StarExpr{
			Star: x.Star,
			X:    c.newVar(stmts, x.X),
		}

	case *ast.UnaryExpr:
		return &ast.UnaryExpr{
			OpPos: x.OpPos,
			Op:    x.Op,
			X:     c.newVar(stmts, x.X),
		}

	case *ast.BinaryExpr:
		switch x.Op {
		case token.LAND, token.LOR:
			v := c.newVar(stmts, x.X)
			cond := v
			if x.Op == token.LOR {
				cond = &ast.UnaryExpr{
					Op: token.NOT,
					X:  cond,
				}
			}
			var ifBody []ast.Stmt
			ifBody = append(ifBody, simpleAssign(v, token.ASSIGN, c.simplifyExpr(&ifBody, x.Y)))
			*stmts = append(*stmts, &ast.IfStmt{
				Cond: cond,
				Body: &ast.BlockStmt{
					List: ifBody,
				},
			})
			return v
		default:
			return &ast.BinaryExpr{
				X:     c.newVar(stmts, x.X),
				OpPos: x.OpPos,
				Op:    x.Op,
				Y:     c.newVar(stmts, x.Y),
			}
		}

	default:
		return x
	}
}

func (c *simplifyContext) newVar(stmts *[]ast.Stmt, x ast.Expr) ast.Expr {
	if x == nil {
		return nil
	}
	if id, ok := x.(*ast.Ident); ok {
		return id
	}

	c.varCounter++
	id := ast.NewIdent(fmt.Sprintf("_%d", c.varCounter))
	*stmts = append(*stmts, simpleAssign(id, token.DEFINE, c.simplifyExpr(stmts, x)))
	return id
}

func simpleAssign(lhs ast.Expr, tok token.Token, rhs ast.Expr) *ast.AssignStmt {
	return &ast.AssignStmt{
		Lhs: []ast.Expr{lhs},
		Tok: tok,
		Rhs: []ast.Expr{rhs},
	}
}
