package model

import "narr/internal/ast"

type FieldValue struct {
	Name       string
	Value      *ast.Expr
	Statements []ast.Stmt
}

type Novel struct {
	ID     SymbolID
	Name   string
	Fields []FieldValue
	Decl   *ast.Decl
}

type Volume struct {
	ID     SymbolID
	Code   VolumeCode
	Alias  string
	Fields []FieldValue
	Decl   *ast.Decl
}

type Chapter struct {
	ID     SymbolID
	Code   ChapterCode
	Alias  string
	Fields []FieldValue
	Decl   *ast.Decl
}

type Beat struct {
	ID      SymbolID
	Name    string
	Anchor  *ast.Expr
	Fields  []FieldValue
	Effects []ast.Stmt
	Decl    *ast.Decl
}

type StartPattern struct {
	ID     SymbolID
	Name   string
	Fields []FieldValue
	Decl   *ast.Decl
}

type Thread struct {
	ID     SymbolID
	Name   string
	Fields []FieldValue
	Decl   *ast.Decl
}

type Promise struct {
	ID     SymbolID
	Name   string
	Fields []FieldValue
	Decl   *ast.Decl
}

type Arc struct {
	ID     SymbolID
	Name   string
	Fields []FieldValue
	Decl   *ast.Decl
}

type Invariant struct {
	ID     SymbolID
	Name   string
	Fields []FieldValue
	Decl   *ast.Decl
}
