package structure

import (
	"strconv"

	"narr/internal/ast"
	"narr/internal/model"
	"narr/internal/resolve"
	"narr/internal/state"
)

func (i *Index) evalExpr(env resolve.FileEnv, store state.Store, expr *ast.Expr) state.Value {
	if expr == nil {
		return state.Missing()
	}
	switch expr.Kind {
	case ast.ExprString, ast.ExprMultiline:
		return state.String(expr.Value)
	case ast.ExprInteger:
		value, err := strconv.Atoi(expr.Value)
		if err != nil {
			return state.Missing()
		}
		return state.Int(value)
	case ast.ExprBool:
		return state.Bool(expr.Value == "true")
	case ast.ExprRef, ast.ExprPath:
		if key, _, ok := i.Timeline.LookupStateField(env, expr.Value, expr.Span); ok {
			return store.Get(key)
		}
		if symbol, _ := i.Resolved.ResolveName(env, expr.Value, expr.Span, false); symbol != nil {
			return state.Ref(model.SymbolIDFor(symbol.Namespace, symbol.Name))
		}
		return state.Symbol(expr.Value)
	case ast.ExprList:
		values := make([]state.Value, 0, len(expr.Children))
		for _, child := range expr.Children {
			values = append(values, i.evalExpr(env, store, child))
		}
		return state.List(values)
	case ast.ExprSet:
		values := make([]state.Value, 0, len(expr.Children))
		for _, child := range expr.Children {
			values = append(values, i.evalExpr(env, store, child))
		}
		return state.Set(values)
	case ast.ExprParen:
		if len(expr.Children) == 0 {
			return state.Missing()
		}
		return i.evalExpr(env, store, expr.Children[0])
	case ast.ExprUnary:
		if expr.Op == "not" && len(expr.Children) == 1 {
			return state.Bool(!truthy(i.evalExpr(env, store, expr.Children[0])))
		}
	case ast.ExprPostfix:
		if len(expr.Children) != 1 {
			return state.Missing()
		}
		value := i.evalExpr(env, store, expr.Children[0])
		switch expr.Op {
		case "exists":
			return state.Bool(value.Kind != state.ValueMissing)
		case "missing":
			return state.Bool(value.Kind == state.ValueMissing)
		}
	case ast.ExprBinary:
		return i.evalBinary(env, store, expr)
	}
	return state.Missing()
}

func (i *Index) evalBinary(env resolve.FileEnv, store state.Store, expr *ast.Expr) state.Value {
	if len(expr.Children) < 2 {
		return state.Missing()
	}
	switch expr.Op {
	case "and":
		return state.Bool(truthy(i.evalExpr(env, store, expr.Children[0])) && truthy(i.evalExpr(env, store, expr.Children[1])))
	case "or":
		return state.Bool(truthy(i.evalExpr(env, store, expr.Children[0])) || truthy(i.evalExpr(env, store, expr.Children[1])))
	case "=>":
		return state.Bool(!truthy(i.evalExpr(env, store, expr.Children[0])) || truthy(i.evalExpr(env, store, expr.Children[1])))
	case "==", "!=", "<", "<=", ">", ">=":
		left := i.evalExpr(env, store, expr.Children[0])
		right := i.evalExpr(env, store, expr.Children[1])
		return state.Bool(compareValues(left, right, expr.Op))
	case "in", "not in":
		left := i.evalExpr(env, store, expr.Children[0])
		right := i.evalExpr(env, store, expr.Children[1])
		ok := valueIn(left, right)
		if expr.Op == "not in" {
			ok = !ok
		}
		return state.Bool(ok)
	}
	return state.Missing()
}

func truthy(value state.Value) bool {
	switch value.Kind {
	case state.ValueBool:
		return value.Bool
	case state.ValueMissing, state.ValueNull:
		return false
	default:
		return true
	}
}

func compareValues(left, right state.Value, op string) bool {
	switch op {
	case "==":
		return left.StableKey() == right.StableKey()
	case "!=":
		return left.StableKey() != right.StableKey()
	}
	if left.Kind == state.ValueInt && right.Kind == state.ValueInt {
		switch op {
		case "<":
			return left.Int < right.Int
		case "<=":
			return left.Int <= right.Int
		case ">":
			return left.Int > right.Int
		case ">=":
			return left.Int >= right.Int
		}
	}
	return false
}

func valueIn(value, collection state.Value) bool {
	if collection.Kind != state.ValueSet && collection.Kind != state.ValueList {
		return false
	}
	key := value.StableKey()
	for _, item := range collection.Items {
		if item.StableKey() == key {
			return true
		}
	}
	return false
}
