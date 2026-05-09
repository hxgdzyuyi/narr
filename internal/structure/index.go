package structure

import (
	"fmt"
	"sort"
	"strings"

	"narr/internal/ast"
	"narr/internal/model"
	"narr/internal/resolve"
	"narr/internal/source"
	"narr/internal/state"
)

type ChapterView struct {
	Code           string           `json:"code"`
	StartPatterns  []model.SymbolID `json:"start_patterns"`
	ActiveThreads  []model.SymbolID `json:"active_threads"`
	ActivePromises []model.SymbolID `json:"active_promises"`
	ActiveArcs     []model.SymbolID `json:"active_arcs"`
	ServedThreads  []model.SymbolID `json:"served_threads"`
	ServedPromises []model.SymbolID `json:"served_promises"`
	ServedArcs     []model.SymbolID `json:"served_arcs"`
	Reveals        []model.SymbolID `json:"reveals"`
	Mentions       []model.SymbolID `json:"mentions"`
}

type Index struct {
	Project  *model.Project
	Resolved *resolve.Project
	Timeline *state.Timeline

	Chapters map[string]*ChapterView

	beatChapters map[model.SymbolID]string
	beatOrder    map[model.SymbolID]int
	positions    map[string]int

	startPatterns map[model.SymbolID]*startPatternInfo
	promises      map[model.SymbolID]*promiseInfo
	threads       map[model.SymbolID]*threadInfo
	arcs          map[model.SymbolID]*arcInfo

	explicitPromisePayoffs map[model.SymbolID]state.Anchor
	explicitThreadResolves map[model.SymbolID]state.Anchor

	diagnostics []source.Diagnostic
}

type startPatternInfo struct {
	ID          model.SymbolID
	Anchor      state.Anchor
	HasAnchor   bool
	ThreadIDs   []model.SymbolID
	PromiseIDs  []model.SymbolID
	ArcIDs      []model.SymbolID
	AnchorField model.FieldValue
}

type promiseInfo struct {
	ID              model.SymbolID
	Setup           state.Anchor
	HasSetup        bool
	PayoffBy        state.Anchor
	HasPayoffBy     bool
	PayoffAt        state.Anchor
	HasPayoffAt     bool
	StartPattern    model.SymbolID
	HasStartPattern bool
}

type threadInfo struct {
	ID              model.SymbolID
	StartsAt        state.Anchor
	HasStartsAt     bool
	ResolvedAt      state.Anchor
	HasResolvedAt   bool
	StartPattern    model.SymbolID
	HasStartPattern bool
}

type arcInfo struct {
	ID              model.SymbolID
	Subject         model.SymbolID
	HasSubject      bool
	StartsAt        state.Anchor
	HasStartsAt     bool
	StartPattern    model.SymbolID
	HasStartPattern bool
	StateField      string
	HasStateField   bool
	FieldKey        state.FieldKey
	FieldType       model.TypeRef
	States          map[string]bool
	HasStates       bool
}

func Build(project *model.Project, resolved *resolve.Project, timeline *state.Timeline) (*Index, []source.Diagnostic) {
	index := &Index{
		Project:                project,
		Resolved:               resolved,
		Timeline:               timeline,
		Chapters:               map[string]*ChapterView{},
		beatChapters:           map[model.SymbolID]string{},
		beatOrder:              map[model.SymbolID]int{},
		positions:              map[string]int{},
		startPatterns:          map[model.SymbolID]*startPatternInfo{},
		promises:               map[model.SymbolID]*promiseInfo{},
		threads:                map[model.SymbolID]*threadInfo{},
		arcs:                   map[model.SymbolID]*arcInfo{},
		explicitPromisePayoffs: map[model.SymbolID]state.Anchor{},
		explicitThreadResolves: map[model.SymbolID]state.Anchor{},
	}
	index.indexTimeline()
	index.checkChaptersAndBeats()
	index.checkStartPatterns()
	index.checkPromises()
	index.checkThreads()
	index.checkArcs()
	index.deriveViews()
	index.checkInvariants()
	return index, index.diagnostics
}

func (i *Index) View(code string) *ChapterView {
	return i.Chapters[code]
}

func (i *Index) indexTimeline() {
	order := 0
	pos := 0
	for _, code := range i.Timeline.OrderedCodes {
		i.Chapters[code] = &ChapterView{Code: code}
		i.positions[anchorKey(state.Anchor{Kind: state.AnchorChapterBegin, Chapter: code})] = pos
		pos++
		for _, step := range i.Timeline.OrderedBeats {
			if step.Chapter != code {
				continue
			}
			i.beatChapters[step.Beat.ID] = code
			i.beatOrder[step.Beat.ID] = order
			i.positions[anchorKey(state.Anchor{Kind: state.AnchorBeatBefore, Beat: step.Beat.ID})] = pos
			pos++
			i.positions[anchorKey(state.Anchor{Kind: state.AnchorBeatAfter, Beat: step.Beat.ID})] = pos
			pos++
			order++
		}
		i.positions[anchorKey(state.Anchor{Kind: state.AnchorChapterEnd, Chapter: code})] = pos
		pos++
	}
}

func (i *Index) checkChaptersAndBeats() {
	listed := map[model.SymbolID]string{}
	for code, chapter := range i.Project.Chapters {
		seen := map[model.SymbolID]source.Span{}
		env := i.envForDecl(chapter.Decl)
		for _, field := range chapter.Fields {
			if field.Name != "beats" || field.Value == nil {
				continue
			}
			for _, child := range field.Value.Children {
				symbol := i.resolveSymbol(env, child, resolve.SymbolBeat, "chapter.beats")
				if symbol == nil {
					continue
				}
				id := model.SymbolIDFor(symbol.Namespace, symbol.Name)
				if previous, ok := seen[id]; ok {
					i.addError("E0601", child.Span, fmt.Sprintf("beat %s appears more than once in chapter %s", child.Value, code))
					_ = previous
					continue
				}
				seen[id] = child.Span
				if previousChapter, ok := listed[id]; ok && previousChapter != code {
					i.addError("E0627", child.Span, fmt.Sprintf("beat %s appears in both %s and %s", child.Value, previousChapter, code))
					continue
				}
				listed[id] = code
			}
		}
	}

	for _, beat := range i.Project.Beats {
		env := i.envForDecl(beat.Decl)
		if beat.Anchor == nil {
			i.addError("E0602", beat.Decl.Span, fmt.Sprintf("beat %s must have a chapter anchor", beat.Name))
			continue
		}
		anchor, diagnostics := i.Timeline.AnchorFromExpr(env, beat.Anchor)
		i.diagnostics = append(i.diagnostics, diagnostics...)
		if len(diagnostics) > 0 {
			continue
		}
		if anchor.Kind != state.AnchorChapterBegin && anchor.Kind != state.AnchorChapterEnd {
			i.addError("E0603", beat.Anchor.Span, fmt.Sprintf("beat %s anchor must point to a chapter", beat.Name))
			continue
		}
		listedChapter, ok := listed[beat.ID]
		if !ok {
			i.addError("E0604", beat.Decl.Span, fmt.Sprintf("beat %s must appear in chapter %s beats list", beat.Name, anchor.Chapter))
			continue
		}
		if listedChapter != anchor.Chapter {
			i.addError("E0605", beat.Decl.Span, fmt.Sprintf("beat %s anchor points to %s but is listed in %s", beat.Name, anchor.Chapter, listedChapter))
		}
	}

	for _, beat := range i.Project.Beats {
		i.checkBeat(beat)
	}
}

func (i *Index) checkBeat(beat *model.Beat) {
	env := i.envForDecl(beat.Decl)
	if _, ok := i.Timeline.StoreAt(state.Anchor{Kind: state.AnchorBeatBefore, Beat: beat.ID}); ok {
		store, _ := i.Timeline.StoreAt(state.Anchor{Kind: state.AnchorBeatBefore, Beat: beat.ID})
		for _, field := range beat.Fields {
			if field.Name != "precondition" {
				continue
			}
			i.checkConditionBlock(env, store, field.Statements, "beat precondition")
		}
	}
	i.checkBeatNarrativeLinks(env, beat)
	i.checkBeatStateConflicts(env, beat)
}

func (i *Index) checkBeatNarrativeLinks(env resolve.FileEnv, beat *model.Beat) {
	for _, field := range beat.Fields {
		switch field.Name {
		case "sets_up", "pays_off":
			i.refsFromExpr(env, field.Value, field.Name, resolve.SymbolPromise)
		case "advances":
			i.refsFromExpr(env, field.Value, field.Name, resolve.SymbolThread, resolve.SymbolArc)
		case "resolves":
			i.refsFromExpr(env, field.Value, field.Name, resolve.SymbolThread)
		case "reveals":
			i.refsFromExpr(env, field.Value, field.Name, resolve.SymbolFact)
		case "mentions":
			i.refsFromExpr(env, field.Value, field.Name)
		}
	}
}

func (i *Index) checkBeatStateConflicts(env resolve.FileEnv, beat *model.Beat) {
	assignments := map[state.FieldKey]state.Value{}
	setOps := map[string]ast.Stmt{}
	for _, stmt := range beat.Effects {
		key, fieldType, ok := i.Timeline.LookupStateField(env, stmt.Target.Value, stmt.Target.Span)
		if !ok {
			continue
		}
		switch stmt.Kind {
		case ast.StmtAssignment, ast.StmtInit:
			value := i.Timeline.ValueFromExpr(env, stmt.Value, fieldType)
			if previous, ok := assignments[key]; ok && previous.StableKey() != value.StableKey() {
				i.addError("E0606", stmt.Span, fmt.Sprintf("beat %s assigns conflicting values to %s", beat.Name, key.String()))
				continue
			}
			assignments[key] = value
		case ast.StmtSetAdd, ast.StmtSetRemove:
			elemType := elementType(fieldType)
			value := i.Timeline.ValueFromExpr(env, stmt.Value, elemType)
			opposite := ast.StmtSetRemove
			if stmt.Kind == ast.StmtSetRemove {
				opposite = ast.StmtSetAdd
			}
			opKey := fmt.Sprintf("%s:%s:%s", key.String(), opposite, value.StableKey())
			if _, ok := setOps[opKey]; ok {
				i.addError("E0607", stmt.Span, fmt.Sprintf("beat %s both adds and removes %s on %s", beat.Name, value.String(), key.String()))
			}
			setOps[fmt.Sprintf("%s:%s:%s", key.String(), stmt.Kind, value.StableKey())] = stmt
		}
	}
}

func (i *Index) checkStartPatterns() {
	for id, pattern := range i.Project.StartPatterns {
		info := &startPatternInfo{ID: id}
		i.startPatterns[id] = info
		env := i.envForDecl(pattern.Decl)
		if field, ok := fieldByName(pattern.Fields, "at"); ok && field.Value != nil {
			anchor, diagnostics := i.Timeline.AnchorFromExpr(env, field.Value)
			i.diagnostics = append(i.diagnostics, diagnostics...)
			if len(diagnostics) == 0 {
				info.Anchor = anchor
				info.HasAnchor = true
				info.AnchorField = field
				if _, ok := i.Timeline.StoreAt(anchor); !ok {
					i.addError("E0608", field.Value.Span, fmt.Sprintf("start_pattern %s has no state checkpoint at %s", pattern.Name, field.Value.Value))
				}
			}
		} else {
			i.addError("E0609", pattern.Decl.Span, fmt.Sprintf("start_pattern %s is missing at", pattern.Name))
		}

		if info.HasAnchor {
			if store, ok := i.Timeline.StoreAt(info.Anchor); ok {
				for _, field := range pattern.Fields {
					if field.Name == "requires" {
						i.checkConditionBlock(env, store, field.Statements, "start_pattern requires")
					}
				}
			}
		}

		for _, field := range pattern.Fields {
			if field.Name != "starts" {
				continue
			}
			for _, stmt := range field.Statements {
				i.checkStartTarget(env, info, stmt)
			}
		}
	}
}

func (i *Index) checkStartTarget(env resolve.FileEnv, info *startPatternInfo, stmt ast.Stmt) {
	var want resolve.SymbolKind
	switch stmt.Name {
	case "thread":
		want = resolve.SymbolThread
	case "promise":
		want = resolve.SymbolPromise
	case "arc":
		want = resolve.SymbolArc
	default:
		i.addError("E0610", stmt.Span, fmt.Sprintf("start target kind %s is not valid", stmt.Name))
		return
	}
	symbol := i.resolveSymbol(env, stmt.Value, want, "start_pattern.starts")
	if symbol == nil {
		return
	}
	id := model.SymbolIDFor(symbol.Namespace, symbol.Name)
	switch want {
	case resolve.SymbolThread:
		info.ThreadIDs = append(info.ThreadIDs, id)
	case resolve.SymbolPromise:
		info.PromiseIDs = append(info.PromiseIDs, id)
	case resolve.SymbolArc:
		info.ArcIDs = append(info.ArcIDs, id)
	}
}

func (i *Index) checkPromises() {
	for id, promise := range i.Project.Promises {
		info := &promiseInfo{ID: id}
		i.promises[id] = info
		env := i.envForDecl(promise.Decl)
		if field, ok := fieldByName(promise.Fields, "setup_at"); ok && field.Value != nil {
			anchor, diagnostics := i.Timeline.AnchorFromExpr(env, field.Value)
			i.diagnostics = append(i.diagnostics, diagnostics...)
			if len(diagnostics) == 0 {
				info.Setup = anchor
				info.HasSetup = true
			}
		} else {
			i.addError("E0611", promise.Decl.Span, fmt.Sprintf("promise %s is missing setup_at", promise.Name))
		}
		if field, ok := fieldByName(promise.Fields, "payoff_by"); ok && field.Value != nil {
			anchor, diagnostics := i.Timeline.AnchorFromExpr(env, field.Value)
			i.diagnostics = append(i.diagnostics, diagnostics...)
			if len(diagnostics) == 0 {
				info.PayoffBy = anchor
				info.HasPayoffBy = true
			}
		}
		if field, ok := fieldByName(promise.Fields, "payoff_at"); ok && field.Value != nil {
			anchor, diagnostics := i.Timeline.AnchorFromExpr(env, field.Value)
			i.diagnostics = append(i.diagnostics, diagnostics...)
			if len(diagnostics) == 0 {
				info.PayoffAt = anchor
				info.HasPayoffAt = true
			}
		}
		if field, ok := fieldByName(promise.Fields, "start_pattern"); ok && field.Value != nil {
			symbol := i.resolveSymbol(env, field.Value, resolve.SymbolStartPattern, "promise.start_pattern")
			if symbol != nil {
				info.StartPattern = model.SymbolIDFor(symbol.Namespace, symbol.Name)
				info.HasStartPattern = true
			}
		}
		if info.HasSetup && info.HasPayoffAt && i.anchorAfter(info.Setup, info.PayoffAt) {
			i.addError("E0612", promise.Decl.Span, fmt.Sprintf("promise %s payoff_at is before setup_at", promise.Name))
		}
		if info.HasSetup && info.HasPayoffBy && i.anchorAfter(info.Setup, info.PayoffBy) {
			i.addError("E0613", promise.Decl.Span, fmt.Sprintf("promise %s payoff_by is before setup_at", promise.Name))
		}
		if info.HasStartPattern && info.HasSetup {
			i.checkStartPatternAnchorMatch("promise", promise.Name, promise.Decl.Span, info.StartPattern, info.Setup)
		}
	}
}

func (i *Index) checkThreads() {
	for id, thread := range i.Project.Threads {
		info := &threadInfo{ID: id}
		i.threads[id] = info
		env := i.envForDecl(thread.Decl)
		if field, ok := fieldByName(thread.Fields, "starts_at"); ok && field.Value != nil {
			anchor, diagnostics := i.Timeline.AnchorFromExpr(env, field.Value)
			i.diagnostics = append(i.diagnostics, diagnostics...)
			if len(diagnostics) == 0 {
				info.StartsAt = anchor
				info.HasStartsAt = true
			}
		} else {
			i.addError("E0614", thread.Decl.Span, fmt.Sprintf("thread %s is missing starts_at", thread.Name))
		}
		if field, ok := fieldByName(thread.Fields, "resolved_at"); ok && field.Value != nil {
			anchor, diagnostics := i.Timeline.AnchorFromExpr(env, field.Value)
			i.diagnostics = append(i.diagnostics, diagnostics...)
			if len(diagnostics) == 0 {
				info.ResolvedAt = anchor
				info.HasResolvedAt = true
			}
		}
		if field, ok := fieldByName(thread.Fields, "expected_resolution"); ok && field.Value != nil {
			_, diagnostics := i.Timeline.AnchorFromExpr(env, field.Value)
			i.diagnostics = append(i.diagnostics, diagnostics...)
		}
		if field, ok := fieldByName(thread.Fields, "start_pattern"); ok && field.Value != nil {
			symbol := i.resolveSymbol(env, field.Value, resolve.SymbolStartPattern, "thread.start_pattern")
			if symbol != nil {
				info.StartPattern = model.SymbolIDFor(symbol.Namespace, symbol.Name)
				info.HasStartPattern = true
			}
		}
		if info.HasStartPattern && info.HasStartsAt {
			i.checkStartPatternAnchorMatch("thread", thread.Name, thread.Decl.Span, info.StartPattern, info.StartsAt)
		}
	}
}

func (i *Index) checkArcs() {
	for id, arc := range i.Project.Arcs {
		info := &arcInfo{ID: id, States: map[string]bool{}}
		i.arcs[id] = info
		env := i.envForDecl(arc.Decl)
		if field, ok := fieldByName(arc.Fields, "subject"); ok && field.Value != nil {
			symbol := i.resolveSymbol(env, field.Value, resolve.SymbolCharacter, "arc.subject")
			if symbol != nil {
				info.Subject = model.SymbolIDFor(symbol.Namespace, symbol.Name)
				info.HasSubject = true
			}
		} else {
			i.addError("E0615", arc.Decl.Span, fmt.Sprintf("arc %s is missing subject", arc.Name))
		}
		if field, ok := fieldByName(arc.Fields, "starts_at"); ok && field.Value != nil {
			anchor, diagnostics := i.Timeline.AnchorFromExpr(env, field.Value)
			i.diagnostics = append(i.diagnostics, diagnostics...)
			if len(diagnostics) == 0 {
				info.StartsAt = anchor
				info.HasStartsAt = true
			}
		} else {
			i.addError("E0616", arc.Decl.Span, fmt.Sprintf("arc %s is missing starts_at", arc.Name))
		}
		if field, ok := fieldByName(arc.Fields, "start_pattern"); ok && field.Value != nil {
			symbol := i.resolveSymbol(env, field.Value, resolve.SymbolStartPattern, "arc.start_pattern")
			if symbol != nil {
				info.StartPattern = model.SymbolIDFor(symbol.Namespace, symbol.Name)
				info.HasStartPattern = true
			}
		}
		if field, ok := fieldByName(arc.Fields, "state_field"); ok && field.Value != nil {
			info.StateField = field.Value.Value
			info.HasStateField = true
		} else {
			i.addError("E0617", arc.Decl.Span, fmt.Sprintf("arc %s is missing state_field", arc.Name))
		}
		if field, ok := fieldByName(arc.Fields, "states"); ok && field.Value != nil {
			info.HasStates = true
			for _, child := range field.Value.Children {
				value := i.Timeline.ValueFromExpr(env, child, model.TypeRef{Kind: model.TypeSymbol})
				info.States[value.StableKey()] = true
			}
		}
		if info.HasSubject && info.HasStateField {
			key, fieldType, ok := i.Timeline.LookupStateField(env, string(info.Subject)+"."+info.StateField, arc.Decl.Span)
			if !ok {
				i.addError("E0618", arc.Decl.Span, fmt.Sprintf("arc %s state_field %s does not exist on subject", arc.Name, info.StateField))
			} else {
				info.FieldKey = key
				info.FieldType = fieldType
			}
		}
		if field, ok := fieldByName(arc.Fields, "initial"); ok && field.Value != nil && info.HasStates {
			value := i.Timeline.ValueFromExpr(env, field.Value, model.TypeRef{Kind: model.TypeSymbol})
			if !info.States[value.StableKey()] {
				i.addError("E0619", field.Value.Span, fmt.Sprintf("arc %s initial state %s is not listed in states", arc.Name, value.String()))
			}
		}
		if field, ok := fieldByName(arc.Fields, "expected_resolution"); ok && field.Value != nil {
			_, diagnostics := i.Timeline.AnchorFromExpr(env, field.Value)
			i.diagnostics = append(i.diagnostics, diagnostics...)
		}
		if info.HasStartPattern && info.HasStartsAt {
			i.checkStartPatternAnchorMatch("arc", arc.Name, arc.Decl.Span, info.StartPattern, info.StartsAt)
		}
	}
}

func (i *Index) deriveViews() {
	for _, info := range i.startPatterns {
		code, ok := i.chapterForAnchor(info.Anchor)
		if !ok {
			continue
		}
		view := i.ensureView(code)
		view.StartPatterns = append(view.StartPatterns, info.ID)
		view.ServedThreads = append(view.ServedThreads, info.ThreadIDs...)
		view.ServedPromises = append(view.ServedPromises, info.PromiseIDs...)
		view.ServedArcs = append(view.ServedArcs, info.ArcIDs...)
	}

	for _, promise := range i.promises {
		if promise.HasSetup {
			if code, ok := i.chapterForAnchor(promise.Setup); ok {
				i.ensureView(code).ServedPromises = append(i.ensureView(code).ServedPromises, promise.ID)
			}
		}
		if promise.HasPayoffAt {
			if code, ok := i.chapterForAnchor(promise.PayoffAt); ok {
				i.ensureView(code).ServedPromises = append(i.ensureView(code).ServedPromises, promise.ID)
			}
		}
	}
	for _, thread := range i.threads {
		if thread.HasResolvedAt {
			if code, ok := i.chapterForAnchor(thread.ResolvedAt); ok {
				i.ensureView(code).ServedThreads = append(i.ensureView(code).ServedThreads, thread.ID)
			}
		}
	}

	for _, step := range i.Timeline.OrderedBeats {
		i.deriveBeatLinks(step)
		i.deriveArcEffects(step)
	}

	for _, code := range i.Timeline.OrderedCodes {
		view := i.ensureView(code)
		chapterEnd := state.Anchor{Kind: state.AnchorChapterEnd, Chapter: code}
		chapterBegin := state.Anchor{Kind: state.AnchorChapterBegin, Chapter: code}
		for _, thread := range i.threads {
			if thread.HasStartsAt && i.anchorAtOrBefore(thread.StartsAt, chapterEnd) && !i.closedBefore(threadCloseAnchor(thread, i.explicitThreadResolves), chapterBegin) {
				view.ActiveThreads = append(view.ActiveThreads, thread.ID)
			}
		}
		for _, promise := range i.promises {
			if promise.HasSetup && i.anchorAtOrBefore(promise.Setup, chapterEnd) && !i.closedBefore(promiseCloseAnchor(promise, i.explicitPromisePayoffs), chapterBegin) {
				view.ActivePromises = append(view.ActivePromises, promise.ID)
			}
		}
		for _, arc := range i.arcs {
			if arc.HasStartsAt && i.anchorAtOrBefore(arc.StartsAt, chapterEnd) {
				view.ActiveArcs = append(view.ActiveArcs, arc.ID)
			}
		}
		normalizeView(view)
	}
}

func (i *Index) deriveBeatLinks(step state.BeatStep) {
	env := i.envForDecl(step.Beat.Decl)
	view := i.ensureView(step.Chapter)
	anchor := state.Anchor{Kind: state.AnchorBeatAfter, Beat: step.Beat.ID}
	for _, field := range step.Beat.Fields {
		switch field.Name {
		case "sets_up":
			view.ServedPromises = append(view.ServedPromises, i.refsFromExpr(env, field.Value, field.Name, resolve.SymbolPromise)...)
		case "pays_off":
			ids := i.refsFromExpr(env, field.Value, field.Name, resolve.SymbolPromise)
			view.ServedPromises = append(view.ServedPromises, ids...)
			for _, id := range ids {
				if existing, ok := i.explicitPromisePayoffs[id]; !ok || i.anchorAfter(existing, anchor) {
					i.explicitPromisePayoffs[id] = anchor
				}
			}
		case "advances":
			for _, id := range i.refsFromExpr(env, field.Value, field.Name, resolve.SymbolThread, resolve.SymbolArc) {
				if i.Project.Threads[id] != nil {
					view.ServedThreads = append(view.ServedThreads, id)
				}
				if i.Project.Arcs[id] != nil {
					view.ServedArcs = append(view.ServedArcs, id)
				}
			}
		case "resolves":
			ids := i.refsFromExpr(env, field.Value, field.Name, resolve.SymbolThread)
			view.ServedThreads = append(view.ServedThreads, ids...)
			for _, id := range ids {
				if existing, ok := i.explicitThreadResolves[id]; !ok || i.anchorAfter(existing, anchor) {
					i.explicitThreadResolves[id] = anchor
				}
			}
		case "reveals":
			view.Reveals = append(view.Reveals, i.refsFromExpr(env, field.Value, field.Name, resolve.SymbolFact)...)
		case "mentions":
			view.Mentions = append(view.Mentions, i.refsFromExpr(env, field.Value, field.Name)...)
		}
	}
}

func (i *Index) deriveArcEffects(step state.BeatStep) {
	env := i.envForDecl(step.Beat.Decl)
	view := i.ensureView(step.Chapter)
	for _, stmt := range step.Beat.Effects {
		if stmt.Target == nil {
			continue
		}
		key, fieldType, ok := i.Timeline.LookupStateField(env, stmt.Target.Value, stmt.Target.Span)
		if !ok {
			continue
		}
		for _, arc := range i.arcs {
			if !arc.HasSubject || !arc.HasStateField || arc.FieldKey != key {
				continue
			}
			view.ServedArcs = append(view.ServedArcs, arc.ID)
			if arc.HasStates && (stmt.Kind == ast.StmtAssignment || stmt.Kind == ast.StmtInit) {
				value := i.Timeline.ValueFromExpr(env, stmt.Value, fieldType)
				if !arc.States[value.StableKey()] {
					i.addError("E0620", stmt.Value.Span, fmt.Sprintf("arc state change %s for %s is not listed in states", value.String(), arc.ID))
				}
			}
		}
	}
}

func (i *Index) checkInvariants() {
	for _, invariant := range i.Project.Invariants {
		env := i.envForDecl(invariant.Decl)
		i.checkHiddenInvariant(env, invariant)
		i.checkAlwaysInvariant(env, invariant)
	}
}

func (i *Index) checkHiddenInvariant(env resolve.FileEnv, invariant *model.Invariant) {
	field, ok := fieldByName(invariant.Fields, "hidden")
	if !ok || field.Value == nil || field.Value.Kind != ast.ExprBinary || field.Value.Op != "until" || len(field.Value.Children) != 2 {
		return
	}
	fact := i.resolveSymbol(env, field.Value.Children[0], resolve.SymbolFact, "invariant.hidden")
	if fact == nil {
		return
	}
	until, diagnostics := i.Timeline.AnchorFromExpr(env, field.Value.Children[1])
	i.diagnostics = append(i.diagnostics, diagnostics...)
	if len(diagnostics) > 0 {
		return
	}
	factID := model.SymbolIDFor(fact.Namespace, fact.Name)
	for _, step := range i.Timeline.OrderedBeats {
		beatEnv := i.envForDecl(step.Beat.Decl)
		for _, beatField := range step.Beat.Fields {
			if beatField.Name != "reveals" {
				continue
			}
			if !containsID(i.refsFromExpr(beatEnv, beatField.Value, "reveals", resolve.SymbolFact), factID) {
				continue
			}
			revealAt := state.Anchor{Kind: state.AnchorBeatAfter, Beat: step.Beat.ID}
			if i.anchorBefore(revealAt, until) {
				i.addError("E0621", step.Beat.Decl.Span, fmt.Sprintf("invariant %s hides %s until %s, but beat %s reveals it earlier", invariant.Name, factID, anchorKey(until), step.Beat.Name))
			}
		}
	}
}

func (i *Index) checkAlwaysInvariant(env resolve.FileEnv, invariant *model.Invariant) {
	var activeUntil state.Anchor
	hasActiveUntil := false
	if field, ok := fieldByName(invariant.Fields, "active_until"); ok && field.Value != nil {
		anchor, diagnostics := i.Timeline.AnchorFromExpr(env, field.Value)
		i.diagnostics = append(i.diagnostics, diagnostics...)
		if len(diagnostics) == 0 {
			activeUntil = anchor
			hasActiveUntil = true
		}
	}
	for _, field := range invariant.Fields {
		if field.Name != "always" {
			continue
		}
		for _, code := range i.Timeline.OrderedCodes {
			for _, anchor := range []state.Anchor{
				{Kind: state.AnchorChapterBegin, Chapter: code},
				{Kind: state.AnchorChapterEnd, Chapter: code},
			} {
				if hasActiveUntil && !i.anchorBefore(anchor, activeUntil) {
					continue
				}
				store, ok := i.Timeline.StoreAt(anchor)
				if !ok {
					continue
				}
				i.checkConditionBlock(env, store, field.Statements, fmt.Sprintf("invariant %s always", invariant.Name))
			}
		}
	}
}

func (i *Index) checkStartPatternAnchorMatch(kind, name string, span source.Span, patternID model.SymbolID, anchor state.Anchor) {
	pattern := i.startPatterns[patternID]
	if pattern == nil || !pattern.HasAnchor {
		return
	}
	if anchorKey(pattern.Anchor) != anchorKey(anchor) {
		i.addError("E0622", span, fmt.Sprintf("%s %s starts at %s, not start_pattern %s at %s", kind, name, anchorKey(anchor), patternID, anchorKey(pattern.Anchor)))
	}
}

func (i *Index) checkConditionBlock(env resolve.FileEnv, store state.Store, statements []ast.Stmt, label string) {
	for _, stmt := range statements {
		if stmt.Kind != ast.StmtCondition || stmt.Expr == nil {
			continue
		}
		value := i.evalExpr(env, store, stmt.Expr)
		if value.Kind != state.ValueBool || !value.Bool {
			i.addError("E0623", stmt.Span, fmt.Sprintf("%s is not satisfied: %s", label, exprText(stmt.Expr)))
		}
	}
}

func (i *Index) refsFromExpr(env resolve.FileEnv, expr *ast.Expr, label string, allowed ...resolve.SymbolKind) []model.SymbolID {
	if expr == nil {
		return nil
	}
	var out []model.SymbolID
	switch expr.Kind {
	case ast.ExprList, ast.ExprSet:
		for _, child := range expr.Children {
			out = append(out, i.refsFromExpr(env, child, label, allowed...)...)
		}
		return out
	case ast.ExprParen:
		for _, child := range expr.Children {
			out = append(out, i.refsFromExpr(env, child, label, allowed...)...)
		}
		return out
	}
	symbol := i.resolveAnySymbol(env, expr, label)
	if symbol == nil {
		return nil
	}
	if len(allowed) > 0 && !kindAllowed(symbol.Kind, allowed) {
		i.addError("E0624", expr.Span, fmt.Sprintf("%s expects %s, got %s", label, kindsText(allowed), symbol.Kind))
		return nil
	}
	return []model.SymbolID{model.SymbolIDFor(symbol.Namespace, symbol.Name)}
}

func (i *Index) resolveSymbol(env resolve.FileEnv, expr *ast.Expr, kind resolve.SymbolKind, label string) *resolve.Symbol {
	symbol := i.resolveAnySymbol(env, expr, label)
	if symbol == nil {
		return nil
	}
	if symbol.Kind != kind {
		i.addError("E0625", expr.Span, fmt.Sprintf("%s expects %s, got %s", label, kind, symbol.Kind))
		return nil
	}
	return symbol
}

func (i *Index) resolveAnySymbol(env resolve.FileEnv, expr *ast.Expr, label string) *resolve.Symbol {
	if expr == nil {
		return nil
	}
	if expr.Kind != ast.ExprRef && expr.Kind != ast.ExprPath {
		i.addError("E0626", expr.Span, fmt.Sprintf("%s expects a reference", label))
		return nil
	}
	symbol, diagnostics := i.Resolved.ResolveName(env, expr.Value, expr.Span, true)
	i.diagnostics = append(i.diagnostics, diagnostics...)
	if len(diagnostics) > 0 || symbol == nil {
		return nil
	}
	return symbol
}

func (i *Index) envForDecl(decl *ast.Decl) resolve.FileEnv {
	return i.Timeline.EnvForDecl(decl)
}

func (i *Index) ensureView(code string) *ChapterView {
	view := i.Chapters[code]
	if view == nil {
		view = &ChapterView{Code: code}
		i.Chapters[code] = view
	}
	return view
}

func (i *Index) chapterForAnchor(anchor state.Anchor) (string, bool) {
	switch anchor.Kind {
	case state.AnchorChapterBegin, state.AnchorChapterEnd:
		return anchor.Chapter, anchor.Chapter != ""
	case state.AnchorBeatBefore, state.AnchorBeatAfter:
		code, ok := i.beatChapters[anchor.Beat]
		return code, ok
	default:
		return "", false
	}
}

func (i *Index) anchorBefore(left, right state.Anchor) bool {
	leftPos, leftOK := i.positions[anchorKey(left)]
	rightPos, rightOK := i.positions[anchorKey(right)]
	return leftOK && rightOK && leftPos < rightPos
}

func (i *Index) anchorAfter(left, right state.Anchor) bool {
	leftPos, leftOK := i.positions[anchorKey(left)]
	rightPos, rightOK := i.positions[anchorKey(right)]
	return leftOK && rightOK && leftPos > rightPos
}

func (i *Index) anchorAtOrBefore(left, right state.Anchor) bool {
	leftPos, leftOK := i.positions[anchorKey(left)]
	rightPos, rightOK := i.positions[anchorKey(right)]
	return leftOK && rightOK && leftPos <= rightPos
}

func (i *Index) closedBefore(close *state.Anchor, chapterBegin state.Anchor) bool {
	if close == nil {
		return false
	}
	return i.anchorBefore(*close, chapterBegin)
}

func (i *Index) addError(code string, span source.Span, message string) {
	i.diagnostics = append(i.diagnostics, source.Error(code, span.Start.File, span.Start.Line, span.Start.Column, message))
}

func anchorKey(anchor state.Anchor) string {
	switch anchor.Kind {
	case state.AnchorBeginning, state.AnchorEndOfStory:
		return string(anchor.Kind)
	case state.AnchorChapterBegin, state.AnchorChapterEnd:
		return string(anchor.Kind) + ":" + anchor.Chapter
	case state.AnchorBeatBefore, state.AnchorBeatAfter:
		return string(anchor.Kind) + ":" + string(anchor.Beat)
	default:
		return string(anchor.Kind)
	}
}

func fieldByName(fields []model.FieldValue, name string) (model.FieldValue, bool) {
	for _, field := range fields {
		if field.Name == name {
			return field, true
		}
	}
	return model.FieldValue{}, false
}

func normalizeView(view *ChapterView) {
	view.StartPatterns = uniqueSortedIDs(view.StartPatterns)
	view.ActiveThreads = uniqueSortedIDs(view.ActiveThreads)
	view.ActivePromises = uniqueSortedIDs(view.ActivePromises)
	view.ActiveArcs = uniqueSortedIDs(view.ActiveArcs)
	view.ServedThreads = uniqueSortedIDs(view.ServedThreads)
	view.ServedPromises = uniqueSortedIDs(view.ServedPromises)
	view.ServedArcs = uniqueSortedIDs(view.ServedArcs)
	view.Reveals = uniqueSortedIDs(view.Reveals)
	view.Mentions = uniqueSortedIDs(view.Mentions)
}

func uniqueSortedIDs(ids []model.SymbolID) []model.SymbolID {
	seen := map[model.SymbolID]bool{}
	out := ids[:0]
	for _, id := range ids {
		if seen[id] {
			continue
		}
		seen[id] = true
		out = append(out, id)
	}
	sort.Slice(out, func(a, b int) bool { return out[a] < out[b] })
	return out
}

func containsID(ids []model.SymbolID, target model.SymbolID) bool {
	for _, id := range ids {
		if id == target {
			return true
		}
	}
	return false
}

func kindAllowed(kind resolve.SymbolKind, allowed []resolve.SymbolKind) bool {
	for _, candidate := range allowed {
		if kind == candidate {
			return true
		}
	}
	return false
}

func kindsText(kinds []resolve.SymbolKind) string {
	parts := make([]string, 0, len(kinds))
	for _, kind := range kinds {
		parts = append(parts, string(kind))
	}
	return strings.Join(parts, " or ")
}

func promiseCloseAnchor(info *promiseInfo, explicit map[model.SymbolID]state.Anchor) *state.Anchor {
	if info.HasPayoffAt {
		return &info.PayoffAt
	}
	if anchor, ok := explicit[info.ID]; ok {
		return &anchor
	}
	return nil
}

func threadCloseAnchor(info *threadInfo, explicit map[model.SymbolID]state.Anchor) *state.Anchor {
	if info.HasResolvedAt {
		return &info.ResolvedAt
	}
	if anchor, ok := explicit[info.ID]; ok {
		return &anchor
	}
	return nil
}

func elementType(value model.TypeRef) model.TypeRef {
	if value.Elem != nil {
		return *value.Elem
	}
	return model.TypeRef{Kind: model.TypeSymbol}
}

func exprText(expr *ast.Expr) string {
	if expr == nil {
		return "<nil>"
	}
	if expr.Value != "" {
		return expr.Value
	}
	if expr.Op != "" {
		return expr.Op
	}
	return string(expr.Kind)
}
