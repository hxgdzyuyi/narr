package model

import "strings"

type SymbolID string

type TypeKind string

const (
	TypeInvalid      TypeKind = "Invalid"
	TypeBool         TypeKind = "Bool"
	TypeInt          TypeKind = "Int"
	TypeString       TypeKind = "String"
	TypeSymbol       TypeKind = "Symbol"
	TypeText         TypeKind = "Text"
	TypeNovel        TypeKind = "Novel"
	TypeVolume       TypeKind = "Volume"
	TypeChapter      TypeKind = "Chapter"
	TypeBeat         TypeKind = "Beat"
	TypeThread       TypeKind = "Thread"
	TypePromise      TypeKind = "Promise"
	TypeArc          TypeKind = "Arc"
	TypeStartPattern TypeKind = "StartPattern"
	TypeInvariant    TypeKind = "Invariant"
	TypePlace        TypeKind = "Place"
	TypeCharacter    TypeKind = "Character"
	TypeObject       TypeKind = "Object"
	TypeFaction      TypeKind = "Faction"
	TypeFact         TypeKind = "Fact"
	TypeClaim        TypeKind = "Claim"
	TypeSet          TypeKind = "Set"
	TypeList         TypeKind = "List"
	TypeEnum         TypeKind = "Enum"
	TypeClass        TypeKind = "Class"
)

type TypeRef struct {
	Kind TypeKind
	Name string
	Elem *TypeRef
}

func BuiltinType(name string) (TypeRef, bool) {
	switch name {
	case "Bool":
		return TypeRef{Kind: TypeBool, Name: name}, true
	case "Int":
		return TypeRef{Kind: TypeInt, Name: name}, true
	case "String":
		return TypeRef{Kind: TypeString, Name: name}, true
	case "Symbol":
		return TypeRef{Kind: TypeSymbol, Name: name}, true
	case "Text":
		return TypeRef{Kind: TypeText, Name: name}, true
	case "Novel":
		return TypeRef{Kind: TypeNovel, Name: name}, true
	case "Volume":
		return TypeRef{Kind: TypeVolume, Name: name}, true
	case "Chapter":
		return TypeRef{Kind: TypeChapter, Name: name}, true
	case "Beat":
		return TypeRef{Kind: TypeBeat, Name: name}, true
	case "Thread":
		return TypeRef{Kind: TypeThread, Name: name}, true
	case "Promise":
		return TypeRef{Kind: TypePromise, Name: name}, true
	case "Arc":
		return TypeRef{Kind: TypeArc, Name: name}, true
	case "StartPattern":
		return TypeRef{Kind: TypeStartPattern, Name: name}, true
	case "Invariant":
		return TypeRef{Kind: TypeInvariant, Name: name}, true
	case "Place":
		return TypeRef{Kind: TypePlace, Name: name}, true
	case "Character":
		return TypeRef{Kind: TypeCharacter, Name: name}, true
	case "Object":
		return TypeRef{Kind: TypeObject, Name: name}, true
	case "Faction":
		return TypeRef{Kind: TypeFaction, Name: name}, true
	case "Fact":
		return TypeRef{Kind: TypeFact, Name: name}, true
	case "Claim":
		return TypeRef{Kind: TypeClaim, Name: name}, true
	default:
		return TypeRef{}, false
	}
}

func (t TypeRef) String() string {
	if t.Kind == TypeSet || t.Kind == TypeList {
		elem := "Invalid"
		if t.Elem != nil {
			elem = t.Elem.String()
		}
		return string(t.Kind) + "<" + elem + ">"
	}
	if t.Name != "" {
		return t.Name
	}
	return string(t.Kind)
}

func (t TypeRef) Same(other TypeRef) bool {
	if t.Kind != other.Kind || t.Name != other.Name {
		return false
	}
	if t.Elem == nil || other.Elem == nil {
		return t.Elem == nil && other.Elem == nil
	}
	return t.Elem.Same(*other.Elem)
}

func SymbolIDFor(namespace, name string) SymbolID {
	return SymbolID(namespace + "." + name)
}

func (id SymbolID) Namespace() string {
	text := string(id)
	index := strings.LastIndex(text, ".")
	if index < 0 {
		return ""
	}
	return text[:index]
}

func (id SymbolID) Name() string {
	text := string(id)
	index := strings.LastIndex(text, ".")
	if index < 0 {
		return text
	}
	return text[index+1:]
}
