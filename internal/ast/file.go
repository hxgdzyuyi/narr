package ast

import "narr/internal/source"

type FileMode string

const (
	ModeNarr FileMode = "narr"
	ModeTest FileMode = "test"
)

type File struct {
	Path      string
	Mode      FileMode
	Namespace string
	Imports   []ImportDecl
	Decls     []Decl
	Tests     []TestDecl
	Span      source.Span
}

type ImportDecl struct {
	Path  string
	Alias string
	Span  source.Span
}
