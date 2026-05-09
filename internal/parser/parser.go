package parser

import (
	"fmt"
	"os"

	"narr/internal/ast"
	"narr/internal/lexer"
	"narr/internal/project"
	"narr/internal/source"
)

type parserMode int

const (
	modeNarr parserMode = iota
	modeTest
)

type Parser struct {
	path        string
	tokens      []lexer.Token
	index       int
	diagnostics []source.Diagnostic
	last        lexer.Token
}

func ParseNarrFile(path, text string) (*ast.File, []source.Diagnostic) {
	p := newParser(path, text)
	return p.parseFile(modeNarr), p.diagnostics
}

func ParseTestFile(path, text string) (*ast.File, []source.Diagnostic) {
	p := newParser(path, text)
	return p.parseFile(modeTest), p.diagnostics
}

func ParseExpression(path, text string) (*ast.Expr, []source.Diagnostic) {
	p := newParser(path, text)
	expr := p.parseExpr()
	p.skipNewlines()
	if !p.at(lexer.TokenEOF) {
		p.errorAt(p.current(), "E0201", fmt.Sprintf("unexpected token %q after expression", p.current().Literal))
	}
	return expr, p.diagnostics
}

func ParseProject(loaded *project.Project) ([]*ast.File, []source.Diagnostic) {
	var files []*ast.File
	var diagnostics []source.Diagnostic
	for _, file := range loaded.Files {
		data, err := os.ReadFile(file.Path)
		if err != nil {
			diagnostics = append(diagnostics, source.Error("E0200", file.Path, 1, 1, "failed to read source file: "+err.Error()))
			continue
		}
		var parsed *ast.File
		var fileDiagnostics []source.Diagnostic
		switch file.Kind {
		case project.FileKindTest:
			parsed, fileDiagnostics = ParseTestFile(file.Path, string(data))
		default:
			parsed, fileDiagnostics = ParseNarrFile(file.Path, string(data))
		}
		files = append(files, parsed)
		diagnostics = append(diagnostics, fileDiagnostics...)
	}
	return files, diagnostics
}

func newParser(path, text string) *Parser {
	tokens := lexer.Lex(path, text)
	last := tokens[0]
	return &Parser{path: path, tokens: tokens, last: last}
}

func (p *Parser) parseFile(mode parserMode) *ast.File {
	file := &ast.File{Path: p.path}
	if mode == modeTest {
		file.Mode = ast.ModeTest
	} else {
		file.Mode = ast.ModeNarr
	}

	p.skipNewlines()
	if p.matchText("namespace") {
		file.Namespace = p.parseNamespacePath()
	} else {
		p.errorAt(p.current(), "E0202", "file must start with namespace declaration")
		p.synchronizeTopLevel(mode)
	}
	p.skipNewlines()

	for p.matchText("import") {
		file.Imports = append(file.Imports, p.parseImportAfterKeyword())
		p.skipNewlines()
	}

	for !p.at(lexer.TokenEOF) {
		p.skipNewlines()
		if p.at(lexer.TokenEOF) {
			break
		}
		if mode == modeTest {
			if p.checkText("test") {
				file.Tests = append(file.Tests, p.parseTestDecl())
			} else {
				p.errorAt(p.current(), "E0203", ".test.narr files may only contain test declarations after imports")
				p.synchronizeTopLevel(mode)
			}
			continue
		}
		if p.checkText("test") {
			p.errorAt(p.current(), "E0204", ".narr files may not contain test declarations")
			p.synchronizeTopLevel(mode)
			continue
		}
		if isNarrDeclarationKeyword(p.current().Literal) {
			file.Decls = append(file.Decls, p.parseDecl())
			continue
		}
		p.errorAt(p.current(), "E0205", fmt.Sprintf("unexpected top-level token %q", p.current().Literal))
		p.synchronizeTopLevel(mode)
	}

	if len(p.tokens) > 0 {
		file.Span = source.Span{Start: p.tokens[0].Span.Start, End: p.last.Span.End}
	}
	return file
}

func (p *Parser) parseNamespacePath() string {
	path := p.parseDottedName()
	p.consumeLineEnd()
	return path
}

func (p *Parser) parseImportAfterKeyword() ast.ImportDecl {
	start := p.last.Span.Start
	path := p.parseDottedName()
	alias := ""
	if p.matchText("as") {
		alias = p.expectName("import alias")
	}
	p.consumeLineEnd()
	return ast.ImportDecl{Path: path, Alias: alias, Span: source.Span{Start: start, End: p.last.Span.End}}
}

func (p *Parser) parseDottedName() string {
	if !p.isName(p.current()) {
		p.errorAt(p.current(), "E0206", "expected identifier")
		return ""
	}
	parts := []string{p.advance().Literal}
	for p.match(lexer.TokenDot) {
		if !p.isName(p.current()) {
			p.errorAt(p.current(), "E0207", "expected identifier after dot")
			break
		}
		parts = append(parts, p.advance().Literal)
	}
	return joinDotted(parts)
}

func (p *Parser) current() lexer.Token {
	if p.index >= len(p.tokens) {
		return p.tokens[len(p.tokens)-1]
	}
	return p.tokens[p.index]
}

func (p *Parser) peek(offset int) lexer.Token {
	index := p.index + offset
	if index >= len(p.tokens) {
		return p.tokens[len(p.tokens)-1]
	}
	return p.tokens[index]
}

func (p *Parser) advance() lexer.Token {
	token := p.current()
	if token.Kind != lexer.TokenEOF {
		p.index++
	}
	p.last = token
	return token
}

func (p *Parser) at(kind lexer.TokenKind) bool {
	return p.current().Kind == kind
}

func (p *Parser) match(kind lexer.TokenKind) bool {
	if !p.at(kind) {
		return false
	}
	p.advance()
	return true
}

func (p *Parser) checkText(text string) bool {
	token := p.current()
	return (token.Kind == lexer.TokenIdentifier || token.Kind == lexer.TokenKeyword) && token.Literal == text
}

func (p *Parser) peekText(offset int, text string) bool {
	token := p.peek(offset)
	return (token.Kind == lexer.TokenIdentifier || token.Kind == lexer.TokenKeyword) && token.Literal == text
}

func (p *Parser) matchText(text string) bool {
	if !p.checkText(text) {
		return false
	}
	p.advance()
	return true
}

func (p *Parser) expect(kind lexer.TokenKind, description string) lexer.Token {
	if p.at(kind) {
		return p.advance()
	}
	p.errorAt(p.current(), "E0208", "expected "+description)
	return lexer.Token{Kind: kind, Span: p.current().Span}
}

func (p *Parser) expectText(text string) lexer.Token {
	if p.checkText(text) {
		return p.advance()
	}
	p.errorAt(p.current(), "E0209", fmt.Sprintf("expected %q", text))
	return lexer.Token{Kind: lexer.TokenIdentifier, Literal: text, Span: p.current().Span}
}

func (p *Parser) expectName(description string) string {
	if p.isName(p.current()) {
		return p.advance().Literal
	}
	p.errorAt(p.current(), "E0210", "expected "+description)
	return ""
}

func (p *Parser) isName(token lexer.Token) bool {
	return token.Kind == lexer.TokenIdentifier || token.Kind == lexer.TokenKeyword
}

func (p *Parser) skipNewlines() {
	for p.at(lexer.TokenNewline) {
		p.advance()
	}
}

func (p *Parser) consumeLineEnd() {
	for !p.at(lexer.TokenEOF) && !p.at(lexer.TokenNewline) && !p.at(lexer.TokenRBrace) {
		p.errorAt(p.current(), "E0211", fmt.Sprintf("unexpected token %q before end of line", p.current().Literal))
		p.advance()
	}
	p.skipNewlines()
}

func (p *Parser) consumeOptionalLineEnd() {
	if p.at(lexer.TokenNewline) {
		p.skipNewlines()
	}
}

func (p *Parser) errorAt(token lexer.Token, code, message string) {
	p.diagnostics = append(p.diagnostics, source.Error(code, token.Span.Start.File, token.Span.Start.Line, token.Span.Start.Column, message))
}

func (p *Parser) synchronizeTopLevel(mode parserMode) {
	for !p.at(lexer.TokenEOF) {
		if p.at(lexer.TokenNewline) {
			p.advance()
			if mode == modeTest && p.checkText("test") {
				return
			}
			if mode == modeNarr && isNarrDeclarationKeyword(p.current().Literal) {
				return
			}
			if p.checkText("import") {
				return
			}
			continue
		}
		p.advance()
	}
}

func (p *Parser) sameLine(token lexer.Token) bool {
	return token.Span.Start.Line == p.last.Span.End.Line
}

func spanFrom(start source.Position, end source.Position) source.Span {
	return source.Span{Start: start, End: end}
}

func joinDotted(parts []string) string {
	if len(parts) == 0 {
		return ""
	}
	out := parts[0]
	for _, part := range parts[1:] {
		out += "." + part
	}
	return out
}

func isNarrDeclarationKeyword(text string) bool {
	switch text {
	case "novel", "enum", "class", "volume", "chapter", "beat", "start_pattern",
		"place", "character", "collective", "faction", "object", "fact",
		"promise", "thread", "arc", "invariant", "style_note":
		return true
	default:
		return false
	}
}

func isDomainType(text string) bool {
	switch text {
	case "novel", "volume", "chapter", "beat", "thread", "promise", "arc", "character", "place", "object", "fact":
		return true
	default:
		return false
	}
}
