package parser

import (
	"fmt"

	"narr/internal/ast"
	"narr/internal/lexer"
	"narr/internal/source"
)

func (p *Parser) parseBlock(kind ast.DeclKind) []ast.Field {
	p.expect(lexer.TokenLBrace, "{")
	p.skipNewlines()
	var fields []ast.Field
	for !p.at(lexer.TokenEOF) && !p.at(lexer.TokenRBrace) {
		if p.at(lexer.TokenNewline) {
			p.advance()
			continue
		}
		field, ok := p.parseBlockItem(kind)
		if ok {
			fields = append(fields, field)
			continue
		}
		p.consumeLineEnd()
	}
	p.expect(lexer.TokenRBrace, "}")
	p.consumeOptionalLineEnd()
	return fields
}

func (p *Parser) parseBlockItem(kind ast.DeclKind) (ast.Field, bool) {
	start := p.current().Span.Start
	if kind == ast.DeclClass && p.checkText("field") {
		return p.parseClassField(), true
	}
	if p.isName(p.current()) && p.peek(1).Kind == lexer.TokenColon {
		name := p.advance().Literal
		p.expect(lexer.TokenColon, ":")
		return p.parseNamedField(name, start), true
	}
	if p.isName(p.current()) {
		stmt := p.parseEffectLikeStmt()
		return ast.Field{Name: "", Statements: []ast.Stmt{stmt}, Span: fieldSpan(start, p)}, true
	}
	p.errorAt(p.current(), "E0214", fmt.Sprintf("unexpected token %q in block", p.current().Literal))
	return ast.Field{}, false
}

func (p *Parser) parseClassField() ast.Field {
	start := p.current().Span.Start
	p.expectText("field")
	name := p.expectName("class field name")
	p.expect(lexer.TokenColon, ":")
	typeExpr := p.parseTypeExpr()
	field := ast.Field{Name: name, Value: typeExpr}
	if p.match(lexer.TokenEqual) {
		value := p.parseExpr()
		field.Statements = append(field.Statements, ast.Stmt{Kind: ast.StmtDefault, Value: value, Span: value.Span})
	}
	p.consumeLineEnd()
	field.Span = fieldSpan(start, p)
	return field
}

func (p *Parser) parseNamedField(name string, start source.Position) ast.Field {
	field := ast.Field{Name: name}
	baseColumn := start.Column

	if name == "language" {
		field.Value = p.parseLanguageTag()
		p.consumeLineEnd()
		field.Span = fieldSpan(start, p)
		return field
	}
	if name == "hidden" {
		target := p.parseRef()
		p.expectText("until")
		value := p.parseAnchorRef()
		field.Value = &ast.Expr{Kind: ast.ExprBinary, Op: "until", Children: []*ast.Expr{target, value}, Span: spanFrom(start, p.last.Span.End)}
		p.consumeLineEnd()
		field.Span = fieldSpan(start, p)
		return field
	}

	if !p.at(lexer.TokenNewline) {
		value := p.parseExpr()
		if value != nil && value.Kind == ast.ExprInteger && p.isLengthUnitOnSameLine() {
			unit := p.advance()
			value = &ast.Expr{Kind: ast.ExprLength, Value: unit.Literal, Children: []*ast.Expr{value}, Span: spanFrom(value.Span.Start, unit.Span.End)}
		}
		if p.match(lexer.TokenEqual) {
			defaultValue := p.parseExpr()
			field.Statements = append(field.Statements, ast.Stmt{Kind: ast.StmtDefault, Value: defaultValue, Span: defaultValue.Span})
		}
		field.Value = value
		p.consumeLineEnd()
		field.Span = fieldSpan(start, p)
		return field
	}

	p.skipNewlines()
	switch name {
	case "length":
		field.Statements = p.parseLengthBlock(baseColumn)
	case "precondition", "requires", "always":
		field.Statements = p.parseConditionBlock(baseColumn)
	case "effect":
		field.Statements = p.parseEffectBlock(baseColumn)
	case "starts":
		field.Statements = p.parseStartTargetBlock(baseColumn)
	default:
		p.errorAt(p.current(), "E0215", fmt.Sprintf("field %q requires a value", name))
	}
	field.Span = fieldSpan(start, p)
	return field
}

func (p *Parser) parseLengthBlock(baseColumn int) []ast.Stmt {
	var stmts []ast.Stmt
	for p.currentStartsIndentedLine(baseColumn) {
		start := p.current().Span.Start
		name := p.expectName("length field name")
		p.expect(lexer.TokenEqual, "=")
		value := p.parseExpr()
		if value != nil && value.Kind == ast.ExprInteger && p.isLengthUnitOnSameLine() {
			unit := p.advance()
			value = &ast.Expr{Kind: ast.ExprLength, Value: unit.Literal, Children: []*ast.Expr{value}, Span: spanFrom(value.Span.Start, unit.Span.End)}
		}
		p.consumeLineEnd()
		stmts = append(stmts, ast.Stmt{Kind: ast.StmtLength, Name: name, Value: value, Span: spanFrom(start, p.last.Span.End)})
	}
	return stmts
}

func (p *Parser) parseConditionBlock(baseColumn int) []ast.Stmt {
	var stmts []ast.Stmt
	for p.currentStartsIndentedLine(baseColumn) {
		start := p.current().Span.Start
		expr := p.parseExpr()
		p.consumeLineEnd()
		stmts = append(stmts, ast.Stmt{Kind: ast.StmtCondition, Expr: expr, Span: spanFrom(start, p.last.Span.End)})
	}
	return stmts
}

func (p *Parser) parseEffectBlock(baseColumn int) []ast.Stmt {
	var stmts []ast.Stmt
	for p.currentStartsIndentedLine(baseColumn) {
		stmts = append(stmts, p.parseEffectLikeStmt())
	}
	return stmts
}

func (p *Parser) parseEffectLikeStmt() ast.Stmt {
	start := p.current().Span.Start
	target := p.parseRef()
	stmt := ast.Stmt{Target: target}
	switch {
	case p.match(lexer.TokenEqual):
		stmt.Kind = ast.StmtAssignment
		stmt.Op = "="
		stmt.Value = p.parseExpr()
	case p.match(lexer.TokenPlusEqual):
		stmt.Kind = ast.StmtSetAdd
		stmt.Op = "+="
		stmt.Value = p.parseExpr()
	case p.match(lexer.TokenMinusEqual):
		stmt.Kind = ast.StmtSetRemove
		stmt.Op = "-="
		stmt.Value = p.parseExpr()
	case p.matchText("append"):
		stmt.Kind = ast.StmtListAppend
		stmt.Op = "append"
		stmt.Value = p.parseExpr()
	default:
		p.errorAt(p.current(), "E0216", "expected effect operator")
		stmt.Kind = ast.StmtInit
	}
	p.consumeLineEnd()
	stmt.Span = spanFrom(start, p.last.Span.End)
	return stmt
}

func (p *Parser) parseStartTargetBlock(baseColumn int) []ast.Stmt {
	var stmts []ast.Stmt
	for p.currentStartsIndentedLine(baseColumn) {
		start := p.current().Span.Start
		targetKind := p.expectName("start target kind")
		if targetKind != "thread" && targetKind != "promise" && targetKind != "arc" {
			p.errorAt(p.last, "E0217", "start target must be thread, promise, or arc")
		}
		value := p.parseRef()
		p.consumeLineEnd()
		stmts = append(stmts, ast.Stmt{Kind: ast.StmtStartTarget, Name: targetKind, Value: value, Span: spanFrom(start, p.last.Span.End)})
	}
	return stmts
}

func (p *Parser) currentStartsIndentedLine(baseColumn int) bool {
	p.skipNewlines()
	if p.at(lexer.TokenEOF) || p.at(lexer.TokenRBrace) {
		return false
	}
	return p.current().Span.Start.Column > baseColumn
}

func (p *Parser) isLengthUnitOnSameLine() bool {
	if !p.isName(p.current()) {
		return false
	}
	switch p.current().Literal {
	case "字", "chars", "words":
		return p.current().Span.Start.Line == p.last.Span.End.Line
	default:
		return false
	}
}

func (p *Parser) parseLanguageTag() *ast.Expr {
	start := p.current().Span.Start
	if !p.isName(p.current()) {
		p.errorAt(p.current(), "E0218", "expected language tag")
		return ast.NewInvalidExpr()
	}
	parts := []string{p.advance().Literal}
	for p.match(lexer.TokenMinus) {
		parts = append(parts, p.expectName("language tag segment"))
	}
	return &ast.Expr{Kind: ast.ExprLanguage, Value: joinHyphen(parts), Span: spanFrom(start, p.last.Span.End)}
}

func joinHyphen(parts []string) string {
	if len(parts) == 0 {
		return ""
	}
	out := parts[0]
	for _, part := range parts[1:] {
		out += "-" + part
	}
	return out
}
