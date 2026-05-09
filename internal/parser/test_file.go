package parser

import (
	"fmt"

	"narr/internal/ast"
	"narr/internal/lexer"
)

func (p *Parser) parseTestDecl() ast.TestDecl {
	start := p.expectText("test").Span.Start
	nameToken := p.expect(lexer.TokenString, "test name string")
	test := ast.TestDecl{Name: nameToken.Literal}
	for p.checkText("tags") {
		attrStart := p.advance().Span.Start
		value := p.parseValueExpr()
		test.Attrs = append(test.Attrs, ast.TestAttr{Name: "tags", Value: value, Span: spanFrom(attrStart, p.last.Span.End)})
	}
	p.expect(lexer.TokenLBrace, "{")
	p.skipNewlines()
	test.Statements = p.parseTestStatementsUntil(0)
	p.expect(lexer.TokenRBrace, "}")
	p.consumeOptionalLineEnd()
	test.Span = spanFrom(start, p.last.Span.End)
	return test
}

func (p *Parser) parseTestStatementsUntil(parentColumn int) []ast.Stmt {
	var stmts []ast.Stmt
	for !p.at(lexer.TokenEOF) && !p.at(lexer.TokenRBrace) {
		p.skipNewlines()
		if p.at(lexer.TokenEOF) || p.at(lexer.TokenRBrace) {
			break
		}
		if parentColumn > 0 && p.current().Span.Start.Column <= parentColumn {
			break
		}
		stmt := p.parseTestStmt()
		stmts = append(stmts, stmt)
	}
	return stmts
}

func (p *Parser) parseTestStmt() ast.Stmt {
	switch {
	case p.checkText("assert"):
		return p.parseAssertStmt()
	case p.checkText("let"):
		return p.parseLetStmt()
	case p.checkText("forall"):
		return p.parseForallStmt()
	case p.checkText("exists"):
		return p.parseExistsStmt()
	default:
		start := p.current().Span.Start
		p.errorAt(p.current(), "E0222", fmt.Sprintf("expected test statement, got %q", p.current().Literal))
		p.consumeLineEnd()
		return ast.Stmt{Span: spanFrom(start, p.last.Span.End)}
	}
}

func (p *Parser) parseAssertStmt() ast.Stmt {
	start := p.expectText("assert").Span.Start
	expr := p.parseExpr()
	stmt := ast.Stmt{Kind: ast.StmtAssert, Expr: expr}
	if p.matchText("message") {
		stmt.Message = p.parseTextValue()
	}
	p.consumeLineEnd()
	stmt.Span = spanFrom(start, p.last.Span.End)
	return stmt
}

func (p *Parser) parseLetStmt() ast.Stmt {
	start := p.expectText("let").Span.Start
	name := p.expectName("let variable")
	p.expect(lexer.TokenEqual, "=")
	value := p.parseExpr()
	p.consumeLineEnd()
	return ast.Stmt{Kind: ast.StmtLet, Name: name, Value: value, Span: spanFrom(start, p.last.Span.End)}
}

func (p *Parser) parseForallStmt() ast.Stmt {
	startToken := p.expectText("forall")
	binder := p.parseBinder()
	if p.matchText("where") {
		binder.Where = p.parseExpr()
	}
	body := p.parseTestStmtBody(startToken.Span.Start.Column)
	return ast.Stmt{Kind: ast.StmtForall, Binder: binder, Body: body, Span: spanFrom(startToken.Span.Start, p.last.Span.End)}
}

func (p *Parser) parseExistsStmt() ast.Stmt {
	startToken := p.expectText("exists")
	binder := p.parseBinder()
	if p.matchText("where") {
		binder.Where = p.parseExpr()
	}
	var body []ast.Stmt
	if p.at(lexer.TokenColon) {
		body = p.parseTestStmtBody(startToken.Span.Start.Column)
	} else {
		p.consumeLineEnd()
	}
	return ast.Stmt{Kind: ast.StmtExists, Binder: binder, Body: body, Span: spanFrom(startToken.Span.Start, p.last.Span.End)}
}

func (p *Parser) parseTestStmtBody(parentColumn int) []ast.Stmt {
	p.expect(lexer.TokenColon, ":")
	if !p.at(lexer.TokenNewline) {
		stmt := p.parseTestStmt()
		return []ast.Stmt{stmt}
	}
	p.skipNewlines()
	return p.parseTestStatementsUntil(parentColumn)
}
