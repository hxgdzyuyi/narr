package ast

import "narr/internal/source"

type TestDecl struct {
	Name       string
	Attrs      []TestAttr
	Statements []Stmt
	Span       source.Span
}

type TestAttr struct {
	Name  string
	Value *Expr
	Span  source.Span
}
