package structure

import (
	"fmt"
	"strings"

	"narr/internal/ast"
	"narr/internal/model"
	"narr/internal/resolve"
	"narr/internal/source"
	"narr/internal/state"
)

func (e *queryEvaluator) fieldForRef(id model.SymbolID, name string) (model.FieldValue, *ast.Decl, bool) {
	if e.index.Project.Novel != nil && e.index.Project.Novel.ID == id {
		field, ok := fieldByName(e.index.Project.Novel.Fields, name)
		return field, e.index.Project.Novel.Decl, ok
	}
	if volume, ok := e.volumeForRef(id); ok {
		field, ok := fieldByName(volume.Fields, name)
		return field, volume.Decl, ok
	}
	if chapter, ok := e.chapterForRef(id); ok {
		field, ok := fieldByName(chapter.Fields, name)
		return field, chapter.Decl, ok
	}
	if beat := e.index.Project.Beats[id]; beat != nil {
		field, ok := fieldByName(beat.Fields, name)
		return field, beat.Decl, ok
	}
	if pattern := e.index.Project.StartPatterns[id]; pattern != nil {
		field, ok := fieldByName(pattern.Fields, name)
		return field, pattern.Decl, ok
	}
	if thread := e.index.Project.Threads[id]; thread != nil {
		field, ok := fieldByName(thread.Fields, name)
		return field, thread.Decl, ok
	}
	if promise := e.index.Project.Promises[id]; promise != nil {
		field, ok := fieldByName(promise.Fields, name)
		return field, promise.Decl, ok
	}
	if arc := e.index.Project.Arcs[id]; arc != nil {
		field, ok := fieldByName(arc.Fields, name)
		return field, arc.Decl, ok
	}
	if invariant := e.index.Project.Invariants[id]; invariant != nil {
		field, ok := fieldByName(invariant.Fields, name)
		return field, invariant.Decl, ok
	}
	if entity := e.index.Project.Entities.All[id]; entity != nil {
		for _, stmt := range entity.Initializers {
			if stmt.Target != nil && stmt.Target.Value == name {
				return model.FieldValue{Name: name, Value: stmt.Value}, entity.Decl, true
			}
		}
		if field, ok := entity.Fields[name]; ok && field.Default != nil {
			return model.FieldValue{Name: name, Value: field.Default}, entity.Decl, true
		}
	}
	return model.FieldValue{}, nil, false
}

func (e *queryEvaluator) symbolKind(id model.SymbolID) resolve.SymbolKind {
	if e.index.Project.Novel != nil && e.index.Project.Novel.ID == id {
		return resolve.SymbolNovel
	}
	if _, ok := e.volumeForRef(id); ok {
		return resolve.SymbolVolume
	}
	if _, ok := e.chapterForRef(id); ok {
		return resolve.SymbolChapter
	}
	if e.index.Project.Beats[id] != nil {
		return resolve.SymbolBeat
	}
	if e.index.Project.StartPatterns[id] != nil {
		return resolve.SymbolStartPattern
	}
	if e.index.Project.Threads[id] != nil {
		return resolve.SymbolThread
	}
	if e.index.Project.Promises[id] != nil {
		return resolve.SymbolPromise
	}
	if e.index.Project.Arcs[id] != nil {
		return resolve.SymbolArc
	}
	if e.index.Project.Invariants[id] != nil {
		return resolve.SymbolInvariant
	}
	if entity := e.index.Project.Entities.All[id]; entity != nil {
		switch entity.Kind {
		case model.EntityPlace:
			return resolve.SymbolPlace
		case model.EntityCharacter:
			return resolve.SymbolCharacter
		case model.EntityCollective:
			return resolve.SymbolCollective
		case model.EntityFaction:
			return resolve.SymbolFaction
		case model.EntityObject:
			return resolve.SymbolObject
		case model.EntityFact:
			return resolve.SymbolFact
		}
	}
	return ""
}

func (e *queryEvaluator) volumeForRef(id model.SymbolID) (*model.Volume, bool) {
	for _, volume := range e.index.Project.Volumes {
		if volume.ID == id {
			return volume, true
		}
	}
	return nil, false
}

func (e *queryEvaluator) chapterForRef(id model.SymbolID) (*model.Chapter, bool) {
	for _, chapter := range e.index.Project.Chapters {
		if chapter.ID == id {
			return chapter, true
		}
	}
	return nil, false
}

func (e *queryEvaluator) chapterCodeForRef(id model.SymbolID) (string, bool) {
	chapter, ok := e.chapterForRef(id)
	if !ok {
		return "", false
	}
	return chapter.Code.Canonical(), true
}

func (e *queryEvaluator) volumeCodeForRef(id model.SymbolID) (string, bool) {
	volume, ok := e.volumeForRef(id)
	if !ok {
		return "", false
	}
	return volume.Code.Canonical(), true
}

func (e *queryEvaluator) chapterRef(code string, span source.Span) EvalValue {
	chapter := e.index.Project.Chapters[code]
	if chapter == nil {
		e.addError("E0724", span, fmt.Sprintf("unknown chapter %s", code))
		return EvalMissingValue()
	}
	return EvalRefValue(chapter.ID)
}

func (e *queryEvaluator) anchorForRef(id model.SymbolID, suffix string) (state.Anchor, bool) {
	switch e.symbolKind(id) {
	case resolve.SymbolVolume:
		code, ok := e.volumeCodeForRef(id)
		if !ok {
			return state.Anchor{}, false
		}
		volume, err := model.ParseVolumeCode(code)
		if err != nil {
			return state.Anchor{}, false
		}
		return e.index.TimelineAnchorForVolume(volume, suffix)
	case resolve.SymbolChapter:
		code, ok := e.chapterCodeForRef(id)
		if !ok {
			return state.Anchor{}, false
		}
		if suffix == "" {
			suffix = "begin"
		}
		if suffix != "begin" && suffix != "end" {
			return state.Anchor{}, false
		}
		return e.index.Timeline.ChapterBoundary(code, suffix), true
	case resolve.SymbolBeat:
		if suffix == "" {
			suffix = "before"
		}
		if suffix != "before" && suffix != "after" {
			return state.Anchor{}, false
		}
		kind := state.AnchorBeatBefore
		if suffix == "after" {
			kind = state.AnchorBeatAfter
		}
		return state.Anchor{Kind: kind, Beat: id}, true
	default:
		return state.Anchor{}, false
	}
}

func (i *Index) TimelineAnchorForVolume(volume model.VolumeCode, suffix string) (state.Anchor, bool) {
	if suffix == "" {
		suffix = "begin"
	}
	first := ""
	last := ""
	for _, code := range i.Timeline.OrderedCodes {
		chapter := i.Project.Chapters[code]
		if chapter == nil || chapter.Code.VolumeCode().Compare(volume) != 0 {
			continue
		}
		if first == "" {
			first = code
		}
		last = code
	}
	if first == "" {
		return state.Anchor{}, false
	}
	if suffix == "end" {
		return state.Anchor{Kind: state.AnchorChapterEnd, Chapter: last}, true
	}
	if suffix == "begin" {
		return state.Anchor{Kind: state.AnchorChapterBegin, Chapter: first}, true
	}
	return state.Anchor{}, false
}

func (e *queryEvaluator) anchorFromValue(value EvalValue) (state.Anchor, bool) {
	switch value.Kind {
	case EvalAnchor:
		return value.Anchor, true
	case EvalRef:
		return e.anchorForRef(value.Ref, "")
	default:
		return state.Anchor{}, false
	}
}

func (e *queryEvaluator) chapterCodeFromValue(value EvalValue) (string, bool) {
	switch value.Kind {
	case EvalRef:
		if code, ok := e.chapterCodeForRef(value.Ref); ok {
			return code, true
		}
		if beat := e.index.Project.Beats[value.Ref]; beat != nil {
			code, ok := e.index.beatChapters[beat.ID]
			return code, ok
		}
	case EvalAnchor:
		return e.chapterCodeFromAnchor(value.Anchor)
	case EvalSymbol, EvalString:
		code, err := model.ParseChapterCode(strings.Trim(value.Text, `"`))
		if err == nil {
			return code.Canonical(), true
		}
	}
	return "", false
}

func (e *queryEvaluator) chapterCodeFromAnchor(anchor state.Anchor) (string, bool) {
	switch anchor.Kind {
	case state.AnchorChapterBegin, state.AnchorChapterEnd:
		return anchor.Chapter, anchor.Chapter != ""
	case state.AnchorBeatBefore, state.AnchorBeatAfter:
		code, ok := e.index.beatChapters[anchor.Beat]
		return code, ok
	default:
		return "", false
	}
}

func (e *queryEvaluator) volumeCodeFromValue(value EvalValue) (string, bool) {
	switch value.Kind {
	case EvalRef:
		if code, ok := e.volumeCodeForRef(value.Ref); ok {
			return code, true
		}
		if code, ok := e.chapterCodeForRef(value.Ref); ok {
			chapter := e.index.Project.Chapters[code]
			return chapter.Code.VolumeCode().Canonical(), true
		}
		if beat := e.index.Project.Beats[value.Ref]; beat != nil {
			code, ok := e.index.beatChapters[beat.ID]
			if !ok {
				return "", false
			}
			chapter := e.index.Project.Chapters[code]
			return chapter.Code.VolumeCode().Canonical(), true
		}
	case EvalAnchor:
		code, ok := e.chapterCodeFromAnchor(value.Anchor)
		if !ok {
			return "", false
		}
		chapter := e.index.Project.Chapters[code]
		return chapter.Code.VolumeCode().Canonical(), true
	case EvalSymbol, EvalString:
		volume, err := model.ParseVolumeCode(strings.Trim(value.Text, `"`))
		if err == nil {
			return volume.Canonical(), true
		}
		if chapter, err := model.ParseChapterCode(strings.Trim(value.Text, `"`)); err == nil {
			return chapter.VolumeCode().Canonical(), true
		}
	}
	return "", false
}

func (e *queryEvaluator) anchorPosition(anchor state.Anchor) int {
	switch anchor.Kind {
	case state.AnchorBeginning:
		return -1
	case state.AnchorEndOfStory:
		return len(e.index.positions) + 1
	default:
		position, ok := e.index.positions[anchorKey(anchor)]
		if !ok {
			return len(e.index.positions) + 1
		}
		return position
	}
}

func (e *queryEvaluator) chapterIndex(code string) int {
	for index, candidate := range e.index.Timeline.OrderedCodes {
		if candidate == code {
			return index
		}
	}
	return -1
}

func (e *queryEvaluator) domainMatches(domain string, value EvalValue) bool {
	if value.Kind != EvalRef {
		return false
	}
	switch domain {
	case "novel":
		return e.symbolKind(value.Ref) == resolve.SymbolNovel
	case "volume":
		return e.symbolKind(value.Ref) == resolve.SymbolVolume
	case "chapter":
		return e.symbolKind(value.Ref) == resolve.SymbolChapter
	case "beat":
		return e.symbolKind(value.Ref) == resolve.SymbolBeat
	case "thread":
		return e.symbolKind(value.Ref) == resolve.SymbolThread
	case "promise":
		return e.symbolKind(value.Ref) == resolve.SymbolPromise
	case "arc":
		return e.symbolKind(value.Ref) == resolve.SymbolArc
	case "character":
		return e.symbolKind(value.Ref) == resolve.SymbolCharacter
	case "place":
		return e.symbolKind(value.Ref) == resolve.SymbolPlace
	case "object":
		return e.symbolKind(value.Ref) == resolve.SymbolObject
	case "fact":
		return e.symbolKind(value.Ref) == resolve.SymbolFact
	default:
		return false
	}
}
