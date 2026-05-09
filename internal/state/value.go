package state

import (
	"fmt"
	"sort"
	"strconv"
	"strings"

	"narr/internal/model"
)

type ValueKind string

const (
	ValueNull    ValueKind = "Null"
	ValueMissing ValueKind = "Missing"
	ValueBool    ValueKind = "Bool"
	ValueInt     ValueKind = "Int"
	ValueString  ValueKind = "String"
	ValueSymbol  ValueKind = "Symbol"
	ValueRef     ValueKind = "Ref"
	ValueList    ValueKind = "List"
	ValueSet     ValueKind = "Set"
	ValueObject  ValueKind = "Object"
)

type Value struct {
	Kind  ValueKind        `json:"kind"`
	Bool  bool             `json:"bool,omitempty"`
	Int   int              `json:"int,omitempty"`
	Text  string           `json:"text,omitempty"`
	Ref   model.SymbolID   `json:"ref,omitempty"`
	Items []Value          `json:"items,omitempty"`
	Obj   map[string]Value `json:"object,omitempty"`
}

func Null() Value {
	return Value{Kind: ValueNull}
}

func Missing() Value {
	return Value{Kind: ValueMissing}
}

func Bool(value bool) Value {
	return Value{Kind: ValueBool, Bool: value}
}

func Int(value int) Value {
	return Value{Kind: ValueInt, Int: value}
}

func String(value string) Value {
	return Value{Kind: ValueString, Text: value}
}

func Symbol(value string) Value {
	return Value{Kind: ValueSymbol, Text: value}
}

func Ref(value model.SymbolID) Value {
	return Value{Kind: ValueRef, Ref: value}
}

func List(values []Value) Value {
	items := append([]Value(nil), values...)
	return Value{Kind: ValueList, Items: items}
}

func Set(values []Value) Value {
	seen := map[string]Value{}
	for _, value := range values {
		seen[value.StableKey()] = value
	}
	keys := make([]string, 0, len(seen))
	for key := range seen {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	items := make([]Value, 0, len(keys))
	for _, key := range keys {
		items = append(items, seen[key])
	}
	return Value{Kind: ValueSet, Items: items}
}

func Object(values map[string]Value) Value {
	copy := map[string]Value{}
	for key, value := range values {
		copy[key] = value
	}
	return Value{Kind: ValueObject, Obj: copy}
}

func (v Value) StableKey() string {
	switch v.Kind {
	case ValueNull, ValueMissing:
		return string(v.Kind)
	case ValueBool:
		return "Bool:" + strconv.FormatBool(v.Bool)
	case ValueInt:
		return "Int:" + strconv.Itoa(v.Int)
	case ValueString:
		return "String:" + v.Text
	case ValueSymbol:
		return "Symbol:" + v.Text
	case ValueRef:
		return "Ref:" + string(v.Ref)
	case ValueList, ValueSet:
		parts := make([]string, 0, len(v.Items))
		for _, item := range v.Items {
			parts = append(parts, item.StableKey())
		}
		if v.Kind == ValueSet {
			sort.Strings(parts)
		}
		return string(v.Kind) + ":[" + strings.Join(parts, ",") + "]"
	case ValueObject:
		keys := make([]string, 0, len(v.Obj))
		for key := range v.Obj {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		parts := make([]string, 0, len(keys))
		for _, key := range keys {
			parts = append(parts, key+"="+v.Obj[key].StableKey())
		}
		return "Object:{" + strings.Join(parts, ",") + "}"
	default:
		return fmt.Sprintf("%s:%s", v.Kind, v.Text)
	}
}

func (v Value) String() string {
	switch v.Kind {
	case ValueNull, ValueMissing:
		return string(v.Kind)
	case ValueBool:
		return strconv.FormatBool(v.Bool)
	case ValueInt:
		return strconv.Itoa(v.Int)
	case ValueString:
		return strconv.Quote(v.Text)
	case ValueSymbol:
		return v.Text
	case ValueRef:
		return string(v.Ref)
	case ValueList, ValueSet:
		parts := make([]string, 0, len(v.Items))
		for _, item := range v.Items {
			parts = append(parts, item.String())
		}
		open, close := "[", "]"
		if v.Kind == ValueSet {
			open, close = "{", "}"
		}
		return open + strings.Join(parts, ", ") + close
	default:
		return v.StableKey()
	}
}
