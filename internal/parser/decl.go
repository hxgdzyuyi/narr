package parser

import (
	"fmt"

	"narr/internal/ast"
	"narr/internal/lexer"
	"narr/internal/source"
)

func (p *Parser) parseDecl() ast.Decl {
	start := p.current().Span.Start
	keyword := p.advance().Literal
	decl := ast.Decl{Kind: ast.DeclKind(keyword)}

	switch decl.Kind {
	case ast.DeclNovel:
		if p.isName(p.current()) && !p.at(lexer.TokenLBrace) {
			decl.Name = p.advance().Literal
		}
		decl.Fields = p.parseOptionalBlock(decl.Kind)
	case ast.DeclEnum:
		decl.Name = p.expectName("enum name")
		decl.Members = p.parseEnumBlock()
	case ast.DeclClass:
		decl.Name = p.expectName("class name")
		decl.Fields = p.parseOptionalBlock(decl.Kind)
	case ast.DeclVolume:
		decl.Code = p.parseDottedName()
		p.parseAlias(&decl)
		decl.Fields = p.parseOptionalBlock(decl.Kind)
	case ast.DeclChapter:
		decl.Code = p.parseDottedName()
		p.parseAlias(&decl)
		decl.Fields = p.parseOptionalBlock(decl.Kind)
	case ast.DeclBeat:
		decl.Name = p.expectName("beat name")
		if p.match(lexer.TokenAt) {
			decl.Anchor = p.parseAnchorRef()
		}
		decl.Fields = p.parseOptionalBlock(decl.Kind)
	case ast.DeclStartPattern, ast.DeclPromise, ast.DeclThread, ast.DeclArc, ast.DeclInvariant:
		decl.Name = p.expectName(string(decl.Kind) + " name")
		decl.Fields = p.parseOptionalBlock(decl.Kind)
	case ast.DeclPlace, ast.DeclObject:
		decl.Name = p.expectName(string(decl.Kind) + " name")
		if p.matchText("in") {
			decl.In = p.parseRef()
		}
		decl.Fields = p.parseOptionalBlock(decl.Kind)
	case ast.DeclCharacter:
		decl.Name = p.expectName("character name")
		if p.match(lexer.TokenColon) {
			decl.Class = p.parseRef()
		}
		decl.Fields = p.parseOptionalBlock(decl.Kind)
	case ast.DeclCollective, ast.DeclFaction:
		decl.Name = p.expectName(string(decl.Kind) + " name")
		decl.Fields = p.parseOptionalBlock(decl.Kind)
	case ast.DeclFact:
		decl.Name = p.expectName("fact name")
		p.expect(lexer.TokenEqual, "=")
		decl.Value = p.parseTextValue()
		p.consumeLineEnd()
	case ast.DeclStyleNote:
		if p.isName(p.current()) && !p.at(lexer.TokenLBrace) {
			decl.Name = p.advance().Literal
		}
		decl.Fields = p.parseOptionalBlock(decl.Kind)
	default:
		p.errorAt(p.current(), "E0212", fmt.Sprintf("unsupported declaration %q", keyword))
		p.synchronizeTopLevel(modeNarr)
	}

	decl.Span = spanFrom(start, p.last.Span.End)
	return decl
}

func (p *Parser) parseAlias(decl *ast.Decl) {
	if p.matchText("as") {
		decl.Alias = p.expectName("alias")
		return
	}
	if p.matchText("alias") {
		decl.Alias = p.expectName("alias")
	}
}

func (p *Parser) parseEnumBlock() []string {
	p.expect(lexer.TokenLBrace, "{")
	p.skipNewlines()
	var members []string
	for !p.at(lexer.TokenEOF) && !p.at(lexer.TokenRBrace) {
		if p.at(lexer.TokenNewline) {
			p.advance()
			continue
		}
		if p.isName(p.current()) {
			members = append(members, p.advance().Literal)
			for p.match(lexer.TokenComma) {
				members = append(members, p.expectName("enum member"))
			}
			p.consumeOptionalLineEnd()
			continue
		}
		p.errorAt(p.current(), "E0213", "expected enum member")
		p.consumeLineEnd()
	}
	p.expect(lexer.TokenRBrace, "}")
	p.consumeOptionalLineEnd()
	return members
}

func (p *Parser) parseOptionalBlock(kind ast.DeclKind) []ast.Field {
	if !p.at(lexer.TokenLBrace) {
		p.consumeOptionalLineEnd()
		return nil
	}
	return p.parseBlock(kind)
}

func (p *Parser) parseAnchorRef() *ast.Expr {
	return p.parseRef()
}

func (p *Parser) parseRef() *ast.Expr {
	return p.parsePathExpr()
}

func (p *Parser) parseTextValue() *ast.Expr {
	switch p.current().Kind {
	case lexer.TokenString:
		token := p.advance()
		return &ast.Expr{Kind: ast.ExprString, Value: token.Literal, Span: token.Span}
	case lexer.TokenMultilineString:
		token := p.advance()
		return &ast.Expr{Kind: ast.ExprMultiline, Value: token.Literal, Span: token.Span}
	default:
		return p.parseExpr()
	}
}

func (p *Parser) parseTypeExpr() *ast.Expr {
	start := p.current().Span.Start
	name := p.parseDottedName()
	expr := &ast.Expr{Kind: ast.ExprRef, Value: name, Span: spanFrom(start, p.last.Span.End)}
	if (name == "Set" || name == "List") && p.match(lexer.TokenLess) {
		inner := p.parseTypeExpr()
		p.expect(lexer.TokenGreater, ">")
		expr = &ast.Expr{Kind: ast.ExprCall, Value: name, Args: []*ast.Expr{inner}, Span: spanFrom(start, p.last.Span.End)}
	}
	return expr
}

func fieldSpan(start source.Position, p *Parser) source.Span {
	return spanFrom(start, p.last.Span.End)
}
