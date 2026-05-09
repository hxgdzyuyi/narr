package lexer

import (
	"strings"
	"unicode"
	"unicode/utf8"

	"narr/internal/source"
)

type Lexer struct {
	path   string
	input  []rune
	index  int
	line   int
	column int
}

func Lex(path, text string) []Token {
	lexer := New(path, text)
	return lexer.All()
}

func New(path, text string) *Lexer {
	return &Lexer{
		path:   path,
		input:  []rune(text),
		line:   1,
		column: 1,
	}
}

func (l *Lexer) All() []Token {
	var tokens []Token
	for {
		token := l.Next()
		tokens = append(tokens, token)
		if token.Kind == TokenEOF {
			return tokens
		}
	}
}

func (l *Lexer) Next() Token {
	for {
		r := l.peek()
		switch r {
		case ' ', '\t', '\f', '\v':
			l.advance()
			continue
		case '#':
			l.skipLineComment()
			continue
		case '/':
			if l.peekN(1) == '/' {
				l.skipLineComment()
				continue
			}
		}
		break
	}

	start := l.position()
	r := l.peek()
	if r == 0 {
		return Token{Kind: TokenEOF, Literal: "", Span: source.Span{Start: start, End: start}}
	}
	if r == '\r' || r == '\n' {
		l.consumeNewline()
		return Token{Kind: TokenNewline, Literal: "\n", Span: source.Span{Start: start, End: l.position()}}
	}
	if isIdentifierStart(r) {
		return l.scanIdentifier(start)
	}
	if unicode.IsDigit(r) {
		return l.scanInteger(start)
	}
	if r == '"' {
		return l.scanString(start)
	}

	switch r {
	case '{':
		return l.single(TokenLBrace, "{", start)
	case '}':
		return l.single(TokenRBrace, "}", start)
	case '[':
		return l.single(TokenLBracket, "[", start)
	case ']':
		return l.single(TokenRBracket, "]", start)
	case '(':
		return l.single(TokenLParen, "(", start)
	case ')':
		return l.single(TokenRParen, ")", start)
	case ':':
		return l.single(TokenColon, ":", start)
	case ',':
		return l.single(TokenComma, ",", start)
	case '.':
		return l.single(TokenDot, ".", start)
	case '@':
		return l.single(TokenAt, "@", start)
	case '=':
		l.advance()
		if l.peek() == '=' {
			l.advance()
			return l.token(TokenEqualEqual, "==", start)
		}
		if l.peek() == '>' {
			l.advance()
			return l.token(TokenArrow, "=>", start)
		}
		return l.token(TokenEqual, "=", start)
	case '!':
		l.advance()
		if l.peek() == '=' {
			l.advance()
			return l.token(TokenBangEqual, "!=", start)
		}
		return l.token(TokenIllegal, "!", start)
	case '<':
		l.advance()
		if l.peek() == '=' {
			l.advance()
			return l.token(TokenLessEqual, "<=", start)
		}
		return l.token(TokenLess, "<", start)
	case '>':
		l.advance()
		if l.peek() == '=' {
			l.advance()
			return l.token(TokenGreaterEqual, ">=", start)
		}
		return l.token(TokenGreater, ">", start)
	case '+':
		l.advance()
		if l.peek() == '=' {
			l.advance()
			return l.token(TokenPlusEqual, "+=", start)
		}
		return l.token(TokenIllegal, "+", start)
	case '-':
		l.advance()
		if l.peek() == '=' {
			l.advance()
			return l.token(TokenMinusEqual, "-=", start)
		}
		if l.peek() == '>' {
			l.advance()
			return l.token(TokenArrow, "->", start)
		}
		return l.token(TokenMinus, "-", start)
	default:
		l.advance()
		return l.token(TokenIllegal, string(r), start)
	}
}

func (l *Lexer) scanIdentifier(start source.Position) Token {
	var builder strings.Builder
	for isIdentifierPart(l.peek()) {
		builder.WriteRune(l.advance())
	}
	literal := builder.String()
	kind := TokenIdentifier
	if Keywords[literal] {
		kind = TokenKeyword
	}
	return l.token(kind, literal, start)
}

func (l *Lexer) scanInteger(start source.Position) Token {
	var builder strings.Builder
	for unicode.IsDigit(l.peek()) {
		builder.WriteRune(l.advance())
	}
	return l.token(TokenInteger, builder.String(), start)
}

func (l *Lexer) scanString(start source.Position) Token {
	if l.peekN(0) == '"' && l.peekN(1) == '"' && l.peekN(2) == '"' {
		l.advance()
		l.advance()
		l.advance()
		var builder strings.Builder
		for {
			if l.peek() == 0 {
				return l.token(TokenIllegal, builder.String(), start)
			}
			if l.peekN(0) == '"' && l.peekN(1) == '"' && l.peekN(2) == '"' {
				l.advance()
				l.advance()
				l.advance()
				return l.token(TokenMultilineString, builder.String(), start)
			}
			builder.WriteRune(l.advance())
		}
	}

	l.advance()
	var builder strings.Builder
	for {
		r := l.peek()
		if r == 0 || r == '\n' || r == '\r' {
			return l.token(TokenIllegal, builder.String(), start)
		}
		if r == '"' {
			l.advance()
			return l.token(TokenString, builder.String(), start)
		}
		if r == '\\' {
			l.advance()
			builder.WriteRune(l.scanEscape())
			continue
		}
		builder.WriteRune(l.advance())
	}
}

func (l *Lexer) scanEscape() rune {
	switch r := l.advance(); r {
	case 'n':
		return '\n'
	case 'r':
		return '\r'
	case 't':
		return '\t'
	case '"':
		return '"'
	case '\\':
		return '\\'
	default:
		return r
	}
}

func (l *Lexer) skipLineComment() {
	for {
		r := l.peek()
		if r == 0 || r == '\n' || r == '\r' {
			return
		}
		l.advance()
	}
}

func (l *Lexer) consumeNewline() {
	if l.peek() == '\r' {
		l.advance()
		if l.peek() == '\n' {
			l.advance()
		}
		return
	}
	l.advance()
}

func (l *Lexer) single(kind TokenKind, literal string, start source.Position) Token {
	l.advance()
	return l.token(kind, literal, start)
}

func (l *Lexer) token(kind TokenKind, literal string, start source.Position) Token {
	return Token{
		Kind:    kind,
		Literal: literal,
		Span: source.Span{
			Start: start,
			End:   l.position(),
		},
	}
}

func (l *Lexer) position() source.Position {
	return source.Position{File: l.path, Line: l.line, Column: l.column}
}

func (l *Lexer) peek() rune {
	return l.peekN(0)
}

func (l *Lexer) peekN(offset int) rune {
	index := l.index + offset
	if index >= len(l.input) {
		return 0
	}
	return l.input[index]
}

func (l *Lexer) advance() rune {
	if l.index >= len(l.input) {
		return 0
	}
	r := l.input[l.index]
	l.index++
	if r == '\r' {
		if l.index < len(l.input) && l.input[l.index] == '\n' {
			l.index++
		}
		l.line++
		l.column = 1
		return '\n'
	}
	if r == '\n' {
		l.line++
		l.column = 1
		return r
	}
	if r == '\t' {
		l.column += 4
		return r
	}
	if r == utf8.RuneError {
		l.column++
		return r
	}
	l.column++
	return r
}

func isIdentifierStart(r rune) bool {
	return r == '_' || unicode.IsLetter(r)
}

func isIdentifierPart(r rune) bool {
	return r == '_' || unicode.IsLetter(r) || unicode.IsDigit(r)
}
