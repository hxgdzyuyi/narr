package lexer

import (
	"fmt"

	"narr/internal/source"
)

type TokenKind int

const (
	TokenIllegal TokenKind = iota
	TokenEOF
	TokenNewline
	TokenIdentifier
	TokenKeyword
	TokenString
	TokenMultilineString
	TokenInteger
	TokenLBrace
	TokenRBrace
	TokenLBracket
	TokenRBracket
	TokenLParen
	TokenRParen
	TokenColon
	TokenComma
	TokenDot
	TokenEqual
	TokenEqualEqual
	TokenBangEqual
	TokenLess
	TokenLessEqual
	TokenGreater
	TokenGreaterEqual
	TokenPlusEqual
	TokenMinus
	TokenMinusEqual
	TokenArrow
	TokenAt
)

type Token struct {
	Kind    TokenKind
	Literal string
	Span    source.Span
}

func (t Token) String() string {
	return fmt.Sprintf("%s(%q)", t.Kind, t.Literal)
}

func (k TokenKind) String() string {
	switch k {
	case TokenIllegal:
		return "Illegal"
	case TokenEOF:
		return "EOF"
	case TokenNewline:
		return "Newline"
	case TokenIdentifier:
		return "Identifier"
	case TokenKeyword:
		return "Keyword"
	case TokenString:
		return "String"
	case TokenMultilineString:
		return "MultilineString"
	case TokenInteger:
		return "Integer"
	case TokenLBrace:
		return "LBrace"
	case TokenRBrace:
		return "RBrace"
	case TokenLBracket:
		return "LBracket"
	case TokenRBracket:
		return "RBracket"
	case TokenLParen:
		return "LParen"
	case TokenRParen:
		return "RParen"
	case TokenColon:
		return "Colon"
	case TokenComma:
		return "Comma"
	case TokenDot:
		return "Dot"
	case TokenEqual:
		return "Equal"
	case TokenEqualEqual:
		return "EqualEqual"
	case TokenBangEqual:
		return "BangEqual"
	case TokenLess:
		return "Less"
	case TokenLessEqual:
		return "LessEqual"
	case TokenGreater:
		return "Greater"
	case TokenGreaterEqual:
		return "GreaterEqual"
	case TokenPlusEqual:
		return "PlusEqual"
	case TokenMinus:
		return "Minus"
	case TokenMinusEqual:
		return "MinusEqual"
	case TokenArrow:
		return "Arrow"
	case TokenAt:
		return "At"
	default:
		return fmt.Sprintf("TokenKind(%d)", int(k))
	}
}

var Keywords = map[string]bool{
	"namespace": true, "import": true, "as": true,
	"novel": true, "enum": true, "class": true, "field": true,
	"volume": true, "chapter": true, "beat": true, "start_pattern": true,
	"place": true, "character": true, "collective": true, "faction": true, "object": true, "fact": true,
	"promise": true, "thread": true, "arc": true, "invariant": true, "style_note": true,
	"title": true, "language": true, "summary": true, "length": true, "prose_hint": true,
	"purpose": true, "target_chapters": true, "target_length": true,
	"pov": true, "location": true, "time_hint": true, "beats": true,
	"precondition": true, "effect": true, "on_screen": true, "observers": true,
	"sets_up": true, "pays_off": true, "advances": true, "resolves": true, "reveals": true, "mentions": true, "render_hint": true,
	"at": true, "requires": true, "starts": true, "tags": true, "note": true,
	"setup_at": true, "setup_strength": true, "payoff_by": true, "payoff_at": true, "payoff_kind": true, "question": true, "reader_visibility": true,
	"kind": true, "starts_at": true, "expected_resolution": true, "resolved_at": true, "priority": true,
	"subject": true, "state_field": true, "initial": true, "states": true,
	"hidden": true, "until": true, "always": true, "active_until": true,
	"test": true, "assert": true, "let": true, "forall": true, "exists": true, "where": true, "in": true, "not": true, "and": true, "or": true,
	"true": true, "false": true, "beginning": true, "end_of_story": true, "append": true,
}
