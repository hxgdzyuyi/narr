package build

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"narr/internal/ast"
	outformat "narr/internal/format"
	"narr/internal/model"
	"narr/internal/resolve"
	"narr/internal/source"
	"narr/internal/state"
	"narr/internal/structure"
)

type ChapterBuild struct {
	Chapter   ChapterInfo   `json:"chapter"`
	Summary   SummaryInfo   `json:"summary"`
	Context   ContextInfo   `json:"context"`
	State     StateInfo     `json:"state"`
	Structure StructureInfo `json:"structure"`
	Beats     BeatsInfo     `json:"beats"`
	Prose     ProseInfo     `json:"prose"`
}

type ChapterInfo struct {
	Code            string `json:"code"`
	CanonicalCode   string `json:"canonical_code"`
	Alias           string `json:"alias,omitempty"`
	Title           string `json:"title,omitempty"`
	Purpose         string `json:"purpose,omitempty"`
	TargetLength    string `json:"target_length,omitempty"`
	VolumeCode      string `json:"volume_code"`
	PreviousChapter string `json:"previous_chapter,omitempty"`
	NextChapter     string `json:"next_chapter,omitempty"`
}

type SummaryInfo struct {
	NovelSummary   string `json:"novel_summary,omitempty"`
	VolumeSummary  string `json:"volume_summary,omitempty"`
	ChapterSummary string `json:"chapter_summary,omitempty"`
}

type ContextInfo struct {
	RelevantCharacters []model.SymbolID `json:"relevant_characters"`
	RelevantPlaces     []model.SymbolID `json:"relevant_places"`
	RelevantObjects    []model.SymbolID `json:"relevant_objects"`
	RelevantFacts      []model.SymbolID `json:"relevant_facts"`
}

type StateInfo struct {
	StateAtChapterBegin map[string]state.Value `json:"state_at_chapter_begin"`
	ExpectedChanges     []StateChange          `json:"expected_state_changes"`
	StateAtChapterEnd   map[string]state.Value `json:"state_at_chapter_end"`
}

type StateChange struct {
	Field  string      `json:"field"`
	Before state.Value `json:"before"`
	After  state.Value `json:"after"`
}

type StructureInfo struct {
	StartPatterns  []model.SymbolID `json:"start_patterns"`
	ActiveThreads  []model.SymbolID `json:"active_threads"`
	ActivePromises []model.SymbolID `json:"active_promises"`
	ActiveArcs     []model.SymbolID `json:"active_arcs"`
	ServedThreads  []model.SymbolID `json:"served_threads"`
	ServedPromises []model.SymbolID `json:"served_promises"`
	ServedArcs     []model.SymbolID `json:"served_arcs"`
}

type BeatsInfo struct {
	OrderedBeats      []model.SymbolID `json:"ordered_beats"`
	BeatPreconditions []BeatConditions `json:"beat_preconditions"`
	BeatEffects       []BeatEffects    `json:"beat_effects"`
	BeatRenderHints   []BeatRenderHint `json:"beat_render_hints"`
}

type BeatConditions struct {
	Beat      model.SymbolID `json:"beat"`
	Condition string         `json:"condition"`
}

type BeatEffects struct {
	Beat    model.SymbolID `json:"beat"`
	Effects []EffectInfo   `json:"effects"`
}

type EffectInfo struct {
	Target string `json:"target"`
	Op     string `json:"op"`
	Value  string `json:"value"`
}

type BeatRenderHint struct {
	Beat string `json:"beat"`
	Hint string `json:"hint"`
}

type ProseInfo struct {
	ProseHint    string `json:"prose_hint,omitempty"`
	TargetLength string `json:"target_length,omitempty"`
}

type Generator struct {
	Project  *model.Project
	Resolved *resolve.Project
	Timeline *state.Timeline
	Index    *structure.Index
}

func NewGenerator(project *model.Project, resolved *resolve.Project, timeline *state.Timeline, index *structure.Index) *Generator {
	return &Generator{Project: project, Resolved: resolved, Timeline: timeline, Index: index}
}

func (g *Generator) BuildChapter(raw string, env resolve.FileEnv, span source.Span) (*ChapterBuild, []source.Diagnostic) {
	code, diagnostics := g.resolveChapterCode(raw, env, span)
	if source.HasErrors(diagnostics) {
		return nil, diagnostics
	}
	chapter := g.Project.Chapters[code]
	if chapter == nil {
		return nil, []source.Diagnostic{source.Error("E0801", span.Start.File, span.Start.Line, span.Start.Column, fmt.Sprintf("unknown chapter %s", raw))}
	}
	view := g.Index.View(code)
	if view == nil {
		return nil, []source.Diagnostic{source.Error("E0802", span.Start.File, span.Start.Line, span.Start.Column, fmt.Sprintf("missing structure view for %s", code))}
	}

	begin := state.Anchor{Kind: state.AnchorChapterBegin, Chapter: code}
	end := state.Anchor{Kind: state.AnchorChapterEnd, Chapter: code}
	beginStore, beginOK := g.Timeline.StoreAt(begin)
	endStore, endOK := g.Timeline.StoreAt(end)
	if !beginOK || !endOK {
		return nil, []source.Diagnostic{source.Error("E0803", span.Start.File, span.Start.Line, span.Start.Column, fmt.Sprintf("missing state checkpoints for %s", code))}
	}

	build := &ChapterBuild{
		Chapter:   g.chapterInfo(chapter),
		Summary:   g.summaryInfo(chapter),
		Context:   g.contextInfo(chapter, view),
		State:     g.stateInfo(beginStore, endStore),
		Structure: g.structureInfo(view),
		Beats:     g.beatsInfo(code),
		Prose:     g.proseInfo(chapter),
	}
	return build, nil
}

func WriteJSON(path string, build *ChapterBuild) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()
	return outformat.JSON(file, build)
}

func WriteLLM(path string, build *ChapterBuild) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()
	_, err = file.WriteString(FormatLLM(build))
	return err
}

func OutputPath(outDir, code string) string {
	return filepath.Join(outDir, code+".build.json")
}

func LLMOutputPath(outDir, code string) string {
	return filepath.Join(outDir, code+".build.narr")
}

func FormatLLM(build *ChapterBuild) string {
	var builder strings.Builder
	writeLLMInstructions(&builder)
	fmt.Fprintf(&builder, "build %s {\n", build.Chapter.CanonicalCode)
	writeLLMChapter(&builder, build.Chapter)
	writeLLMSummary(&builder, build.Summary)
	writeLLMContext(&builder, build.Context)
	writeLLMState(&builder, build.State)
	writeLLMStructure(&builder, build.Structure)
	writeLLMBeats(&builder, build.Beats)
	writeLLMProse(&builder, build.Prose)
	builder.WriteString("}\n")
	return builder.String()
}

func (g *Generator) resolveChapterCode(raw string, env resolve.FileEnv, span source.Span) (string, []source.Diagnostic) {
	if code, err := model.ParseChapterCode(raw); err == nil {
		canonical := code.Canonical()
		if g.Project.Chapters[canonical] != nil {
			return canonical, nil
		}
	}
	symbol, diagnostics := g.Resolved.ResolveName(env, raw, span, true)
	if len(diagnostics) > 0 {
		return "", diagnostics
	}
	if symbol == nil || symbol.Kind != resolve.SymbolChapter {
		return "", []source.Diagnostic{source.Error("E0804", span.Start.File, span.Start.Line, span.Start.Column, fmt.Sprintf("%s is not a chapter", raw))}
	}
	code, err := model.ParseChapterCode(symbol.Name)
	if err != nil {
		return "", []source.Diagnostic{source.Error("E0805", span.Start.File, span.Start.Line, span.Start.Column, err.Error())}
	}
	return code.Canonical(), nil
}

func (g *Generator) chapterInfo(chapter *model.Chapter) ChapterInfo {
	code := chapter.Code.Canonical()
	info := ChapterInfo{
		Code:          chapter.Decl.Code,
		CanonicalCode: code,
		Alias:         chapter.Alias,
		Title:         textField(chapter.Fields, "title"),
		Purpose:       symbolField(chapter.Fields, "purpose"),
		TargetLength:  exprField(chapter.Fields, "target_length"),
		VolumeCode:    chapter.Code.VolumeCode().Canonical(),
	}
	if previous, next := g.neighborChapters(code); previous != "" || next != "" {
		info.PreviousChapter = previous
		info.NextChapter = next
	}
	return info
}

func (g *Generator) summaryInfo(chapter *model.Chapter) SummaryInfo {
	volume := g.Project.Volumes[chapter.Code.VolumeCode().Canonical()]
	info := SummaryInfo{
		ChapterSummary: textField(chapter.Fields, "summary"),
	}
	if g.Project.Novel != nil {
		info.NovelSummary = textField(g.Project.Novel.Fields, "summary")
	}
	if volume != nil {
		info.VolumeSummary = textField(volume.Fields, "summary")
	}
	return info
}

func (g *Generator) contextInfo(chapter *model.Chapter, view *structure.ChapterView) ContextInfo {
	refs := map[model.SymbolID]bool{}
	add := func(id model.SymbolID) {
		if id != "" {
			refs[id] = true
		}
	}
	for _, id := range view.ActiveThreads {
		add(id)
		g.collectStructureRefs(id, refs)
	}
	for _, id := range view.ActivePromises {
		add(id)
		g.collectStructureRefs(id, refs)
	}
	for _, id := range view.ActiveArcs {
		add(id)
		g.collectStructureRefs(id, refs)
	}
	for _, id := range view.ServedThreads {
		add(id)
		g.collectStructureRefs(id, refs)
	}
	for _, id := range view.ServedPromises {
		add(id)
		g.collectStructureRefs(id, refs)
	}
	for _, id := range view.ServedArcs {
		add(id)
		g.collectStructureRefs(id, refs)
	}
	for _, id := range view.Reveals {
		add(id)
	}
	for _, id := range view.Mentions {
		add(id)
	}
	g.collectFields(g.envForDecl(chapter.Decl), chapter.Fields, refs)
	for _, beat := range g.orderedBeats(codeForChapter(chapter)) {
		env := g.envForDecl(beat.Decl)
		g.collectFields(env, beat.Fields, refs)
		for _, stmt := range beat.Effects {
			g.collectExprRefs(env, stmt.Target, refs)
			g.collectExprRefs(env, stmt.Value, refs)
		}
	}

	context := ContextInfo{
		RelevantCharacters: []model.SymbolID{},
		RelevantPlaces:     []model.SymbolID{},
		RelevantObjects:    []model.SymbolID{},
		RelevantFacts:      []model.SymbolID{},
	}
	for id := range refs {
		if g.Project.Entities.Characters[id] != nil {
			context.RelevantCharacters = append(context.RelevantCharacters, id)
		}
		if g.Project.Entities.Places[id] != nil {
			context.RelevantPlaces = append(context.RelevantPlaces, id)
		}
		if g.Project.Entities.Objects[id] != nil {
			context.RelevantObjects = append(context.RelevantObjects, id)
		}
		if g.Project.Entities.Facts[id] != nil {
			context.RelevantFacts = append(context.RelevantFacts, id)
		}
	}
	sortIDs(context.RelevantCharacters)
	sortIDs(context.RelevantPlaces)
	sortIDs(context.RelevantObjects)
	sortIDs(context.RelevantFacts)
	return context
}

func (g *Generator) stateInfo(begin, end state.Store) StateInfo {
	return StateInfo{
		StateAtChapterBegin: storeMap(begin),
		ExpectedChanges:     stateChanges(begin, end),
		StateAtChapterEnd:   storeMap(end),
	}
}

func (g *Generator) structureInfo(view *structure.ChapterView) StructureInfo {
	return StructureInfo{
		StartPatterns:  cloneIDs(view.StartPatterns),
		ActiveThreads:  cloneIDs(view.ActiveThreads),
		ActivePromises: cloneIDs(view.ActivePromises),
		ActiveArcs:     cloneIDs(view.ActiveArcs),
		ServedThreads:  cloneIDs(view.ServedThreads),
		ServedPromises: cloneIDs(view.ServedPromises),
		ServedArcs:     cloneIDs(view.ServedArcs),
	}
}

func (g *Generator) beatsInfo(code string) BeatsInfo {
	info := BeatsInfo{}
	for _, beat := range g.orderedBeats(code) {
		info.OrderedBeats = append(info.OrderedBeats, beat.ID)
		var effects []EffectInfo
		for _, field := range beat.Fields {
			switch field.Name {
			case "precondition":
				for _, stmt := range field.Statements {
					if stmt.Expr != nil {
						info.BeatPreconditions = append(info.BeatPreconditions, BeatConditions{Beat: beat.ID, Condition: exprString(stmt.Expr)})
					}
				}
			case "render_hint":
				if text := exprString(field.Value); text != "" {
					info.BeatRenderHints = append(info.BeatRenderHints, BeatRenderHint{Beat: string(beat.ID), Hint: text})
				}
			case "effect":
				for _, stmt := range field.Statements {
					effects = append(effects, EffectInfo{
						Target: exprString(stmt.Target),
						Op:     stmt.Op,
						Value:  exprString(stmt.Value),
					})
				}
			}
		}
		if len(effects) > 0 {
			info.BeatEffects = append(info.BeatEffects, BeatEffects{Beat: beat.ID, Effects: effects})
		}
	}
	return info
}

func (g *Generator) proseInfo(chapter *model.Chapter) ProseInfo {
	return ProseInfo{
		ProseHint:    textField(chapter.Fields, "prose_hint"),
		TargetLength: exprField(chapter.Fields, "target_length"),
	}
}

func (g *Generator) neighborChapters(code string) (string, string) {
	for index, candidate := range g.Timeline.OrderedCodes {
		if candidate != code {
			continue
		}
		previous := ""
		next := ""
		if index > 0 {
			previous = g.Timeline.OrderedCodes[index-1]
		}
		if index+1 < len(g.Timeline.OrderedCodes) {
			next = g.Timeline.OrderedCodes[index+1]
		}
		return previous, next
	}
	return "", ""
}

func (g *Generator) orderedBeats(code string) []*model.Beat {
	var beats []*model.Beat
	for _, step := range g.Timeline.OrderedBeats {
		if step.Chapter == code {
			beats = append(beats, step.Beat)
		}
	}
	return beats
}

func (g *Generator) collectStructureRefs(id model.SymbolID, refs map[model.SymbolID]bool) {
	switch {
	case g.Project.Threads[id] != nil:
		thread := g.Project.Threads[id]
		g.collectFields(g.envForDecl(thread.Decl), thread.Fields, refs)
	case g.Project.Promises[id] != nil:
		promise := g.Project.Promises[id]
		g.collectFields(g.envForDecl(promise.Decl), promise.Fields, refs)
	case g.Project.Arcs[id] != nil:
		arc := g.Project.Arcs[id]
		g.collectFields(g.envForDecl(arc.Decl), arc.Fields, refs)
	}
}

func (g *Generator) collectFields(env resolve.FileEnv, fields []model.FieldValue, refs map[model.SymbolID]bool) {
	for _, field := range fields {
		g.collectExprRefs(env, field.Value, refs)
		for _, stmt := range field.Statements {
			g.collectExprRefs(env, stmt.Target, refs)
			g.collectExprRefs(env, stmt.Value, refs)
			g.collectExprRefs(env, stmt.Expr, refs)
		}
	}
}

func (g *Generator) collectExprRefs(env resolve.FileEnv, expr *ast.Expr, refs map[model.SymbolID]bool) {
	if expr == nil {
		return
	}
	if expr.Kind == ast.ExprRef || expr.Kind == ast.ExprPath {
		symbol, _ := g.Resolved.ResolveName(env, expr.Value, expr.Span, false)
		if symbol != nil {
			refs[model.SymbolIDFor(symbol.Namespace, symbol.Name)] = true
		}
	}
	for _, child := range expr.Children {
		g.collectExprRefs(env, child, refs)
	}
	for _, arg := range expr.Args {
		g.collectExprRefs(env, arg, refs)
	}
}

func (g *Generator) envForDecl(decl *ast.Decl) resolve.FileEnv {
	return g.Timeline.EnvForDecl(decl)
}

func storeMap(store state.Store) map[string]state.Value {
	out := map[string]state.Value{}
	for _, key := range store.Keys() {
		out[key.String()] = store.Get(key)
	}
	return out
}

func stateChanges(begin, end state.Store) []StateChange {
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
	sort.Slice(ordered, func(i, j int) bool { return ordered[i].String() < ordered[j].String() })
	var changes []StateChange
	for _, key := range ordered {
		before := begin.Get(key)
		after := end.Get(key)
		if before.StableKey() == after.StableKey() {
			continue
		}
		changes = append(changes, StateChange{Field: key.String(), Before: before, After: after})
	}
	return changes
}

func textField(fields []model.FieldValue, name string) string {
	field, ok := fieldValue(fields, name)
	if !ok {
		return ""
	}
	return exprString(field.Value)
}

func symbolField(fields []model.FieldValue, name string) string {
	return textField(fields, name)
}

func exprField(fields []model.FieldValue, name string) string {
	return textField(fields, name)
}

func fieldValue(fields []model.FieldValue, name string) (model.FieldValue, bool) {
	for _, field := range fields {
		if field.Name == name {
			return field, true
		}
	}
	return model.FieldValue{}, false
}

func exprString(expr *ast.Expr) string {
	if expr == nil {
		return ""
	}
	switch expr.Kind {
	case ast.ExprString, ast.ExprMultiline, ast.ExprInteger, ast.ExprBool, ast.ExprRef, ast.ExprPath, ast.ExprSymbol, ast.ExprLanguage:
		return expr.Value
	case ast.ExprLength:
		if len(expr.Children) > 0 {
			return exprString(expr.Children[0]) + " " + expr.Value
		}
		return expr.Value
	case ast.ExprList:
		parts := make([]string, 0, len(expr.Children))
		for _, child := range expr.Children {
			parts = append(parts, exprString(child))
		}
		return "[" + strings.Join(parts, ", ") + "]"
	case ast.ExprSet:
		parts := make([]string, 0, len(expr.Children))
		for _, child := range expr.Children {
			parts = append(parts, exprString(child))
		}
		return "{" + strings.Join(parts, ", ") + "}"
	case ast.ExprBinary:
		parts := make([]string, 0, len(expr.Children))
		for _, child := range expr.Children {
			parts = append(parts, exprString(child))
		}
		return strings.Join(parts, " "+expr.Op+" ")
	case ast.ExprUnary:
		if len(expr.Children) > 0 {
			return expr.Op + " " + exprString(expr.Children[0])
		}
	case ast.ExprParen:
		if len(expr.Children) > 0 {
			return "(" + exprString(expr.Children[0]) + ")"
		}
	}
	if expr.Value != "" {
		return expr.Value
	}
	return string(expr.Kind)
}

func cloneIDs(ids []model.SymbolID) []model.SymbolID {
	return append([]model.SymbolID(nil), ids...)
}

func sortIDs(ids []model.SymbolID) {
	sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })
}

func codeForChapter(chapter *model.Chapter) string {
	if chapter == nil {
		return ""
	}
	return chapter.Code.Canonical()
}

func lengthTargetNumber(text string) int {
	fields := strings.Fields(text)
	if len(fields) == 0 {
		return 0
	}
	value, _ := strconv.Atoi(fields[0])
	return value
}

func writeLLMInstructions(builder *strings.Builder) {
	builder.WriteString(llmBuildNarrDoc)
}

func writeLLMChapter(builder *strings.Builder, info ChapterInfo) {
	builder.WriteString("  chapter:\n")
	writeRawField(builder, "    ", "code", info.Code)
	writeRawField(builder, "    ", "canonical_code", info.CanonicalCode)
	writeRawField(builder, "    ", "alias", info.Alias)
	writeStringField(builder, "    ", "title", info.Title)
	writeRawField(builder, "    ", "purpose", info.Purpose)
	writeRawField(builder, "    ", "target_length", info.TargetLength)
	writeRawField(builder, "    ", "volume_code", info.VolumeCode)
	writeRawField(builder, "    ", "previous_chapter", info.PreviousChapter)
	writeRawField(builder, "    ", "next_chapter", info.NextChapter)
}

func writeLLMSummary(builder *strings.Builder, info SummaryInfo) {
	builder.WriteString("  summary:\n")
	writeStringField(builder, "    ", "novel_summary", info.NovelSummary)
	writeStringField(builder, "    ", "volume_summary", info.VolumeSummary)
	writeStringField(builder, "    ", "chapter_summary", info.ChapterSummary)
}

func writeLLMContext(builder *strings.Builder, info ContextInfo) {
	builder.WriteString("  context:\n")
	writeIDList(builder, "    ", "relevant_characters", info.RelevantCharacters)
	writeIDList(builder, "    ", "relevant_places", info.RelevantPlaces)
	writeIDList(builder, "    ", "relevant_objects", info.RelevantObjects)
	writeIDList(builder, "    ", "relevant_facts", info.RelevantFacts)
}

func writeLLMState(builder *strings.Builder, info StateInfo) {
	builder.WriteString("  state:\n")
	writeStateMap(builder, "    ", "state_at_chapter_begin", info.StateAtChapterBegin)
	writeStateChanges(builder, "    ", "expected_state_changes", info.ExpectedChanges)
	writeStateMap(builder, "    ", "state_at_chapter_end", info.StateAtChapterEnd)
}

func writeLLMStructure(builder *strings.Builder, info StructureInfo) {
	builder.WriteString("  structure:\n")
	writeIDList(builder, "    ", "start_patterns", info.StartPatterns)
	writeIDList(builder, "    ", "active_threads", info.ActiveThreads)
	writeIDList(builder, "    ", "active_promises", info.ActivePromises)
	writeIDList(builder, "    ", "active_arcs", info.ActiveArcs)
	writeIDList(builder, "    ", "served_threads", info.ServedThreads)
	writeIDList(builder, "    ", "served_promises", info.ServedPromises)
	writeIDList(builder, "    ", "served_arcs", info.ServedArcs)
}

func writeLLMBeats(builder *strings.Builder, info BeatsInfo) {
	builder.WriteString("  beats:\n")
	writeIDList(builder, "    ", "ordered_beats", info.OrderedBeats)
	writeBeatConditions(builder, "    ", "beat_preconditions", info.BeatPreconditions)
	writeBeatEffects(builder, "    ", "beat_effects", info.BeatEffects)
	writeBeatRenderHints(builder, "    ", "beat_render_hints", info.BeatRenderHints)
}

func writeLLMProse(builder *strings.Builder, info ProseInfo) {
	builder.WriteString("  prose:\n")
	writeStringField(builder, "    ", "prose_hint", info.ProseHint)
	writeRawField(builder, "    ", "target_length", info.TargetLength)
}

func writeStringField(builder *strings.Builder, indent, name, value string) {
	if value == "" {
		return
	}
	fmt.Fprintf(builder, "%s%s: %s\n", indent, name, strconv.Quote(value))
}

func writeRawField(builder *strings.Builder, indent, name, value string) {
	if value == "" {
		return
	}
	fmt.Fprintf(builder, "%s%s: %s\n", indent, name, value)
}

func writeIDList(builder *strings.Builder, indent, name string, ids []model.SymbolID) {
	fmt.Fprintf(builder, "%s%s: [\n", indent, name)
	for _, id := range ids {
		fmt.Fprintf(builder, "%s  %s,\n", indent, id)
	}
	fmt.Fprintf(builder, "%s]\n", indent)
}

func writeStateMap(builder *strings.Builder, indent, name string, values map[string]state.Value) {
	fmt.Fprintf(builder, "%s%s: {\n", indent, name)
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		fmt.Fprintf(builder, "%s  %s: %s\n", indent, key, values[key].String())
	}
	fmt.Fprintf(builder, "%s}\n", indent)
}

func writeStateChanges(builder *strings.Builder, indent, name string, changes []StateChange) {
	fmt.Fprintf(builder, "%s%s: [\n", indent, name)
	for _, change := range changes {
		fmt.Fprintf(builder, "%s  { field: %s, before: %s, after: %s },\n", indent, change.Field, change.Before.String(), change.After.String())
	}
	fmt.Fprintf(builder, "%s]\n", indent)
}

func writeBeatConditions(builder *strings.Builder, indent, name string, conditions []BeatConditions) {
	fmt.Fprintf(builder, "%s%s: [\n", indent, name)
	for _, condition := range conditions {
		fmt.Fprintf(builder, "%s  { beat: %s, condition: %s },\n", indent, condition.Beat, strconv.Quote(condition.Condition))
	}
	fmt.Fprintf(builder, "%s]\n", indent)
}

func writeBeatEffects(builder *strings.Builder, indent, name string, groups []BeatEffects) {
	fmt.Fprintf(builder, "%s%s: [\n", indent, name)
	for _, group := range groups {
		fmt.Fprintf(builder, "%s  {\n", indent)
		fmt.Fprintf(builder, "%s    beat: %s\n", indent, group.Beat)
		fmt.Fprintf(builder, "%s    effect:\n", indent)
		for _, effect := range group.Effects {
			fmt.Fprintf(builder, "%s      %s %s %s\n", indent, effect.Target, effect.Op, effect.Value)
		}
		fmt.Fprintf(builder, "%s  },\n", indent)
	}
	fmt.Fprintf(builder, "%s]\n", indent)
}

func writeBeatRenderHints(builder *strings.Builder, indent, name string, hints []BeatRenderHint) {
	fmt.Fprintf(builder, "%s%s: [\n", indent, name)
	for _, hint := range hints {
		fmt.Fprintf(builder, "%s  { beat: %s, hint: %s },\n", indent, hint.Beat, strconv.Quote(hint.Hint))
	}
	fmt.Fprintf(builder, "%s]\n", indent)
}
