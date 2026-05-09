package ast

import "narr/internal/source"

type ExprKind string

const (
	ExprInvalid    ExprKind = "invalid"
	ExprString     ExprKind = "string"
	ExprMultiline  ExprKind = "multiline_string"
	ExprInteger    ExprKind = "integer"
	ExprBool       ExprKind = "bool"
	ExprSymbol     ExprKind = "symbol"
	ExprRef        ExprKind = "ref"
	ExprPath       ExprKind = "path"
	ExprList       ExprKind = "list"
	ExprSet        ExprKind = "set"
	ExprLength     ExprKind = "length"
	ExprLanguage   ExprKind = "language"
	ExprUnary      ExprKind = "unary"
	ExprBinary     ExprKind = "binary"
	ExprPostfix    ExprKind = "postfix"
	ExprCall       ExprKind = "call"
	ExprMember     ExprKind = "member"
	ExprCollection ExprKind = "collection"
	ExprCount      ExprKind = "count"
	ExprCollect    ExprKind = "collect"
	ExprState      ExprKind = "state"
	ExprParen      ExprKind = "paren"
)

type Expr struct {
	Kind     ExprKind
	Value    string
	Op       string
	Children []*Expr
	Args     []*Expr
	Binder   *Binder
	Span     source.Span
}

type Binder struct {
	Domain string
	Name   string
	In     *Expr
	Where  *Expr
	Span   source.Span
}
