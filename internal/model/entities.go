package model

import (
	"narr/internal/ast"
)

type FieldDef struct {
	Name    string
	Type    TypeRef
	Default *ast.Expr
	Span    astSpan
}

type Class struct {
	ID     SymbolID
	Name   string
	Fields map[string]FieldDef
	Decl   *ast.Decl
}

type Enum struct {
	ID      SymbolID
	Name    string
	Members map[string]bool
	Decl    *ast.Decl
}

type EntityKind string

const (
	EntityPlace      EntityKind = "place"
	EntityCharacter  EntityKind = "character"
	EntityCollective EntityKind = "collective"
	EntityFaction    EntityKind = "faction"
	EntityObject     EntityKind = "object"
	EntityFact       EntityKind = "fact"
)

type Entity struct {
	ID           SymbolID
	Name         string
	Kind         EntityKind
	ClassName    string
	Fields       map[string]FieldDef
	Initializers []ast.Stmt
	Value        *ast.Expr
	Decl         *ast.Decl
}

type Entities struct {
	Enums       map[SymbolID]*Enum
	Classes     map[SymbolID]*Class
	Places      map[SymbolID]*Entity
	Characters  map[SymbolID]*Entity
	Collectives map[SymbolID]*Entity
	Factions    map[SymbolID]*Entity
	Objects     map[SymbolID]*Entity
	Facts       map[SymbolID]*Entity
	All         map[SymbolID]*Entity
}

func NewEntities() *Entities {
	return &Entities{
		Enums:       map[SymbolID]*Enum{},
		Classes:     map[SymbolID]*Class{},
		Places:      map[SymbolID]*Entity{},
		Characters:  map[SymbolID]*Entity{},
		Collectives: map[SymbolID]*Entity{},
		Factions:    map[SymbolID]*Entity{},
		Objects:     map[SymbolID]*Entity{},
		Facts:       map[SymbolID]*Entity{},
		All:         map[SymbolID]*Entity{},
	}
}

func (e *Entities) AddEntity(entity *Entity) {
	e.All[entity.ID] = entity
	switch entity.Kind {
	case EntityPlace:
		e.Places[entity.ID] = entity
	case EntityCharacter:
		e.Characters[entity.ID] = entity
	case EntityCollective:
		e.Collectives[entity.ID] = entity
	case EntityFaction:
		e.Factions[entity.ID] = entity
	case EntityObject:
		e.Objects[entity.ID] = entity
	case EntityFact:
		e.Facts[entity.ID] = entity
	}
}
