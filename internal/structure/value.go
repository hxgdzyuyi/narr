package structure

import (
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"strings"

	"narr/internal/model"
	"narr/internal/state"
)

type EvalValueKind string

const (
	EvalNull    EvalValueKind = "Null"
	EvalMissing EvalValueKind = "Missing"
	EvalBool    EvalValueKind = "Bool"
	EvalInt     EvalValueKind = "Int"
	EvalString  EvalValueKind = "String"
	EvalSymbol  EvalValueKind = "Symbol"
	EvalRef     EvalValueKind = "Ref"
	EvalAnchor  EvalValueKind = "Anchor"
	EvalList    EvalValueKind = "List"
	EvalSet     EvalValueKind = "Set"
	EvalObject  EvalValueKind = "Object"
)

type EvalValue struct {
	Kind   EvalValueKind        `json:"kind"`
	Bool   bool                 `json:"bool,omitempty"`
	Int    int                  `json:"int,omitempty"`
	Text   string               `json:"text,omitempty"`
	Ref    model.SymbolID       `json:"ref,omitempty"`
	Anchor state.Anchor         `json:"anchor,omitempty"`
	Items  []EvalValue          `json:"items,omitempty"`
	Obj    map[string]EvalValue `json:"object,omitempty"`
}

func (v EvalValue) MarshalJSON() ([]byte, error) {
	out := map[string]any{"kind": v.Kind}
	switch v.Kind {
	case EvalBool:
		out["bool"] = v.Bool
	case EvalInt:
		out["int"] = v.Int
	case EvalString, EvalSymbol:
		out["text"] = v.Text
	case EvalRef:
		out["ref"] = v.Ref
	case EvalAnchor:
		out["anchor"] = v.Anchor
	case EvalList, EvalSet:
		out["items"] = v.Items
	case EvalObject:
		out["object"] = v.Obj
	}
	return json.Marshal(out)
}

func EvalNullValue() EvalValue {
	return EvalValue{Kind: EvalNull}
}

func EvalMissingValue() EvalValue {
	return EvalValue{Kind: EvalMissing}
}

func EvalBoolValue(value bool) EvalValue {
	return EvalValue{Kind: EvalBool, Bool: value}
}

func EvalIntValue(value int) EvalValue {
	return EvalValue{Kind: EvalInt, Int: value}
}

func EvalStringValue(value string) EvalValue {
	return EvalValue{Kind: EvalString, Text: value}
}

func EvalSymbolValue(value string) EvalValue {
	return EvalValue{Kind: EvalSymbol, Text: value}
}

func EvalRefValue(value model.SymbolID) EvalValue {
	return EvalValue{Kind: EvalRef, Ref: value}
}

func EvalAnchorValue(value state.Anchor) EvalValue {
	return EvalValue{Kind: EvalAnchor, Anchor: value}
}

func EvalListValue(values []EvalValue) EvalValue {
	items := append([]EvalValue(nil), values...)
	return EvalValue{Kind: EvalList, Items: items}
}

func EvalSetValue(values []EvalValue) EvalValue {
	seen := map[string]EvalValue{}
	for _, value := range values {
		seen[value.StableKey()] = value
	}
	keys := make([]string, 0, len(seen))
	for key := range seen {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	items := make([]EvalValue, 0, len(keys))
	for _, key := range keys {
		items = append(items, seen[key])
	}
	return EvalValue{Kind: EvalSet, Items: items}
}

func EvalObjectValue(values map[string]EvalValue) EvalValue {
	out := map[string]EvalValue{}
	for key, value := range values {
		out[key] = value
	}
	return EvalValue{Kind: EvalObject, Obj: out}
}

func EvalValueFromState(value state.Value) EvalValue {
	switch value.Kind {
	case state.ValueNull:
		return EvalNullValue()
	case state.ValueMissing:
		return EvalMissingValue()
	case state.ValueBool:
		return EvalBoolValue(value.Bool)
	case state.ValueInt:
		return EvalIntValue(value.Int)
	case state.ValueString:
		return EvalStringValue(value.Text)
	case state.ValueSymbol:
		return EvalSymbolValue(value.Text)
	case state.ValueRef:
		return EvalRefValue(value.Ref)
	case state.ValueList:
		items := make([]EvalValue, 0, len(value.Items))
		for _, item := range value.Items {
			items = append(items, EvalValueFromState(item))
		}
		return EvalListValue(items)
	case state.ValueSet:
		items := make([]EvalValue, 0, len(value.Items))
		for _, item := range value.Items {
			items = append(items, EvalValueFromState(item))
		}
		return EvalSetValue(items)
	default:
		return EvalMissingValue()
	}
}

func (v EvalValue) StableKey() string {
	switch v.Kind {
	case EvalNull, EvalMissing:
		return string(v.Kind)
	case EvalBool:
		return "Bool:" + strconv.FormatBool(v.Bool)
	case EvalInt:
		return "Int:" + strconv.Itoa(v.Int)
	case EvalString:
		return "String:" + v.Text
	case EvalSymbol:
		return "Symbol:" + v.Text
	case EvalRef:
		return "Ref:" + string(v.Ref)
	case EvalAnchor:
		return "Anchor:" + anchorKey(v.Anchor)
	case EvalList, EvalSet:
		parts := make([]string, 0, len(v.Items))
		for _, item := range v.Items {
			parts = append(parts, item.StableKey())
		}
		if v.Kind == EvalSet {
			sort.Strings(parts)
		}
		return string(v.Kind) + ":[" + strings.Join(parts, ",") + "]"
	case EvalObject:
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

func (v EvalValue) String() string {
	switch v.Kind {
	case EvalNull, EvalMissing:
		return string(v.Kind)
	case EvalBool:
		return strconv.FormatBool(v.Bool)
	case EvalInt:
		return strconv.Itoa(v.Int)
	case EvalString:
		return strconv.Quote(v.Text)
	case EvalSymbol:
		return v.Text
	case EvalRef:
		return string(v.Ref)
	case EvalAnchor:
		return anchorKey(v.Anchor)
	case EvalList, EvalSet:
		parts := make([]string, 0, len(v.Items))
		for _, item := range v.Items {
			parts = append(parts, item.String())
		}
		open, close := "[", "]"
		if v.Kind == EvalSet {
			open, close = "{", "}"
		}
		return open + strings.Join(parts, ", ") + close
	case EvalObject:
		keys := make([]string, 0, len(v.Obj))
		for key := range v.Obj {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		parts := make([]string, 0, len(keys))
		for _, key := range keys {
			parts = append(parts, key+": "+v.Obj[key].String())
		}
		return "{" + strings.Join(parts, ", ") + "}"
	default:
		return v.StableKey()
	}
}

func EvalTruth(value EvalValue) bool {
	switch value.Kind {
	case EvalBool:
		return value.Bool
	case EvalMissing, EvalNull:
		return false
	case EvalList, EvalSet:
		return len(value.Items) > 0
	case EvalObject:
		return len(value.Obj) > 0
	default:
		return true
	}
}
