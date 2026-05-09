package lexer

import "testing"

func TestLexerUnicodeIdentifiersAndOperators(t *testing.T) {
	tokens := Lex("sample.narr", "namespace 红楼梦.structure\nchars.沈夜.知道 += world.事实\n")
	want := []struct {
		kind    TokenKind
		literal string
	}{
		{TokenKeyword, "namespace"},
		{TokenIdentifier, "红楼梦"},
		{TokenDot, "."},
		{TokenIdentifier, "structure"},
		{TokenNewline, "\n"},
		{TokenIdentifier, "chars"},
		{TokenDot, "."},
		{TokenIdentifier, "沈夜"},
		{TokenDot, "."},
		{TokenIdentifier, "知道"},
		{TokenPlusEqual, "+="},
		{TokenIdentifier, "world"},
		{TokenDot, "."},
		{TokenIdentifier, "事实"},
		{TokenNewline, "\n"},
		{TokenEOF, ""},
	}
	if len(tokens) != len(want) {
		t.Fatalf("len(tokens) = %d, want %d: %#v", len(tokens), len(want), tokens)
	}
	for i, expected := range want {
		if tokens[i].Kind != expected.kind || tokens[i].Literal != expected.literal {
			t.Fatalf("token %d = %s %q, want %s %q", i, tokens[i].Kind, tokens[i].Literal, expected.kind, expected.literal)
		}
	}
}

func TestLexerCommentsStringsAndPositions(t *testing.T) {
	tokens := Lex("sample.narr", "# comment\nfact 事实 = \"文本\"\n")
	if tokens[0].Kind != TokenNewline {
		t.Fatalf("first token = %s, want Newline", tokens[0].Kind)
	}
	if tokens[1].Literal != "fact" || tokens[1].Span.Start.Line != 2 || tokens[1].Span.Start.Column != 1 {
		t.Fatalf("fact token = %#v, want line 2 column 1", tokens[1])
	}
	if tokens[4].Kind != TokenString || tokens[4].Literal != "文本" {
		t.Fatalf("string token = %#v, want string 文本", tokens[4])
	}
}
