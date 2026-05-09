package ast

func NewInvalidExpr() *Expr {
	return &Expr{Kind: ExprInvalid}
}
