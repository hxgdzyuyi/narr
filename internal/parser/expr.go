package parser

import (
	"fmt"

	"narr/internal/ast"
	"narr/internal/lexer"
)

func (p *Parser) parseExpr() *ast.Expr {
	return p.parseImplication()
}

func (p *Parser) parseImplication() *ast.Expr {
	left := p.parseOr()
	if p.match(lexer.TokenArrow) {
		start := left.Span.Start
		right := p.parseImplication()
		return &ast.Expr{Kind: ast.ExprBinary, Op: "=>", Children: []*ast.Expr{left, right}, Span: spanFrom(start, right.Span.End)}
	}
	return left
}

func (p *Parser) parseOr() *ast.Expr {
	left := p.parseAnd()
	for p.matchText("or") {
		start := left.Span.Start
		right := p.parseAnd()
		left = &ast.Expr{Kind: ast.ExprBinary, Op: "or", Children: []*ast.Expr{left, right}, Span: spanFrom(start, right.Span.End)}
	}
	return left
}

func (p *Parser) parseAnd() *ast.Expr {
	left := p.parseNot()
	for p.matchText("and") {
		start := left.Span.Start
		right := p.parseNot()
		left = &ast.Expr{Kind: ast.ExprBinary, Op: "and", Children: []*ast.Expr{left, right}, Span: spanFrom(start, right.Span.End)}
	}
	return left
}

func (p *Parser) parseNot() *ast.Expr {
	if p.matchText("not") {
		start := p.last.Span.Start
		right := p.parseNot()
		return &ast.Expr{Kind: ast.ExprUnary, Op: "not", Children: []*ast.Expr{right}, Span: spanFrom(start, right.Span.End)}
	}
	return p.parsePredicate()
}

func (p *Parser) parsePredicate() *ast.Expr {
	left := p.parseValueExpr()
	if left == nil {
		return ast.NewInvalidExpr()
	}

	if p.matchText("exists") {
		return &ast.Expr{Kind: ast.ExprPostfix, Op: "exists", Children: []*ast.Expr{left}, Span: spanFrom(left.Span.Start, p.last.Span.End)}
	}
	if p.matchText("missing") {
		return &ast.Expr{Kind: ast.ExprPostfix, Op: "missing", Children: []*ast.Expr{left}, Span: spanFrom(left.Span.Start, p.last.Span.End)}
	}
	if op, ok := p.matchComparisonOp(); ok {
		right := p.parseValueExpr()
		return &ast.Expr{Kind: ast.ExprBinary, Op: op, Children: []*ast.Expr{left, right}, Span: spanFrom(left.Span.Start, right.Span.End)}
	}
	if p.matchText("not") {
		if p.matchText("in") {
			right := p.parseValueExpr()
			return &ast.Expr{Kind: ast.ExprBinary, Op: "not in", Children: []*ast.Expr{left, right}, Span: spanFrom(left.Span.Start, right.Span.End)}
		}
		p.errorAt(p.last, "E0219", "expected in after not")
	}
	if p.matchText("in") {
		right := p.parseValueExpr()
		return &ast.Expr{Kind: ast.ExprBinary, Op: "in", Children: []*ast.Expr{left, right}, Span: spanFrom(left.Span.Start, right.Span.End)}
	}
	if p.matchText("between") {
		first := p.parseValueExpr()
		p.expectText("and")
		second := p.parseValueExpr()
		return &ast.Expr{Kind: ast.ExprBinary, Op: "between", Children: []*ast.Expr{left, first, second}, Span: spanFrom(left.Span.Start, second.Span.End)}
	}
	if op := p.matchTemporalOrNarrativeOp(); op != "" {
		right := p.parseValueExpr()
		children := []*ast.Expr{left, right}
		if op == "changes" {
			if p.matchText("to") {
				to := p.parseValueExpr()
				children = append(children, to)
			} else if p.matchText("from") {
				from := p.parseValueExpr()
				p.expectText("to")
				to := p.parseValueExpr()
				children = append(children, from, to)
			}
		}
		return &ast.Expr{Kind: ast.ExprBinary, Op: op, Children: children, Span: spanFrom(left.Span.Start, children[len(children)-1].Span.End)}
	}

	return left
}

func (p *Parser) matchComparisonOp() (string, bool) {
	switch {
	case p.match(lexer.TokenEqualEqual):
		return "==", true
	case p.match(lexer.TokenBangEqual):
		return "!=", true
	case p.match(lexer.TokenLess):
		return "<", true
	case p.match(lexer.TokenLessEqual):
		return "<=", true
	case p.match(lexer.TokenGreater):
		return ">", true
	case p.match(lexer.TokenGreaterEqual):
		return ">=", true
	default:
		return "", false
	}
}

func (p *Parser) matchTemporalOrNarrativeOp() string {
	for _, op := range []string{
		"precedes", "at_or_before", "at_or_after", "in_volume",
		"serves", "mentions", "sets_up", "pays_off", "starts", "advances", "resolves", "reveals", "changes",
	} {
		if p.matchText(op) {
			return op
		}
	}
	return ""
}

func (p *Parser) parseValueExpr() *ast.Expr {
	p.skipNewlines()
	token := p.current()
	var expr *ast.Expr
	switch token.Kind {
	case lexer.TokenString:
		p.advance()
		expr = &ast.Expr{Kind: ast.ExprString, Value: token.Literal, Span: token.Span}
	case lexer.TokenMultilineString:
		p.advance()
		expr = &ast.Expr{Kind: ast.ExprMultiline, Value: token.Literal, Span: token.Span}
	case lexer.TokenInteger:
		p.advance()
		expr = &ast.Expr{Kind: ast.ExprInteger, Value: token.Literal, Span: token.Span}
	case lexer.TokenLBracket:
		expr = p.parseDelimitedExprList(lexer.TokenLBracket, lexer.TokenRBracket, ast.ExprList)
	case lexer.TokenLBrace:
		expr = p.parseDelimitedExprList(lexer.TokenLBrace, lexer.TokenRBrace, ast.ExprSet)
	case lexer.TokenLParen:
		start := p.advance().Span.Start
		inner := p.parseExpr()
		p.expect(lexer.TokenRParen, ")")
		expr = &ast.Expr{Kind: ast.ExprParen, Children: []*ast.Expr{inner}, Span: spanFrom(start, p.last.Span.End)}
	default:
		if p.isName(token) {
			if token.Literal == "true" || token.Literal == "false" {
				p.advance()
				expr = &ast.Expr{Kind: ast.ExprBool, Value: token.Literal, Span: token.Span}
			} else {
				expr = p.parseNamedValueExpr()
			}
		}
	}
	if expr != nil {
		return p.parseMemberExpr(expr)
	}
	p.errorAt(token, "E0220", fmt.Sprintf("expected expression, got %q", token.Literal))
	p.advance()
	return &ast.Expr{Kind: ast.ExprInvalid, Span: token.Span}
}

func (p *Parser) parseMemberExpr(base *ast.Expr) *ast.Expr {
	for p.match(lexer.TokenDot) {
		start := base.Span.Start
		parts := []string{p.expectName("property name")}
		for p.match(lexer.TokenDot) {
			parts = append(parts, p.expectName("property name"))
		}
		base = &ast.Expr{Kind: ast.ExprMember, Value: joinDotted(parts), Children: []*ast.Expr{base}, Span: spanFrom(start, p.last.Span.End)}
	}
	return base
}

func (p *Parser) parseNamedValueExpr() *ast.Expr {
	if p.checkText("count") && p.peek(1).Kind == lexer.TokenLParen {
		return p.parseCountExpr()
	}
	if p.checkText("collect") && p.peek(1).Kind == lexer.TokenLParen {
		return p.parseCollectExpr()
	}
	if p.checkText("state") && p.peek(1).Kind == lexer.TokenLParen {
		return p.parseStateExpr()
	}

	start := p.current().Span.Start
	name := p.parseDottedName()
	if p.at(lexer.TokenLParen) {
		p.advance()
		args := p.parseArgumentList()
		p.expect(lexer.TokenRParen, ")")
		return &ast.Expr{Kind: ast.ExprCall, Value: name, Args: args, Span: spanFrom(start, p.last.Span.End)}
	}
	if isCollectionName(name) {
		return &ast.Expr{Kind: ast.ExprCollection, Value: name, Span: spanFrom(start, p.last.Span.End)}
	}
	if containsDot(name) {
		return &ast.Expr{Kind: ast.ExprPath, Value: name, Span: spanFrom(start, p.last.Span.End)}
	}
	return &ast.Expr{Kind: ast.ExprRef, Value: name, Span: spanFrom(start, p.last.Span.End)}
}

func (p *Parser) parsePathExpr() *ast.Expr {
	start := p.current().Span.Start
	name := p.parseDottedName()
	kind := ast.ExprRef
	if containsDot(name) {
		kind = ast.ExprPath
	}
	return &ast.Expr{Kind: kind, Value: name, Span: spanFrom(start, p.last.Span.End)}
}

func (p *Parser) parseDelimitedExprList(open, close lexer.TokenKind, kind ast.ExprKind) *ast.Expr {
	start := p.expect(open, open.String()).Span.Start
	var children []*ast.Expr
	p.skipNewlines()
	for !p.at(lexer.TokenEOF) && !p.at(close) {
		children = append(children, p.parseExpr())
		p.skipNewlines()
		if !p.match(lexer.TokenComma) {
			break
		}
		p.skipNewlines()
	}
	p.expect(close, close.String())
	return &ast.Expr{Kind: kind, Children: children, Span: spanFrom(start, p.last.Span.End)}
}

func (p *Parser) parseArgumentList() []*ast.Expr {
	var args []*ast.Expr
	p.skipNewlines()
	for !p.at(lexer.TokenEOF) && !p.at(lexer.TokenRParen) {
		args = append(args, p.parseExpr())
		p.skipNewlines()
		if !p.match(lexer.TokenComma) {
			break
		}
		p.skipNewlines()
	}
	return args
}

func (p *Parser) parseCountExpr() *ast.Expr {
	start := p.expectText("count").Span.Start
	p.expect(lexer.TokenLParen, "(")
	expr := &ast.Expr{Kind: ast.ExprCount}
	p.skipNewlines()
	if isDomainType(p.current().Literal) && p.isName(p.peek(1)) {
		binder := p.parseBinder()
		if p.matchText("where") {
			binder.Where = p.parseExpr()
		}
		expr.Binder = binder
	} else {
		expr.Children = append(expr.Children, p.parseExpr())
	}
	p.expect(lexer.TokenRParen, ")")
	expr.Span = spanFrom(start, p.last.Span.End)
	return expr
}

func (p *Parser) parseCollectExpr() *ast.Expr {
	start := p.expectText("collect").Span.Start
	p.expect(lexer.TokenLParen, "(")
	value := p.parseValueExpr()
	p.expectText("from")
	binder := p.parseBinder()
	if p.matchText("where") {
		binder.Where = p.parseExpr()
	}
	p.expect(lexer.TokenRParen, ")")
	return &ast.Expr{Kind: ast.ExprCollect, Children: []*ast.Expr{value}, Binder: binder, Span: spanFrom(start, p.last.Span.End)}
}

func (p *Parser) parseStateExpr() *ast.Expr {
	start := p.expectText("state").Span.Start
	p.expect(lexer.TokenLParen, "(")
	stateRef := p.parseRef()
	p.expect(lexer.TokenComma, ",")
	p.expectText("at")
	anchor := p.parseExpr()
	p.expect(lexer.TokenRParen, ")")
	return &ast.Expr{Kind: ast.ExprState, Children: []*ast.Expr{stateRef, anchor}, Span: spanFrom(start, p.last.Span.End)}
}

func (p *Parser) parseBinder() *ast.Binder {
	start := p.current().Span.Start
	domain := p.expectName("domain type")
	if !isDomainType(domain) {
		p.errorAt(p.last, "E0221", fmt.Sprintf("unknown domain type %q", domain))
	}
	name := p.expectName("binder name")
	binder := &ast.Binder{Domain: domain, Name: name, Span: spanFrom(start, p.last.Span.End)}
	if p.matchText("in") {
		binder.In = p.parseExpr()
		binder.Span = spanFrom(start, p.last.Span.End)
	}
	return binder
}

func isCollectionName(name string) bool {
	switch name {
	case "novels", "volumes", "chapters", "beats", "threads", "promises", "arcs", "characters", "places", "objects", "facts":
		return true
	default:
		return false
	}
}

func containsDot(text string) bool {
	for _, r := range text {
		if r == '.' {
			return true
		}
	}
	return false
}
