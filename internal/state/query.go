package state

import (
	"fmt"
	"strings"

	"narr/internal/ast"
	"narr/internal/model"
	"narr/internal/resolve"
	"narr/internal/source"
)

func (t *Timeline) StateExpr(env resolve.FileEnv, expr *ast.Expr) (Value, []source.Diagnostic) {
	if expr == nil || expr.Kind != ast.ExprState || len(expr.Children) != 2 {
		return Missing(), []source.Diagnostic{source.Error("E0503", "", 0, 0, "expected state(...) expression")}
	}
	key, ok := t.fieldKeyFromRaw(env, expr.Children[0].Value, expr.Children[0].Span)
	if !ok {
		return Missing(), t.diagnostics
	}
	anchor, diagnostics := t.AnchorFromExpr(env, expr.Children[1])
	if len(diagnostics) > 0 {
		return Missing(), diagnostics
	}
	store, ok := t.storeAt(anchor)
	if !ok {
		return Missing(), []source.Diagnostic{source.Error("E0504", expr.Children[1].Span.Start.File, expr.Children[1].Span.Start.Line, expr.Children[1].Span.Start.Column, "unknown state checkpoint")}
	}
	return store.Get(key), nil
}

func (t *Timeline) AnchorFromExpr(env resolve.FileEnv, expr *ast.Expr) (Anchor, []source.Diagnostic) {
	if expr == nil {
		return Anchor{}, []source.Diagnostic{source.Error("E0505", "", 0, 0, "missing anchor")}
	}
	if expr.Kind != ast.ExprRef && expr.Kind != ast.ExprPath {
		return Anchor{}, []source.Diagnostic{source.Error("E0506", expr.Span.Start.File, expr.Span.Start.Line, expr.Span.Start.Column, "anchor must be a reference")}
	}
	if expr.Value == "beginning" {
		return Anchor{Kind: AnchorBeginning}, nil
	}
	if expr.Value == "end_of_story" {
		return Anchor{Kind: AnchorEndOfStory}, nil
	}

	parts := strings.Split(expr.Value, ".")
	suffix := ""
	switch parts[len(parts)-1] {
	case "begin", "end", "before", "after":
		suffix = parts[len(parts)-1]
		parts = parts[:len(parts)-1]
	}
	base := strings.Join(parts, ".")
	symbol, diagnostics := t.Resolved.ResolveName(env, base, expr.Span, true)
	if len(diagnostics) > 0 {
		return Anchor{}, diagnostics
	}
	switch symbol.Kind {
	case resolve.SymbolVolume:
		code, err := model.ParseVolumeCode(symbol.Name)
		if err != nil {
			return Anchor{}, []source.Diagnostic{source.Error("E0511", expr.Span.Start.File, expr.Span.Start.Line, expr.Span.Start.Column, err.Error())}
		}
		if suffix == "" {
			suffix = "begin"
		}
		if suffix != "begin" && suffix != "end" {
			return Anchor{}, []source.Diagnostic{source.Error("E0512", expr.Span.Start.File, expr.Span.Start.Line, expr.Span.Start.Column, fmt.Sprintf("volume anchor does not support .%s", suffix))}
		}
		anchor, ok := t.volumeBoundary(code, suffix)
		if !ok {
			return Anchor{}, []source.Diagnostic{source.Error("E0513", expr.Span.Start.File, expr.Span.Start.Line, expr.Span.Start.Column, fmt.Sprintf("volume %s contains no chapters", code.Canonical()))}
		}
		return anchor, nil
	case resolve.SymbolChapter:
		code, err := model.ParseChapterCode(symbol.Name)
		if err != nil {
			return Anchor{}, []source.Diagnostic{source.Error("E0507", expr.Span.Start.File, expr.Span.Start.Line, expr.Span.Start.Column, err.Error())}
		}
		if suffix == "" {
			suffix = "begin"
		}
		if suffix != "begin" && suffix != "end" {
			return Anchor{}, []source.Diagnostic{source.Error("E0508", expr.Span.Start.File, expr.Span.Start.Line, expr.Span.Start.Column, fmt.Sprintf("chapter anchor does not support .%s", suffix))}
		}
		return t.ChapterBoundary(code.Canonical(), suffix), nil
	case resolve.SymbolBeat:
		if suffix == "" {
			suffix = "before"
		}
		if suffix != "before" && suffix != "after" {
			return Anchor{}, []source.Diagnostic{source.Error("E0509", expr.Span.Start.File, expr.Span.Start.Line, expr.Span.Start.Column, fmt.Sprintf("beat anchor does not support .%s", suffix))}
		}
		kind := AnchorBeatBefore
		if suffix == "after" {
			kind = AnchorBeatAfter
		}
		return Anchor{Kind: kind, Beat: model.SymbolIDFor(symbol.Namespace, symbol.Name)}, nil
	default:
		return Anchor{}, []source.Diagnostic{source.Error("E0510", expr.Span.Start.File, expr.Span.Start.Line, expr.Span.Start.Column, fmt.Sprintf("%s is not a chapter or beat anchor", expr.Value))}
	}
}

func (t *Timeline) volumeBoundary(volume model.VolumeCode, suffix string) (Anchor, bool) {
	first := ""
	last := ""
	for _, code := range t.OrderedCodes {
		chapter := t.Project.Chapters[code]
		if chapter == nil || chapter.Code.VolumeCode().Compare(volume) != 0 {
			continue
		}
		if first == "" {
			first = code
		}
		last = code
	}
	if first == "" {
		if suffix == "end" {
			return Anchor{Kind: AnchorEndOfStory}, true
		}
		return Anchor{Kind: AnchorBeginning}, true
	}
	if suffix == "end" {
		return Anchor{Kind: AnchorChapterEnd, Chapter: last}, true
	}
	return Anchor{Kind: AnchorChapterBegin, Chapter: first}, true
}
