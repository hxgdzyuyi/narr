package structure

import (
	"fmt"
	"sort"
	"strconv"
	"strings"

	"narr/internal/ast"
	"narr/internal/model"
	"narr/internal/resolve"
	"narr/internal/source"
	"narr/internal/state"
)

type QueryResult struct {
	Value EvalValue `json:"value"`
}

type queryEvaluator struct {
	index       *Index
	env         resolve.FileEnv
	locals      []map[string]EvalValue
	diagnostics []source.Diagnostic
}

func EvalQuery(index *Index, env resolve.FileEnv, expr *ast.Expr) (QueryResult, []source.Diagnostic) {
	return EvalQueryWithLocals(index, env, expr, nil)
}

func EvalQueryWithLocals(index *Index, env resolve.FileEnv, expr *ast.Expr, locals map[string]EvalValue) (QueryResult, []source.Diagnostic) {
	evaluator := &queryEvaluator{index: index, env: env}
	if len(locals) > 0 {
		evaluator.locals = append(evaluator.locals, cloneLocals(locals))
	}
	value := evaluator.eval(expr)
	return QueryResult{Value: value}, evaluator.diagnostics
}

func EvalViewQuery(index *Index, env resolve.FileEnv, expr *ast.Expr) (QueryResult, []source.Diagnostic) {
	return EvalQuery(index, env, expr)
}

func EvalBinder(index *Index, env resolve.FileEnv, binder *ast.Binder, locals map[string]EvalValue) ([]EvalValue, []source.Diagnostic) {
	evaluator := &queryEvaluator{index: index, env: env}
	if len(locals) > 0 {
		evaluator.locals = append(evaluator.locals, cloneLocals(locals))
	}
	return evaluator.evalBinder(binder), evaluator.diagnostics
}

func (e *queryEvaluator) eval(expr *ast.Expr) EvalValue {
	if expr == nil {
		return EvalMissingValue()
	}
	switch expr.Kind {
	case ast.ExprInvalid:
		return EvalMissingValue()
	case ast.ExprString, ast.ExprMultiline:
		return EvalStringValue(expr.Value)
	case ast.ExprInteger:
		value, err := strconv.Atoi(expr.Value)
		if err != nil {
			e.addError("E0701", expr.Span, fmt.Sprintf("invalid integer %s", expr.Value))
			return EvalMissingValue()
		}
		return EvalIntValue(value)
	case ast.ExprBool:
		return EvalBoolValue(expr.Value == "true")
	case ast.ExprRef, ast.ExprPath:
		return e.evalRefPath(expr)
	case ast.ExprList:
		items := make([]EvalValue, 0, len(expr.Children))
		for _, child := range expr.Children {
			items = append(items, e.eval(child))
		}
		return EvalListValue(items)
	case ast.ExprSet:
		items := make([]EvalValue, 0, len(expr.Children))
		for _, child := range expr.Children {
			items = append(items, e.eval(child))
		}
		return EvalSetValue(items)
	case ast.ExprLength:
		if len(expr.Children) == 0 {
			return EvalMissingValue()
		}
		return e.eval(expr.Children[0])
	case ast.ExprUnary:
		if expr.Op == "not" && len(expr.Children) == 1 {
			return EvalBoolValue(!EvalTruth(e.eval(expr.Children[0])))
		}
	case ast.ExprBinary:
		return e.evalBinary(expr)
	case ast.ExprPostfix:
		return e.evalPostfix(expr)
	case ast.ExprCall:
		return e.evalCall(expr)
	case ast.ExprMember:
		return e.evalMember(expr)
	case ast.ExprCollection:
		return e.collection(expr.Value, expr.Span)
	case ast.ExprCount:
		return e.evalCount(expr)
	case ast.ExprCollect:
		return e.evalCollect(expr)
	case ast.ExprState:
		return e.evalState(expr)
	case ast.ExprParen:
		if len(expr.Children) == 0 {
			return EvalMissingValue()
		}
		return e.eval(expr.Children[0])
	}
	e.addError("E0702", expr.Span, fmt.Sprintf("unsupported expression %s", expr.Kind))
	return EvalMissingValue()
}

func (e *queryEvaluator) evalMember(expr *ast.Expr) EvalValue {
	if len(expr.Children) != 1 {
		return EvalMissingValue()
	}
	base := e.eval(expr.Children[0])
	if expr.Value == "" {
		return base
	}
	return e.evalPropertyPath(base, strings.Split(expr.Value, "."), expr.Span)
}

func (e *queryEvaluator) evalPostfix(expr *ast.Expr) EvalValue {
	if len(expr.Children) != 1 {
		return EvalMissingValue()
	}
	value := e.eval(expr.Children[0])
	switch expr.Op {
	case "exists":
		return EvalBoolValue(value.Kind != EvalMissing)
	case "missing":
		return EvalBoolValue(value.Kind == EvalMissing)
	default:
		e.addError("E0703", expr.Span, fmt.Sprintf("unsupported postfix operator %s", expr.Op))
		return EvalMissingValue()
	}
}

func (e *queryEvaluator) evalBinary(expr *ast.Expr) EvalValue {
	if len(expr.Children) < 2 {
		return EvalMissingValue()
	}
	switch expr.Op {
	case "and":
		return EvalBoolValue(EvalTruth(e.eval(expr.Children[0])) && EvalTruth(e.eval(expr.Children[1])))
	case "or":
		return EvalBoolValue(EvalTruth(e.eval(expr.Children[0])) || EvalTruth(e.eval(expr.Children[1])))
	case "=>":
		return EvalBoolValue(!EvalTruth(e.eval(expr.Children[0])) || EvalTruth(e.eval(expr.Children[1])))
	case "==", "!=", "<", "<=", ">", ">=":
		left := e.eval(expr.Children[0])
		right := e.eval(expr.Children[1])
		return EvalBoolValue(compareEvalValues(left, right, expr.Op))
	case "in", "not in":
		left := e.eval(expr.Children[0])
		right := e.eval(expr.Children[1])
		ok := evalValueIn(left, right)
		if expr.Op == "not in" {
			ok = !ok
		}
		return EvalBoolValue(ok)
	case "precedes", "at_or_before", "at_or_after", "between", "in_volume":
		return e.evalTemporalPredicate(expr)
	case "serves", "mentions", "sets_up", "pays_off", "starts", "advances", "resolves", "reveals", "changes":
		return e.evalNarrativePredicate(expr)
	default:
		e.addError("E0704", expr.Span, fmt.Sprintf("unsupported binary operator %s", expr.Op))
		return EvalMissingValue()
	}
}

func (e *queryEvaluator) evalRefPath(expr *ast.Expr) EvalValue {
	if value, ok := e.local(expr.Value); ok {
		return value
	}
	parts := strings.Split(expr.Value, ".")
	if len(parts) > 1 {
		if value, ok := e.local(parts[0]); ok {
			return e.evalPropertyPath(value, parts[1:], expr.Span)
		}
	}
	if expr.Value == "beginning" || expr.Value == "end_of_story" || hasAnchorSuffix(parts) {
		anchor, diagnostics := e.index.Timeline.AnchorFromExpr(e.env, expr)
		if len(diagnostics) == 0 {
			return EvalAnchorValue(anchor)
		}
		if hasAnchorSuffix(parts) {
			e.diagnostics = append(e.diagnostics, diagnostics...)
			return EvalMissingValue()
		}
	}
	for split := len(parts); split >= 1; split-- {
		prefix := strings.Join(parts[:split], ".")
		symbol, _ := e.index.Resolved.ResolveName(e.env, prefix, expr.Span, false)
		if symbol == nil {
			continue
		}
		consumed := e.consumedPathSegments(parts, split, symbol)
		if consumed < split {
			split = consumed
		}
		value := EvalRefValue(model.SymbolIDFor(symbol.Namespace, symbol.Name))
		if consumed >= len(parts) {
			return value
		}
		return e.evalPropertyPath(value, parts[consumed:], expr.Span)
	}
	if expr.Kind == ast.ExprRef {
		return EvalSymbolValue(expr.Value)
	}
	symbol, diagnostics := e.index.Resolved.ResolveName(e.env, expr.Value, expr.Span, true)
	e.diagnostics = append(e.diagnostics, diagnostics...)
	if symbol != nil {
		return EvalRefValue(model.SymbolIDFor(symbol.Namespace, symbol.Name))
	}
	return EvalMissingValue()
}

func (e *queryEvaluator) consumedPathSegments(parts []string, split int, symbol *resolve.Symbol) int {
	if len(parts) > 0 {
		if namespace, ok := e.env.Imports[parts[0]]; ok && namespace == symbol.Namespace {
			consumed := 1 + pathSegmentCount(symbol.Name)
			if consumed <= len(parts) {
				return consumed
			}
		}
	}
	nameParts := strings.Split(symbol.Name, ".")
	if hasPathPrefix(parts, nameParts) {
		return len(nameParts)
	}
	namespaceParts := strings.Split(symbol.Namespace, ".")
	qualifiedParts := append(append([]string(nil), namespaceParts...), nameParts...)
	if hasPathPrefix(parts, qualifiedParts) {
		return len(qualifiedParts)
	}
	return split
}

func pathSegmentCount(value string) int {
	if value == "" {
		return 0
	}
	return len(strings.Split(value, "."))
}

func hasPathPrefix(parts []string, prefix []string) bool {
	if len(prefix) > len(parts) {
		return false
	}
	for index, part := range prefix {
		if parts[index] != part {
			return false
		}
	}
	return true
}

func (e *queryEvaluator) evalPropertyPath(value EvalValue, parts []string, span source.Span) EvalValue {
	current := value
	for len(parts) > 0 {
		next, ok := e.evalProperty(current, parts[0], span)
		if !ok {
			return EvalMissingValue()
		}
		current = next
		parts = parts[1:]
	}
	return current
}

func (e *queryEvaluator) evalProperty(value EvalValue, name string, span source.Span) (EvalValue, bool) {
	if value.Kind == EvalObject {
		if property, ok := value.Obj[name]; ok {
			return property, true
		}
		e.addError("E0725", span, fmt.Sprintf("object has no property %s", name))
		return EvalMissingValue(), false
	}
	if value.Kind == EvalAnchor {
		switch name {
		case "chapter":
			code, ok := e.chapterCodeFromAnchor(value.Anchor)
			if !ok {
				break
			}
			return e.chapterRef(code, span), true
		}
	}
	if value.Kind != EvalRef {
		e.addError("E0705", span, fmt.Sprintf("%s has no property %s", value.String(), name))
		return EvalMissingValue(), false
	}
	if isAnchorSuffix(name) {
		anchor, ok := e.anchorForRef(value.Ref, name)
		if !ok {
			e.addError("E0706", span, fmt.Sprintf("%s does not support .%s", value.Ref, name))
			return EvalMissingValue(), false
		}
		return EvalAnchorValue(anchor), true
	}
	switch name {
	case "code":
		if code, ok := e.chapterCodeForRef(value.Ref); ok {
			return EvalSymbolValue(code), true
		}
		if code, ok := e.volumeCodeForRef(value.Ref); ok {
			return EvalSymbolValue(code), true
		}
	case "alias":
		if chapter, ok := e.chapterForRef(value.Ref); ok {
			if chapter.Alias == "" {
				return EvalMissingValue(), true
			}
			return EvalSymbolValue(chapter.Alias), true
		}
		if volume, ok := e.volumeForRef(value.Ref); ok {
			if volume.Alias == "" {
				return EvalMissingValue(), true
			}
			return EvalSymbolValue(volume.Alias), true
		}
	case "name":
		return EvalSymbolValue(symbolName(value.Ref)), true
	}
	field, decl, ok := e.fieldForRef(value.Ref, name)
	if !ok {
		return EvalMissingValue(), true
	}
	return e.evalFieldValue(decl, field), true
}

func (e *queryEvaluator) evalFieldValue(decl *ast.Decl, field model.FieldValue) EvalValue {
	env := e.index.envForDecl(decl)
	switch field.Name {
	case "at", "setup_at", "payoff_by", "payoff_at", "starts_at", "expected_resolution", "resolved_at", "active_until":
		anchor, diagnostics := e.index.Timeline.AnchorFromExpr(env, field.Value)
		e.diagnostics = append(e.diagnostics, diagnostics...)
		if len(diagnostics) > 0 {
			return EvalMissingValue()
		}
		return EvalAnchorValue(anchor)
	case "hidden":
		if field.Value == nil || len(field.Value.Children) != 2 {
			return EvalMissingValue()
		}
		return e.withEnv(env, field.Value.Children[0])
	default:
		if field.Value != nil {
			return e.withEnv(env, field.Value)
		}
	}
	return EvalMissingValue()
}

func (e *queryEvaluator) withEnv(env resolve.FileEnv, expr *ast.Expr) EvalValue {
	previous := e.env
	e.env = env
	value := e.eval(expr)
	e.env = previous
	return value
}

func (e *queryEvaluator) evalCall(expr *ast.Expr) EvalValue {
	switch expr.Value {
	case "chapters_in":
		if len(expr.Args) != 1 {
			return e.callArity(expr, 1)
		}
		volume, ok := e.volumeCodeFromValue(e.eval(expr.Args[0]))
		if !ok {
			e.addError("E0708", expr.Args[0].Span, "chapters_in expects a volume")
			return EvalMissingValue()
		}
		var items []EvalValue
		for _, code := range e.index.Timeline.OrderedCodes {
			chapter := e.index.Project.Chapters[code]
			if chapter != nil && chapter.Code.VolumeCode().Canonical() == volume {
				items = append(items, EvalRefValue(chapter.ID))
			}
		}
		return EvalListValue(items)
	case "beats":
		if len(expr.Args) != 1 {
			return e.callArity(expr, 1)
		}
		code, ok := e.chapterCodeFromValue(e.eval(expr.Args[0]))
		if !ok {
			e.addError("E0709", expr.Args[0].Span, "beats expects a chapter")
			return EvalMissingValue()
		}
		var items []EvalValue
		for _, step := range e.index.Timeline.OrderedBeats {
			if step.Chapter == code {
				items = append(items, EvalRefValue(step.Beat.ID))
			}
		}
		return EvalListValue(items)
	case "active_threads", "active_promises", "active_arcs", "served_threads", "served_promises", "served_arcs", "reveals_in", "mentions_in":
		return e.evalViewCall(expr)
	case "build":
		if len(expr.Args) != 1 {
			return e.callArity(expr, 1)
		}
		code, ok := e.chapterCodeFromValue(e.eval(expr.Args[0]))
		if !ok {
			e.addError("E0710", expr.Args[0].Span, "build expects a chapter")
			return EvalMissingValue()
		}
		return e.evalBuildObject(code, expr.Args[0].Span)
	case "canonical", "volume_of", "chapter_of", "previous", "next", "chapter_distance", "chapters_between":
		return e.evalBuiltin(expr)
	default:
		e.addError("E0711", expr.Span, fmt.Sprintf("unknown function %s", expr.Value))
		return EvalMissingValue()
	}
}

func (e *queryEvaluator) evalBuildObject(code string, span source.Span) EvalValue {
	view := e.index.View(code)
	chapter := e.index.Project.Chapters[code]
	if view == nil || chapter == nil {
		e.addError("E0726", span, fmt.Sprintf("unknown chapter %s", code))
		return EvalMissingValue()
	}
	previous, next := e.neighborChapters(code)
	volume := e.index.Project.Volumes[chapter.Code.VolumeCode().Canonical()]
	begin := state.Anchor{Kind: state.AnchorChapterBegin, Chapter: code}
	end := state.Anchor{Kind: state.AnchorChapterEnd, Chapter: code}
	beginStore, _ := e.index.Timeline.StoreAt(begin)
	endStore, _ := e.index.Timeline.StoreAt(end)
	return EvalObjectValue(map[string]EvalValue{
		"chapter": EvalObjectValue(map[string]EvalValue{
			"code":             EvalSymbolValue(chapter.Decl.Code),
			"canonical_code":   EvalSymbolValue(code),
			"alias":            missingOrSymbol(chapter.Alias),
			"title":            e.buildFieldValue(chapter.Decl, chapter.Fields, "title"),
			"purpose":          e.buildFieldValue(chapter.Decl, chapter.Fields, "purpose"),
			"target_length":    e.buildFieldValue(chapter.Decl, chapter.Fields, "target_length"),
			"volume_code":      EvalSymbolValue(chapter.Code.VolumeCode().Canonical()),
			"previous_chapter": missingOrSymbol(previous),
			"next_chapter":     missingOrSymbol(next),
		}),
		"summary": EvalObjectValue(map[string]EvalValue{
			"novel_summary":   e.novelSummaryValue(),
			"volume_summary":  e.volumeSummaryValue(volume),
			"chapter_summary": e.buildFieldValue(chapter.Decl, chapter.Fields, "summary"),
		}),
		"context": EvalObjectValue(map[string]EvalValue{
			"relevant_characters": refsList(e.buildContextRefs(chapter, view, resolve.SymbolCharacter)),
			"relevant_places":     refsList(e.buildContextRefs(chapter, view, resolve.SymbolPlace)),
			"relevant_objects":    refsList(e.buildContextRefs(chapter, view, resolve.SymbolObject)),
			"relevant_facts":      refsList(e.buildContextRefs(chapter, view, resolve.SymbolFact)),
		}),
		"state": EvalObjectValue(map[string]EvalValue{
			"state_at_chapter_begin": EvalObjectValue(evalStoreMap(beginStore)),
			"expected_state_changes": e.evalStateChanges(beginStore, endStore),
			"state_at_chapter_end":   EvalObjectValue(evalStoreMap(endStore)),
		}),
		"structure": EvalObjectValue(map[string]EvalValue{
			"start_patterns":  refsList(view.StartPatterns),
			"active_threads":  refsList(view.ActiveThreads),
			"active_promises": refsList(view.ActivePromises),
			"active_arcs":     refsList(view.ActiveArcs),
			"served_threads":  refsList(view.ServedThreads),
			"served_promises": refsList(view.ServedPromises),
			"served_arcs":     refsList(view.ServedArcs),
		}),
		"beats": EvalObjectValue(map[string]EvalValue{
			"ordered_beats":      e.beatRefsForChapter(code),
			"beat_preconditions": e.evalBeatPreconditions(code),
			"beat_effects":       e.evalBeatEffects(code),
			"beat_render_hints":  e.evalBeatRenderHints(code),
		}),
		"prose": EvalObjectValue(map[string]EvalValue{
			"prose_hint":    e.buildFieldValue(chapter.Decl, chapter.Fields, "prose_hint"),
			"target_length": e.buildFieldValue(chapter.Decl, chapter.Fields, "target_length"),
		}),
	})
}

func (e *queryEvaluator) buildFieldValue(decl *ast.Decl, fields []model.FieldValue, name string) EvalValue {
	field, ok := fieldByName(fields, name)
	if !ok || field.Value == nil {
		return EvalMissingValue()
	}
	return e.withEnv(e.index.envForDecl(decl), field.Value)
}

func (e *queryEvaluator) novelSummaryValue() EvalValue {
	if e.index.Project.Novel == nil {
		return EvalMissingValue()
	}
	return e.buildFieldValue(e.index.Project.Novel.Decl, e.index.Project.Novel.Fields, "summary")
}

func (e *queryEvaluator) volumeSummaryValue(volume *model.Volume) EvalValue {
	if volume == nil {
		return EvalMissingValue()
	}
	return e.buildFieldValue(volume.Decl, volume.Fields, "summary")
}

func (e *queryEvaluator) neighborChapters(code string) (string, string) {
	for index, candidate := range e.index.Timeline.OrderedCodes {
		if candidate != code {
			continue
		}
		previous := ""
		next := ""
		if index > 0 {
			previous = e.index.Timeline.OrderedCodes[index-1]
		}
		if index+1 < len(e.index.Timeline.OrderedCodes) {
			next = e.index.Timeline.OrderedCodes[index+1]
		}
		return previous, next
	}
	return "", ""
}

func (e *queryEvaluator) buildContextRefs(chapter *model.Chapter, view *ChapterView, kind resolve.SymbolKind) []model.SymbolID {
	refs := map[model.SymbolID]bool{}
	add := func(ids ...model.SymbolID) {
		for _, id := range ids {
			if id != "" {
				refs[id] = true
			}
		}
	}
	add(view.ActiveThreads...)
	add(view.ActivePromises...)
	add(view.ActiveArcs...)
	add(view.ServedThreads...)
	add(view.ServedPromises...)
	add(view.ServedArcs...)
	add(view.Reveals...)
	add(view.Mentions...)
	for id := range refs {
		e.collectBuildStructureRefs(id, refs)
	}
	e.collectBuildFields(e.index.envForDecl(chapter.Decl), chapter.Fields, refs)
	for _, step := range e.index.Timeline.OrderedBeats {
		if step.Chapter != chapter.Code.Canonical() {
			continue
		}
		env := e.index.envForDecl(step.Beat.Decl)
		e.collectBuildFields(env, step.Beat.Fields, refs)
		for _, stmt := range step.Beat.Effects {
			e.collectBuildExprRefs(env, stmt.Target, refs)
			e.collectBuildExprRefs(env, stmt.Value, refs)
		}
	}
	ids := make([]model.SymbolID, 0, len(refs))
	for id := range refs {
		if e.symbolKind(id) == kind {
			ids = append(ids, id)
		}
	}
	sort.Slice(ids, func(a, b int) bool { return ids[a] < ids[b] })
	return ids
}

func (e *queryEvaluator) collectBuildStructureRefs(id model.SymbolID, refs map[model.SymbolID]bool) {
	if thread := e.index.Project.Threads[id]; thread != nil {
		e.collectBuildFields(e.index.envForDecl(thread.Decl), thread.Fields, refs)
	}
	if promise := e.index.Project.Promises[id]; promise != nil {
		e.collectBuildFields(e.index.envForDecl(promise.Decl), promise.Fields, refs)
	}
	if arc := e.index.Project.Arcs[id]; arc != nil {
		e.collectBuildFields(e.index.envForDecl(arc.Decl), arc.Fields, refs)
	}
}

func (e *queryEvaluator) collectBuildFields(env resolve.FileEnv, fields []model.FieldValue, refs map[model.SymbolID]bool) {
	for _, field := range fields {
		e.collectBuildExprRefs(env, field.Value, refs)
		for _, stmt := range field.Statements {
			e.collectBuildExprRefs(env, stmt.Target, refs)
			e.collectBuildExprRefs(env, stmt.Value, refs)
			e.collectBuildExprRefs(env, stmt.Expr, refs)
		}
	}
}

func (e *queryEvaluator) collectBuildExprRefs(env resolve.FileEnv, expr *ast.Expr, refs map[model.SymbolID]bool) {
	if expr == nil {
		return
	}
	if expr.Kind == ast.ExprRef || expr.Kind == ast.ExprPath {
		symbol, _ := e.index.Resolved.ResolveName(env, expr.Value, expr.Span, false)
		if symbol != nil {
			refs[model.SymbolIDFor(symbol.Namespace, symbol.Name)] = true
		}
	}
	for _, child := range expr.Children {
		e.collectBuildExprRefs(env, child, refs)
	}
	for _, arg := range expr.Args {
		e.collectBuildExprRefs(env, arg, refs)
	}
}

func (e *queryEvaluator) evalStateChanges(begin, end state.Store) EvalValue {
	keys := map[state.FieldKey]bool{}
	for _, key := range begin.Keys() {
		keys[key] = true
	}
	for _, key := range end.Keys() {
		keys[key] = true
	}
	ordered := make([]state.FieldKey, 0, len(keys))
	for key := range keys {
		ordered = append(ordered, key)
	}
	sort.Slice(ordered, func(a, b int) bool { return ordered[a].String() < ordered[b].String() })
	var items []EvalValue
	for _, key := range ordered {
		before := EvalValueFromState(begin.Get(key))
		after := EvalValueFromState(end.Get(key))
		if before.StableKey() == after.StableKey() {
			continue
		}
		items = append(items, EvalObjectValue(map[string]EvalValue{
			"field":  EvalStringValue(key.String()),
			"before": before,
			"after":  after,
		}))
	}
	return EvalListValue(items)
}

func (e *queryEvaluator) evalBeatPreconditions(code string) EvalValue {
	var items []EvalValue
	for _, step := range e.index.Timeline.OrderedBeats {
		if step.Chapter != code {
			continue
		}
		for _, field := range step.Beat.Fields {
			if field.Name != "precondition" {
				continue
			}
			for _, stmt := range field.Statements {
				items = append(items, EvalObjectValue(map[string]EvalValue{
					"beat":      EvalRefValue(step.Beat.ID),
					"condition": EvalStringValue(exprDisplay(stmt.Expr)),
				}))
			}
		}
	}
	return EvalListValue(items)
}

func (e *queryEvaluator) evalBeatEffects(code string) EvalValue {
	var items []EvalValue
	for _, step := range e.index.Timeline.OrderedBeats {
		if step.Chapter != code {
			continue
		}
		var effects []EvalValue
		for _, stmt := range step.Beat.Effects {
			effects = append(effects, EvalObjectValue(map[string]EvalValue{
				"target": EvalStringValue(exprDisplay(stmt.Target)),
				"op":     EvalStringValue(stmt.Op),
				"value":  EvalStringValue(exprDisplay(stmt.Value)),
			}))
		}
		items = append(items, EvalObjectValue(map[string]EvalValue{
			"beat":    EvalRefValue(step.Beat.ID),
			"effects": EvalListValue(effects),
		}))
	}
	return EvalListValue(items)
}

func (e *queryEvaluator) evalBeatRenderHints(code string) EvalValue {
	var items []EvalValue
	for _, step := range e.index.Timeline.OrderedBeats {
		if step.Chapter != code {
			continue
		}
		for _, field := range step.Beat.Fields {
			if field.Name != "render_hint" {
				continue
			}
			items = append(items, EvalObjectValue(map[string]EvalValue{
				"beat": EvalRefValue(step.Beat.ID),
				"hint": EvalStringValue(exprDisplay(field.Value)),
			}))
		}
	}
	return EvalListValue(items)
}

func (e *queryEvaluator) beatRefsForChapter(code string) EvalValue {
	var items []EvalValue
	for _, step := range e.index.Timeline.OrderedBeats {
		if step.Chapter == code {
			items = append(items, EvalRefValue(step.Beat.ID))
		}
	}
	return EvalListValue(items)
}

func (e *queryEvaluator) evalViewCall(expr *ast.Expr) EvalValue {
	if len(expr.Args) != 1 {
		return e.callArity(expr, 1)
	}
	code, ok := e.chapterCodeFromValue(e.eval(expr.Args[0]))
	if !ok {
		e.addError("E0712", expr.Args[0].Span, fmt.Sprintf("%s expects a chapter", expr.Value))
		return EvalMissingValue()
	}
	view := e.index.View(code)
	if view == nil {
		e.addError("E0713", expr.Args[0].Span, fmt.Sprintf("unknown chapter %s", code))
		return EvalMissingValue()
	}
	var ids []model.SymbolID
	switch expr.Value {
	case "active_threads":
		ids = view.ActiveThreads
	case "active_promises":
		ids = view.ActivePromises
	case "active_arcs":
		ids = view.ActiveArcs
	case "served_threads":
		ids = view.ServedThreads
	case "served_promises":
		ids = view.ServedPromises
	case "served_arcs":
		ids = view.ServedArcs
	case "reveals_in":
		ids = view.Reveals
	case "mentions_in":
		ids = view.Mentions
	}
	return refsList(ids)
}

func (e *queryEvaluator) evalCount(expr *ast.Expr) EvalValue {
	if expr.Binder != nil {
		items := e.evalBinder(expr.Binder)
		count := 0
		for _, item := range items {
			e.pushLocal(expr.Binder.Name, item)
			include := true
			if expr.Binder.Where != nil {
				include = EvalTruth(e.eval(expr.Binder.Where))
			}
			e.popLocal()
			if include {
				count++
			}
		}
		return EvalIntValue(count)
	}
	if len(expr.Children) != 1 {
		return EvalIntValue(0)
	}
	value := e.eval(expr.Children[0])
	if value.Kind == EvalList || value.Kind == EvalSet {
		return EvalIntValue(len(value.Items))
	}
	e.addError("E0904", expr.Children[0].Span, "count target must be a collection")
	return EvalMissingValue()
}

func (e *queryEvaluator) evalCollect(expr *ast.Expr) EvalValue {
	if expr.Binder == nil || len(expr.Children) != 1 {
		return EvalListValue(nil)
	}
	items := e.evalBinder(expr.Binder)
	values := make([]EvalValue, 0, len(items))
	for _, item := range items {
		e.pushLocal(expr.Binder.Name, item)
		include := true
		if expr.Binder.Where != nil {
			include = EvalTruth(e.eval(expr.Binder.Where))
		}
		if include {
			values = append(values, e.eval(expr.Children[0]))
		}
		e.popLocal()
	}
	return EvalListValue(values)
}

func (e *queryEvaluator) evalBinder(binder *ast.Binder) []EvalValue {
	var collection EvalValue
	if binder.In != nil {
		collection = e.eval(binder.In)
	} else {
		collection = e.collection(domainCollectionName(binder.Domain), binder.Span)
	}
	if collection.Kind != EvalList && collection.Kind != EvalSet {
		e.addError("E0714", binder.Span, fmt.Sprintf("binder %s expects a collection", binder.Name))
		return nil
	}
	out := make([]EvalValue, 0, len(collection.Items))
	for _, item := range collection.Items {
		if !e.domainMatches(binder.Domain, item) {
			e.addError("E0904", binder.Span, fmt.Sprintf("binder %s expects %s values", binder.Name, binder.Domain))
			return nil
		}
		out = append(out, item)
	}
	return out
}

func (e *queryEvaluator) evalState(expr *ast.Expr) EvalValue {
	if len(expr.Children) != 2 {
		e.addError("E0715", expr.Span, "state(...) expects a field and an anchor")
		return EvalMissingValue()
	}
	if expr.Children[0] == nil {
		return EvalMissingValue()
	}
	key, _, ok := e.index.Timeline.LookupStateField(e.env, expr.Children[0].Value, expr.Children[0].Span)
	if !ok {
		e.addError("E0715", expr.Children[0].Span, fmt.Sprintf("unknown state field %s", expr.Children[0].Value))
		return EvalMissingValue()
	}
	anchorValue := e.eval(expr.Children[1])
	anchor, ok := e.anchorFromValue(anchorValue)
	if !ok {
		e.addError("E0715", expr.Children[1].Span, "state anchor must be a chapter boundary")
		return EvalMissingValue()
	}
	if anchor.Kind != state.AnchorChapterBegin && anchor.Kind != state.AnchorChapterEnd {
		e.addError("E0715", expr.Children[1].Span, "state anchor must be chapter.begin or chapter.end")
		return EvalMissingValue()
	}
	store, ok := e.index.Timeline.StoreAt(anchor)
	if !ok {
		e.addError("E0715", expr.Children[1].Span, "unknown state checkpoint")
		return EvalMissingValue()
	}
	return EvalValueFromState(store.Get(key))
}

func (e *queryEvaluator) collection(name string, span source.Span) EvalValue {
	switch name {
	case "novels":
		if e.index.Project.Novel == nil {
			return EvalListValue(nil)
		}
		return EvalListValue([]EvalValue{EvalRefValue(e.index.Project.Novel.ID)})
	case "volumes":
		var codes []string
		for code := range e.index.Project.Volumes {
			codes = append(codes, code)
		}
		sort.Slice(codes, func(a, b int) bool {
			return e.index.Project.Volumes[codes[a]].Code.Compare(e.index.Project.Volumes[codes[b]].Code) < 0
		})
		items := make([]EvalValue, 0, len(codes))
		for _, code := range codes {
			items = append(items, EvalRefValue(e.index.Project.Volumes[code].ID))
		}
		return EvalListValue(items)
	case "chapters":
		items := make([]EvalValue, 0, len(e.index.Timeline.OrderedCodes))
		for _, code := range e.index.Timeline.OrderedCodes {
			items = append(items, EvalRefValue(e.index.Project.Chapters[code].ID))
		}
		return EvalListValue(items)
	case "beats":
		items := make([]EvalValue, 0, len(e.index.Timeline.OrderedBeats))
		for _, step := range e.index.Timeline.OrderedBeats {
			items = append(items, EvalRefValue(step.Beat.ID))
		}
		return EvalListValue(items)
	case "threads":
		return refsList(sortedMapIDs(e.index.Project.Threads))
	case "promises":
		return refsList(sortedMapIDs(e.index.Project.Promises))
	case "arcs":
		return refsList(sortedMapIDs(e.index.Project.Arcs))
	case "characters":
		return refsList(sortedEntityIDs(e.index.Project.Entities.Characters))
	case "places":
		return refsList(sortedEntityIDs(e.index.Project.Entities.Places))
	case "objects":
		return refsList(sortedEntityIDs(e.index.Project.Entities.Objects))
	case "facts":
		return refsList(sortedEntityIDs(e.index.Project.Entities.Facts))
	default:
		e.addError("E0716", span, fmt.Sprintf("unknown collection %s", name))
		return EvalMissingValue()
	}
}

func (e *queryEvaluator) evalBuiltin(expr *ast.Expr) EvalValue {
	switch expr.Value {
	case "canonical":
		if len(expr.Args) != 1 {
			return e.callArity(expr, 1)
		}
		value := e.eval(expr.Args[0])
		if value.Kind == EvalRef || value.Kind == EvalAnchor {
			return value
		}
		return EvalSymbolValue(value.String())
	case "volume_of":
		if len(expr.Args) != 1 {
			return e.callArity(expr, 1)
		}
		volume, ok := e.volumeCodeFromValue(e.eval(expr.Args[0]))
		if !ok {
			e.addError("E0717", expr.Args[0].Span, "volume_of expects a chapter, beat, anchor, or volume")
			return EvalMissingValue()
		}
		vol := e.index.Project.Volumes[volume]
		if vol == nil {
			return EvalSymbolValue(volume)
		}
		return EvalRefValue(vol.ID)
	case "chapter_of":
		if len(expr.Args) != 1 {
			return e.callArity(expr, 1)
		}
		code, ok := e.chapterCodeFromValue(e.eval(expr.Args[0]))
		if !ok {
			e.addError("E0718", expr.Args[0].Span, "chapter_of expects a chapter, beat, or anchor")
			return EvalMissingValue()
		}
		return e.chapterRef(code, expr.Args[0].Span)
	case "previous", "next":
		if len(expr.Args) != 1 {
			return e.callArity(expr, 1)
		}
		code, ok := e.chapterCodeFromValue(e.eval(expr.Args[0]))
		if !ok {
			e.addError("E0719", expr.Args[0].Span, fmt.Sprintf("%s expects a chapter", expr.Value))
			return EvalMissingValue()
		}
		index := e.chapterIndex(code)
		if index < 0 {
			return EvalMissingValue()
		}
		if expr.Value == "previous" {
			index--
		} else {
			index++
		}
		if index < 0 || index >= len(e.index.Timeline.OrderedCodes) {
			return EvalMissingValue()
		}
		return e.chapterRef(e.index.Timeline.OrderedCodes[index], expr.Args[0].Span)
	case "chapter_distance":
		if len(expr.Args) != 2 {
			return e.callArity(expr, 2)
		}
		left, leftOK := e.chapterCodeFromValue(e.eval(expr.Args[0]))
		right, rightOK := e.chapterCodeFromValue(e.eval(expr.Args[1]))
		if !leftOK || !rightOK {
			e.addError("E0720", expr.Span, "chapter_distance expects two chapter-like values")
			return EvalMissingValue()
		}
		distance := e.chapterIndex(right) - e.chapterIndex(left)
		if distance < 0 {
			distance = -distance
		}
		return EvalIntValue(distance)
	case "chapters_between":
		if len(expr.Args) != 2 {
			return e.callArity(expr, 2)
		}
		left, leftOK := e.chapterCodeFromValue(e.eval(expr.Args[0]))
		right, rightOK := e.chapterCodeFromValue(e.eval(expr.Args[1]))
		if !leftOK || !rightOK {
			e.addError("E0721", expr.Span, "chapters_between expects two chapter-like values")
			return EvalMissingValue()
		}
		startIndex := e.chapterIndex(left)
		endIndex := e.chapterIndex(right)
		if startIndex < 0 || endIndex < 0 {
			return EvalListValue(nil)
		}
		if startIndex > endIndex {
			startIndex, endIndex = endIndex, startIndex
		}
		items := make([]EvalValue, 0, endIndex-startIndex+1)
		for _, code := range e.index.Timeline.OrderedCodes[startIndex : endIndex+1] {
			items = append(items, e.chapterRef(code, expr.Span))
		}
		return EvalListValue(items)
	default:
		e.addError("E0722", expr.Span, fmt.Sprintf("unknown builtin %s", expr.Value))
		return EvalMissingValue()
	}
}

func (e *queryEvaluator) evalTemporalPredicate(expr *ast.Expr) EvalValue {
	switch expr.Op {
	case "in_volume":
		left, ok := e.volumeCodeFromValue(e.eval(expr.Children[0]))
		if !ok {
			return EvalBoolValue(false)
		}
		right, ok := e.volumeCodeFromValue(e.eval(expr.Children[1]))
		return EvalBoolValue(ok && left == right)
	case "between":
		if len(expr.Children) != 3 {
			return EvalBoolValue(false)
		}
		left, leftOK := e.anchorFromValue(e.eval(expr.Children[0]))
		first, firstOK := e.anchorFromValue(e.eval(expr.Children[1]))
		second, secondOK := e.anchorFromValue(e.eval(expr.Children[2]))
		if !leftOK || !firstOK || !secondOK {
			return EvalBoolValue(false)
		}
		pos := e.anchorPosition(left)
		firstPos := e.anchorPosition(first)
		secondPos := e.anchorPosition(second)
		if firstPos > secondPos {
			firstPos, secondPos = secondPos, firstPos
		}
		return EvalBoolValue(pos >= firstPos && pos <= secondPos)
	default:
		left, leftOK := e.anchorFromValue(e.eval(expr.Children[0]))
		right, rightOK := e.anchorFromValue(e.eval(expr.Children[1]))
		if !leftOK || !rightOK {
			return EvalBoolValue(false)
		}
		leftPos := e.anchorPosition(left)
		rightPos := e.anchorPosition(right)
		switch expr.Op {
		case "precedes":
			return EvalBoolValue(leftPos < rightPos)
		case "at_or_before":
			return EvalBoolValue(leftPos <= rightPos)
		case "at_or_after":
			return EvalBoolValue(leftPos >= rightPos)
		default:
			return EvalBoolValue(false)
		}
	}
}

func (e *queryEvaluator) evalNarrativePredicate(expr *ast.Expr) EvalValue {
	subject := e.eval(expr.Children[0])
	if subject.Kind != EvalRef {
		return EvalBoolValue(false)
	}
	targetExpr := expr.Children[1]
	target := e.eval(targetExpr)
	if expr.Op == "changes" {
		if targetExpr == nil || (targetExpr.Kind != ast.ExprRef && targetExpr.Kind != ast.ExprPath) {
			e.addError("E0905", expr.Span, "changes target must be a state field")
			return EvalBoolValue(false)
		}
		if _, _, ok := e.index.Timeline.LookupStateField(e.env, targetExpr.Value, targetExpr.Span); !ok {
			e.addError("E0905", targetExpr.Span, "changes target must be a state field")
			return EvalBoolValue(false)
		}
		return EvalBoolValue(e.subjectChanges(subject.Ref, targetExpr, expr.Children[2:]))
	}
	if target.Kind != EvalRef {
		e.addError("E0905", targetExpr.Span, fmt.Sprintf("%s target must be a reference", expr.Op))
		return EvalBoolValue(false)
	}
	if !e.narrativeTargetAllowed(expr.Op, target.Ref) {
		e.addError("E0905", targetExpr.Span, fmt.Sprintf("%s target has type %s", expr.Op, e.symbolKind(target.Ref)))
		return EvalBoolValue(false)
	}
	switch expr.Op {
	case "serves":
		return EvalBoolValue(e.subjectServes(subject.Ref, target.Ref))
	case "mentions":
		return EvalBoolValue(e.subjectMentions(subject.Ref, target.Ref))
	case "sets_up":
		return EvalBoolValue(e.subjectSetsUp(subject.Ref, target.Ref))
	case "pays_off":
		return EvalBoolValue(e.subjectPaysOff(subject.Ref, target.Ref))
	case "starts":
		return EvalBoolValue(e.subjectStarts(subject.Ref, target.Ref))
	case "advances":
		return EvalBoolValue(e.subjectAdvances(subject.Ref, target.Ref))
	case "resolves":
		return EvalBoolValue(e.subjectResolves(subject.Ref, target.Ref))
	case "reveals":
		return EvalBoolValue(e.subjectReveals(subject.Ref, target.Ref))
	default:
		return EvalBoolValue(false)
	}
}

func (e *queryEvaluator) narrativeTargetAllowed(op string, target model.SymbolID) bool {
	switch op {
	case "serves":
		switch e.symbolKind(target) {
		case resolve.SymbolThread, resolve.SymbolPromise, resolve.SymbolArc, resolve.SymbolStartPattern:
			return true
		default:
			return false
		}
	case "mentions":
		return true
	case "sets_up", "pays_off":
		return e.symbolKind(target) == resolve.SymbolPromise
	case "starts":
		switch e.symbolKind(target) {
		case resolve.SymbolThread, resolve.SymbolArc:
			return true
		default:
			return false
		}
	case "advances":
		switch e.symbolKind(target) {
		case resolve.SymbolThread, resolve.SymbolArc:
			return true
		default:
			return false
		}
	case "resolves":
		return e.symbolKind(target) == resolve.SymbolThread
	case "reveals":
		return e.symbolKind(target) == resolve.SymbolFact
	default:
		return true
	}
}

func (e *queryEvaluator) subjectServes(subject, target model.SymbolID) bool {
	if code, ok := e.chapterCodeForRef(subject); ok {
		view := e.index.View(code)
		if view == nil {
			return false
		}
		switch e.symbolKind(target) {
		case resolve.SymbolThread:
			return containsID(view.ServedThreads, target)
		case resolve.SymbolPromise:
			return containsID(view.ServedPromises, target)
		case resolve.SymbolArc:
			return containsID(view.ServedArcs, target)
		case resolve.SymbolStartPattern:
			return e.startPatternInChapter(target, code)
		default:
			return false
		}
	}
	if beat := e.index.Project.Beats[subject]; beat != nil {
		return e.beatServes(beat, target)
	}
	return false
}

func (e *queryEvaluator) subjectMentions(subject, target model.SymbolID) bool {
	if code, ok := e.chapterCodeForRef(subject); ok {
		return containsID(e.index.View(code).Mentions, target)
	}
	if beat := e.index.Project.Beats[subject]; beat != nil {
		return e.beatHasLink(beat, "mentions", target)
	}
	return false
}

func (e *queryEvaluator) subjectSetsUp(subject, target model.SymbolID) bool {
	if e.symbolKind(target) != resolve.SymbolPromise {
		return false
	}
	if code, ok := e.chapterCodeForRef(subject); ok {
		promise := e.index.promises[target]
		if promise != nil && promise.HasSetup {
			if setupCode, ok := e.index.chapterForAnchor(promise.Setup); ok && setupCode == code {
				return true
			}
		}
		return e.chapterStartPatternsTarget(code, target) || e.chapterHasLink(code, "sets_up", target)
	}
	if beat := e.index.Project.Beats[subject]; beat != nil {
		return e.beatHasLink(beat, "sets_up", target)
	}
	return false
}

func (e *queryEvaluator) subjectPaysOff(subject, target model.SymbolID) bool {
	if e.symbolKind(target) != resolve.SymbolPromise {
		return false
	}
	if code, ok := e.chapterCodeForRef(subject); ok {
		promise := e.index.promises[target]
		if promise != nil && promise.HasPayoffAt {
			if payoffCode, ok := e.index.chapterForAnchor(promise.PayoffAt); ok && payoffCode == code {
				return true
			}
		}
		return e.chapterHasLink(code, "pays_off", target)
	}
	if beat := e.index.Project.Beats[subject]; beat != nil {
		return e.beatHasLink(beat, "pays_off", target)
	}
	return false
}

func (e *queryEvaluator) subjectStarts(subject, target model.SymbolID) bool {
	if code, ok := e.chapterCodeForRef(subject); ok {
		return e.chapterStartPatternsTarget(code, target)
	}
	if beat := e.index.Project.Beats[subject]; beat != nil {
		return e.beatStartPatternsTarget(beat, target)
	}
	return false
}

func (e *queryEvaluator) subjectAdvances(subject, target model.SymbolID) bool {
	if code, ok := e.chapterCodeForRef(subject); ok {
		if e.chapterHasLink(code, "advances", target) {
			return true
		}
		return e.symbolKind(target) == resolve.SymbolArc && e.chapterChangesArc(code, target)
	}
	if beat := e.index.Project.Beats[subject]; beat != nil {
		if e.beatHasLink(beat, "advances", target) {
			return true
		}
		return e.symbolKind(target) == resolve.SymbolArc && e.beatChangesArc(beat, target)
	}
	return false
}

func (e *queryEvaluator) subjectResolves(subject, target model.SymbolID) bool {
	if e.symbolKind(target) != resolve.SymbolThread {
		return false
	}
	if code, ok := e.chapterCodeForRef(subject); ok {
		thread := e.index.threads[target]
		if thread != nil && thread.HasResolvedAt {
			if resolvedCode, ok := e.index.chapterForAnchor(thread.ResolvedAt); ok && resolvedCode == code {
				return true
			}
		}
		return e.chapterHasLink(code, "resolves", target)
	}
	if beat := e.index.Project.Beats[subject]; beat != nil {
		return e.beatHasLink(beat, "resolves", target)
	}
	return false
}

func (e *queryEvaluator) subjectReveals(subject, target model.SymbolID) bool {
	if e.symbolKind(target) != resolve.SymbolFact {
		return false
	}
	if code, ok := e.chapterCodeForRef(subject); ok {
		return containsID(e.index.View(code).Reveals, target)
	}
	if beat := e.index.Project.Beats[subject]; beat != nil {
		return e.beatHasLink(beat, "reveals", target)
	}
	return false
}

func (e *queryEvaluator) subjectChanges(subject model.SymbolID, fieldExpr *ast.Expr, filters []*ast.Expr) bool {
	if fieldExpr == nil || (fieldExpr.Kind != ast.ExprRef && fieldExpr.Kind != ast.ExprPath) {
		return false
	}
	key, fieldType, ok := e.index.Timeline.LookupStateField(e.env, fieldExpr.Value, fieldExpr.Span)
	if !ok {
		return false
	}
	if code, ok := e.chapterCodeForRef(subject); ok {
		for _, step := range e.index.Timeline.OrderedBeats {
			if step.Chapter == code && e.storeChangeMatches(step.Before, step.After, key, fieldType, filters) {
				return true
			}
		}
		return false
	}
	if beat := e.index.Project.Beats[subject]; beat != nil {
		before, beforeOK := e.index.Timeline.StoreAt(state.Anchor{Kind: state.AnchorBeatBefore, Beat: beat.ID})
		after, afterOK := e.index.Timeline.StoreAt(state.Anchor{Kind: state.AnchorBeatAfter, Beat: beat.ID})
		return beforeOK && afterOK && e.storeChangeMatches(before, after, key, fieldType, filters)
	}
	return false
}

func (e *queryEvaluator) storeChangeMatches(before, after state.Store, key state.FieldKey, fieldType model.TypeRef, filters []*ast.Expr) bool {
	beforeValue := EvalValueFromState(before.Get(key))
	afterValue := EvalValueFromState(after.Get(key))
	if beforeValue.StableKey() == afterValue.StableKey() {
		return false
	}
	switch len(filters) {
	case 0:
		return true
	case 1:
		return afterValue.StableKey() == e.evalExpectedFieldValue(filters[0], fieldType).StableKey()
	case 2:
		return beforeValue.StableKey() == e.evalExpectedFieldValue(filters[0], fieldType).StableKey() &&
			afterValue.StableKey() == e.evalExpectedFieldValue(filters[1], fieldType).StableKey()
	default:
		return false
	}
}

func (e *queryEvaluator) evalExpectedFieldValue(expr *ast.Expr, fieldType model.TypeRef) EvalValue {
	return EvalValueFromState(e.index.Timeline.ValueFromExpr(e.env, expr, fieldType))
}

func (e *queryEvaluator) chapterHasLink(code, fieldName string, target model.SymbolID) bool {
	for _, step := range e.index.Timeline.OrderedBeats {
		if step.Chapter == code && e.beatHasLink(step.Beat, fieldName, target) {
			return true
		}
	}
	return false
}

func (e *queryEvaluator) beatHasLink(beat *model.Beat, fieldName string, target model.SymbolID) bool {
	env := e.index.envForDecl(beat.Decl)
	for _, field := range beat.Fields {
		if field.Name != fieldName {
			continue
		}
		if containsID(e.refsFromExpr(env, field.Value), target) {
			return true
		}
	}
	return false
}

func (e *queryEvaluator) beatServes(beat *model.Beat, target model.SymbolID) bool {
	switch e.symbolKind(target) {
	case resolve.SymbolPromise:
		return e.beatHasLink(beat, "sets_up", target) || e.beatHasLink(beat, "pays_off", target)
	case resolve.SymbolThread:
		return e.beatHasLink(beat, "advances", target) || e.beatHasLink(beat, "resolves", target)
	case resolve.SymbolArc:
		return e.beatHasLink(beat, "advances", target) || e.beatChangesArc(beat, target)
	default:
		return false
	}
}

func (e *queryEvaluator) chapterStartPatternsTarget(code string, target model.SymbolID) bool {
	for id, pattern := range e.index.startPatterns {
		if !pattern.HasAnchor {
			continue
		}
		patternCode, ok := e.index.chapterForAnchor(pattern.Anchor)
		if !ok || patternCode != code {
			continue
		}
		if e.startPatternStarts(id, target) {
			return true
		}
	}
	return false
}

func (e *queryEvaluator) beatStartPatternsTarget(beat *model.Beat, target model.SymbolID) bool {
	for id, pattern := range e.index.startPatterns {
		if !pattern.HasAnchor {
			continue
		}
		if (pattern.Anchor.Kind == state.AnchorBeatBefore || pattern.Anchor.Kind == state.AnchorBeatAfter) && pattern.Anchor.Beat == beat.ID {
			if e.startPatternStarts(id, target) {
				return true
			}
		}
	}
	return false
}

func (e *queryEvaluator) startPatternInChapter(patternID model.SymbolID, code string) bool {
	pattern := e.index.startPatterns[patternID]
	if pattern == nil || !pattern.HasAnchor {
		return false
	}
	patternCode, ok := e.index.chapterForAnchor(pattern.Anchor)
	return ok && patternCode == code
}

func (e *queryEvaluator) startPatternStarts(patternID, target model.SymbolID) bool {
	pattern := e.index.startPatterns[patternID]
	if pattern == nil {
		return false
	}
	return containsID(pattern.ThreadIDs, target) || containsID(pattern.PromiseIDs, target) || containsID(pattern.ArcIDs, target)
}

func (e *queryEvaluator) chapterChangesArc(code string, arcID model.SymbolID) bool {
	for _, step := range e.index.Timeline.OrderedBeats {
		if step.Chapter == code && e.beatChangesArc(step.Beat, arcID) {
			return true
		}
	}
	return false
}

func (e *queryEvaluator) beatChangesArc(beat *model.Beat, arcID model.SymbolID) bool {
	arc := e.index.arcs[arcID]
	if arc == nil || !arc.HasSubject || !arc.HasStateField {
		return false
	}
	env := e.index.envForDecl(beat.Decl)
	for _, stmt := range beat.Effects {
		if stmt.Target == nil {
			continue
		}
		key, _, ok := e.index.Timeline.LookupStateField(env, stmt.Target.Value, stmt.Target.Span)
		if ok && key == arc.FieldKey {
			return true
		}
	}
	return false
}

func (e *queryEvaluator) refsFromExpr(env resolve.FileEnv, expr *ast.Expr) []model.SymbolID {
	if expr == nil {
		return nil
	}
	switch expr.Kind {
	case ast.ExprList, ast.ExprSet:
		var out []model.SymbolID
		for _, child := range expr.Children {
			out = append(out, e.refsFromExpr(env, child)...)
		}
		return out
	case ast.ExprParen:
		var out []model.SymbolID
		for _, child := range expr.Children {
			out = append(out, e.refsFromExpr(env, child)...)
		}
		return out
	case ast.ExprRef, ast.ExprPath:
		symbol, diagnostics := e.index.Resolved.ResolveName(env, expr.Value, expr.Span, true)
		e.diagnostics = append(e.diagnostics, diagnostics...)
		if symbol == nil {
			return nil
		}
		return []model.SymbolID{model.SymbolIDFor(symbol.Namespace, symbol.Name)}
	default:
		return nil
	}
}

func (e *queryEvaluator) pushLocal(name string, value EvalValue) {
	e.locals = append(e.locals, map[string]EvalValue{name: value})
}

func (e *queryEvaluator) popLocal() {
	e.locals = e.locals[:len(e.locals)-1]
}

func (e *queryEvaluator) local(name string) (EvalValue, bool) {
	for index := len(e.locals) - 1; index >= 0; index-- {
		if value, ok := e.locals[index][name]; ok {
			return value, true
		}
	}
	return EvalMissingValue(), false
}

func (e *queryEvaluator) callArity(expr *ast.Expr, want int) EvalValue {
	e.addError("E0723", expr.Span, fmt.Sprintf("%s expects %d arguments", expr.Value, want))
	return EvalMissingValue()
}

func (e *queryEvaluator) addError(code string, span source.Span, message string) {
	e.diagnostics = append(e.diagnostics, source.Error(code, span.Start.File, span.Start.Line, span.Start.Column, message))
}

func cloneLocals(locals map[string]EvalValue) map[string]EvalValue {
	out := map[string]EvalValue{}
	for key, value := range locals {
		out[key] = value
	}
	return out
}

func compareEvalValues(left, right EvalValue, op string) bool {
	switch op {
	case "==":
		return left.StableKey() == right.StableKey()
	case "!=":
		return left.StableKey() != right.StableKey()
	}
	if left.Kind == EvalInt && right.Kind == EvalInt {
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

func evalValueIn(value, collection EvalValue) bool {
	if collection.Kind != EvalList && collection.Kind != EvalSet {
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

func refsList(ids []model.SymbolID) EvalValue {
	items := make([]EvalValue, 0, len(ids))
	for _, id := range ids {
		items = append(items, EvalRefValue(id))
	}
	return EvalListValue(items)
}

func hasAnchorSuffix(parts []string) bool {
	if len(parts) == 0 {
		return false
	}
	return isAnchorSuffix(parts[len(parts)-1])
}

func isAnchorSuffix(value string) bool {
	switch value {
	case "begin", "end", "before", "after":
		return true
	default:
		return false
	}
}

func domainCollectionName(domain string) string {
	switch domain {
	case "novel":
		return "novels"
	case "volume":
		return "volumes"
	case "chapter":
		return "chapters"
	case "beat":
		return "beats"
	case "thread":
		return "threads"
	case "promise":
		return "promises"
	case "arc":
		return "arcs"
	case "character":
		return "characters"
	case "place":
		return "places"
	case "object":
		return "objects"
	case "fact":
		return "facts"
	default:
		return domain + "s"
	}
}

func sortedMapIDs[T any](values map[model.SymbolID]T) []model.SymbolID {
	ids := make([]model.SymbolID, 0, len(values))
	for id := range values {
		ids = append(ids, id)
	}
	sort.Slice(ids, func(a, b int) bool { return ids[a] < ids[b] })
	return ids
}

func sortedEntityIDs(values map[model.SymbolID]*model.Entity) []model.SymbolID {
	ids := make([]model.SymbolID, 0, len(values))
	for id := range values {
		ids = append(ids, id)
	}
	sort.Slice(ids, func(a, b int) bool { return ids[a] < ids[b] })
	return ids
}

func symbolName(id model.SymbolID) string {
	parts := strings.Split(string(id), ".")
	if len(parts) == 0 {
		return string(id)
	}
	return parts[len(parts)-1]
}

func missingOrSymbol(value string) EvalValue {
	if value == "" {
		return EvalMissingValue()
	}
	return EvalSymbolValue(value)
}

func evalStoreMap(store state.Store) map[string]EvalValue {
	out := map[string]EvalValue{}
	for _, key := range store.Keys() {
		out[key.String()] = EvalValueFromState(store.Get(key))
	}
	return out
}

func exprDisplay(expr *ast.Expr) string {
	if expr == nil {
		return ""
	}
	switch expr.Kind {
	case ast.ExprString, ast.ExprMultiline, ast.ExprInteger, ast.ExprBool, ast.ExprRef, ast.ExprPath, ast.ExprSymbol, ast.ExprLanguage:
		return expr.Value
	case ast.ExprLength:
		if len(expr.Children) > 0 {
			return exprDisplay(expr.Children[0]) + " " + expr.Value
		}
		return expr.Value
	case ast.ExprList:
		parts := make([]string, 0, len(expr.Children))
		for _, child := range expr.Children {
			parts = append(parts, exprDisplay(child))
		}
		return "[" + strings.Join(parts, ", ") + "]"
	case ast.ExprSet:
		parts := make([]string, 0, len(expr.Children))
		for _, child := range expr.Children {
			parts = append(parts, exprDisplay(child))
		}
		return "{" + strings.Join(parts, ", ") + "}"
	case ast.ExprBinary:
		parts := make([]string, 0, len(expr.Children))
		for _, child := range expr.Children {
			parts = append(parts, exprDisplay(child))
		}
		return strings.Join(parts, " "+expr.Op+" ")
	case ast.ExprUnary:
		if len(expr.Children) > 0 {
			return expr.Op + " " + exprDisplay(expr.Children[0])
		}
	case ast.ExprParen:
		if len(expr.Children) > 0 {
			return "(" + exprDisplay(expr.Children[0]) + ")"
		}
	case ast.ExprMember:
		if len(expr.Children) > 0 {
			return exprDisplay(expr.Children[0]) + "." + expr.Value
		}
	}
	if expr.Value != "" {
		return expr.Value
	}
	return string(expr.Kind)
}
