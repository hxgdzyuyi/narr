package ast

import "narr/internal/source"

type DeclKind string

const (
	DeclNovel        DeclKind = "novel"
	DeclEnum         DeclKind = "enum"
	DeclClass        DeclKind = "class"
	DeclVolume       DeclKind = "volume"
	DeclChapter      DeclKind = "chapter"
	DeclBeat         DeclKind = "beat"
	DeclStartPattern DeclKind = "start_pattern"
	DeclPlace        DeclKind = "place"
	DeclCharacter    DeclKind = "character"
	DeclCollective   DeclKind = "collective"
	DeclFaction      DeclKind = "faction"
	DeclObject       DeclKind = "object"
	DeclFact         DeclKind = "fact"
	DeclPromise      DeclKind = "promise"
	DeclThread       DeclKind = "thread"
	DeclArc          DeclKind = "arc"
	DeclInvariant    DeclKind = "invariant"
	DeclStyleNote    DeclKind = "style_note"
)

type Decl struct {
	Kind    DeclKind
	Name    string
	Code    string
	Alias   string
	Anchor  *Expr
	In      *Expr
	Class   *Expr
	Value   *Expr
	Fields  []Field
	Members []string
	Span    source.Span
}

type Field struct {
	Name       string
	Value      *Expr
	Statements []Stmt
	Span       source.Span
}
