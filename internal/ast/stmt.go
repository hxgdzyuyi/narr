package ast

import "narr/internal/source"

type StmtKind string

const (
	StmtAssert      StmtKind = "assert"
	StmtLet         StmtKind = "let"
	StmtForall      StmtKind = "forall"
	StmtExists      StmtKind = "exists"
	StmtCondition   StmtKind = "condition"
	StmtAssignment  StmtKind = "assignment"
	StmtSetAdd      StmtKind = "set_add"
	StmtSetRemove   StmtKind = "set_remove"
	StmtListAppend  StmtKind = "list_append"
	StmtLength      StmtKind = "length"
	StmtStartTarget StmtKind = "start_target"
	StmtDefault     StmtKind = "default"
	StmtInit        StmtKind = "init"
)

type Stmt struct {
	Kind    StmtKind
	Name    string
	Op      string
	Target  *Expr
	Value   *Expr
	Expr    *Expr
	Message *Expr
	Binder  *Binder
	Body    []Stmt
	Span    source.Span
}
