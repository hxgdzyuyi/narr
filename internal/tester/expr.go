package tester

import (
	"strconv"
	"strings"

	"narr/internal/ast"
)

func exprText(expr *ast.Expr) string {
	if expr == nil {
		return "<nil>"
	}
	switch expr.Kind {
	case ast.ExprInvalid:
		return "<invalid>"
	case ast.ExprString, ast.ExprMultiline:
		return strconv.Quote(expr.Value)
	case ast.ExprInteger, ast.ExprBool, ast.ExprRef, ast.ExprPath, ast.ExprCollection:
		return expr.Value
	case ast.ExprList:
		return "[" + exprListText(expr.Children) + "]"
	case ast.ExprSet:
		return "{" + exprListText(expr.Children) + "}"
	case ast.ExprLength:
		if len(expr.Children) == 0 {
			return "length"
		}
		return exprText(expr.Children[0])
	case ast.ExprUnary:
		if len(expr.Children) == 0 {
			return expr.Op
		}
		return expr.Op + " " + exprText(expr.Children[0])
	case ast.ExprBinary:
		return binaryText(expr)
	case ast.ExprPostfix:
		if len(expr.Children) == 0 {
			return expr.Op
		}
		return exprText(expr.Children[0]) + " " + expr.Op
	case ast.ExprCall:
		args := make([]string, 0, len(expr.Args))
		for _, arg := range expr.Args {
			args = append(args, exprText(arg))
		}
		return expr.Value + "(" + strings.Join(args, ", ") + ")"
	case ast.ExprCount:
		if expr.Binder != nil {
			return "count(" + binderText(expr.Binder) + ")"
		}
		return "count(" + exprListText(expr.Children) + ")"
	case ast.ExprCollect:
		value := ""
		if len(expr.Children) > 0 {
			value = exprText(expr.Children[0])
		}
		return "collect(" + value + " from " + binderText(expr.Binder) + ")"
	case ast.ExprState:
		if len(expr.Children) != 2 {
			return "state(...)"
		}
		return "state(" + exprText(expr.Children[0]) + ", at " + exprText(expr.Children[1]) + ")"
	case ast.ExprParen:
		if len(expr.Children) == 0 {
			return "()"
		}
		return "(" + exprText(expr.Children[0]) + ")"
	default:
		if expr.Value != "" {
			return expr.Value
		}
		if expr.Op != "" {
			return expr.Op
		}
		return string(expr.Kind)
	}
}

func binaryText(expr *ast.Expr) string {
	if len(expr.Children) == 0 {
		return expr.Op
	}
	if expr.Op == "between" && len(expr.Children) == 3 {
		return exprText(expr.Children[0]) + " between " + exprText(expr.Children[1]) + " and " + exprText(expr.Children[2])
	}
	parts := make([]string, 0, len(expr.Children))
	for _, child := range expr.Children {
		parts = append(parts, exprText(child))
	}
	return strings.Join(parts, " "+expr.Op+" ")
}

func binderText(binder *ast.Binder) string {
	if binder == nil {
		return "<nil>"
	}
	var builder strings.Builder
	builder.WriteString(binder.Domain)
	builder.WriteByte(' ')
	builder.WriteString(binder.Name)
	if binder.In != nil {
		builder.WriteString(" in ")
		builder.WriteString(exprText(binder.In))
	}
	if binder.Where != nil {
		builder.WriteString(" where ")
		builder.WriteString(exprText(binder.Where))
	}
	return builder.String()
}

func exprListText(exprs []*ast.Expr) string {
	parts := make([]string, 0, len(exprs))
	for _, expr := range exprs {
		parts = append(parts, exprText(expr))
	}
	return strings.Join(parts, ", ")
}
