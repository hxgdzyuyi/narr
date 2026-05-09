package state

import (
	"fmt"
	"sort"
	"strconv"
	"strings"

	"narr/internal/ast"
	"narr/internal/model"
	"narr/internal/resolve"
	"narr/internal/source"
)

type AnchorKind string

const (
	AnchorBeginning    AnchorKind = "beginning"
	AnchorEndOfStory   AnchorKind = "end_of_story"
	AnchorChapterBegin AnchorKind = "chapter.begin"
	AnchorChapterEnd   AnchorKind = "chapter.end"
	AnchorBeatBefore   AnchorKind = "beat.before"
	AnchorBeatAfter    AnchorKind = "beat.after"
)

type Anchor struct {
	Kind    AnchorKind
	Chapter string
	Beat    model.SymbolID
}

type BeatStep struct {
	Beat    *model.Beat
	Chapter string
	Before  Store
	After   Store
}

type Timeline struct {
	Project       *model.Project
	Resolved      *resolve.Project
	Initial       Store
	Final         Store
	ChapterBegin  map[string]Store
	ChapterEnd    map[string]Store
	BeatBefore    map[model.SymbolID]Store
	BeatAfter     map[model.SymbolID]Store
	OrderedCodes  []string
	OrderedBeats  []BeatStep
	diagnostics   []source.Diagnostic
	entityIndex   map[model.SymbolID]*model.Entity
	classIndex    map[model.SymbolID]*model.Class
	fileEnvs      map[string]resolve.FileEnv
	chapterByDecl map[*ast.Decl]string
}

func Build(project *model.Project, resolved *resolve.Project) (*Timeline, []source.Diagnostic) {
	t := &Timeline{
		Project:       project,
		Resolved:      resolved,
		Initial:       NewStore(),
		ChapterBegin:  map[string]Store{},
		ChapterEnd:    map[string]Store{},
		BeatBefore:    map[model.SymbolID]Store{},
		BeatAfter:     map[model.SymbolID]Store{},
		entityIndex:   project.Entities.All,
		classIndex:    project.Entities.Classes,
		fileEnvs:      map[string]resolve.FileEnv{},
		chapterByDecl: map[*ast.Decl]string{},
	}
	for file, env := range resolved.FileEnvs {
		t.fileEnvs[file.Path] = env
	}
	for code, chapter := range project.Chapters {
		t.chapterByDecl[chapter.Decl] = code
	}

	t.buildInitialState()
	if source.HasErrors(t.diagnostics) {
		return t, t.diagnostics
	}
	t.applyChapters()
	return t, t.diagnostics
}

func (t *Timeline) State(field string, at Anchor) (Value, bool) {
	key, ok := t.fieldKeyFromRaw(t.defaultEnv(), field, source.Span{})
	if !ok {
		return Missing(), false
	}
	store, ok := t.storeAt(at)
	if !ok {
		return Missing(), false
	}
	return store.Get(key), true
}

func (t *Timeline) StoreAt(at Anchor) (Store, bool) {
	return t.storeAt(at)
}

func (t *Timeline) EnvForDecl(decl *ast.Decl) resolve.FileEnv {
	return t.envForDecl(decl)
}

func (t *Timeline) LookupStateField(env resolve.FileEnv, raw string, span source.Span) (FieldKey, model.TypeRef, bool) {
	expr := &ast.Expr{Kind: ast.ExprPath, Value: raw, Span: span}
	return t.lookupStateField(env, expr)
}

func (t *Timeline) ValueFromExpr(env resolve.FileEnv, expr *ast.Expr, expected model.TypeRef) Value {
	return t.valueFromExpr(env, expr, expected)
}

func (t *Timeline) storeAt(at Anchor) (Store, bool) {
	switch at.Kind {
	case AnchorBeginning:
		return t.Initial, true
	case AnchorEndOfStory:
		return t.Final, true
	case AnchorChapterBegin:
		store, ok := t.ChapterBegin[at.Chapter]
		return store, ok
	case AnchorChapterEnd:
		store, ok := t.ChapterEnd[at.Chapter]
		return store, ok
	case AnchorBeatBefore:
		store, ok := t.BeatBefore[at.Beat]
		return store, ok
	case AnchorBeatAfter:
		store, ok := t.BeatAfter[at.Beat]
		return store, ok
	default:
		return Store{}, false
	}
}

func (t *Timeline) ChapterBoundary(chapterCode, suffix string) Anchor {
	kind := AnchorChapterBegin
	if suffix == "end" {
		kind = AnchorChapterEnd
	}
	return Anchor{Kind: kind, Chapter: chapterCode}
}

func (t *Timeline) buildInitialState() {
	for _, entity := range t.Project.Entities.All {
		env := t.envForDecl(entity.Decl)
		for _, field := range entity.Fields {
			if field.Default == nil {
				continue
			}
			value := t.valueFromExpr(env, field.Default, field.Type)
			t.Initial.Set(FieldKey{Entity: entity.ID, Field: field.Name}, value)
		}
		if entity.Kind == model.EntityCharacter && entity.ClassName != "" {
			class := t.classForEntity(entity)
			if class != nil {
				for _, field := range class.Fields {
					if field.Default == nil {
						continue
					}
					if t.Initial.Get(FieldKey{Entity: entity.ID, Field: field.Name}).Kind != ValueMissing {
						continue
					}
					value := t.valueFromExpr(env, field.Default, field.Type)
					t.Initial.Set(FieldKey{Entity: entity.ID, Field: field.Name}, value)
				}
			}
		}
		for _, stmt := range entity.Initializers {
			fieldName := stmt.Target.Value
			fieldType := t.fieldType(entity, fieldName)
			key := FieldKey{Entity: entity.ID, Field: fieldName}
			t.applyStmt(env, t.Initial, key, fieldType, stmt)
		}
	}
}

func (t *Timeline) applyChapters() {
	current := t.Initial.Clone()
	t.OrderedCodes = t.sortedChapterCodes()
	for _, code := range t.OrderedCodes {
		t.ChapterBegin[code] = current.Clone()
		for _, beat := range t.orderedBeatsForChapter(code) {
			t.BeatBefore[beat.ID] = current.Clone()
			env := t.envForDecl(beat.Decl)
			for _, stmt := range beat.Effects {
				key, fieldType, ok := t.effectField(env, stmt.Target)
				if !ok {
					continue
				}
				t.applyStmt(env, current, key, fieldType, stmt)
			}
			t.BeatAfter[beat.ID] = current.Clone()
			t.OrderedBeats = append(t.OrderedBeats, BeatStep{Beat: beat, Chapter: code, Before: t.BeatBefore[beat.ID], After: t.BeatAfter[beat.ID]})
		}
		t.ChapterEnd[code] = current.Clone()
	}
	t.Final = current.Clone()
}

func (t *Timeline) applyStmt(env resolve.FileEnv, store Store, key FieldKey, fieldType model.TypeRef, stmt ast.Stmt) {
	switch stmt.Kind {
	case ast.StmtAssignment, ast.StmtInit:
		store.Set(key, t.valueFromExpr(env, stmt.Value, fieldType))
	case ast.StmtSetAdd:
		elemType := elementType(fieldType)
		store.AddToSet(key, t.valueFromExpr(env, stmt.Value, elemType))
	case ast.StmtSetRemove:
		elemType := elementType(fieldType)
		store.RemoveFromSet(key, t.valueFromExpr(env, stmt.Value, elemType))
	case ast.StmtListAppend:
		elemType := elementType(fieldType)
		store.AppendToList(key, t.valueFromExpr(env, stmt.Value, elemType))
	}
}

func (t *Timeline) valueFromExpr(env resolve.FileEnv, expr *ast.Expr, expected model.TypeRef) Value {
	if expr == nil {
		return Null()
	}
	switch expr.Kind {
	case ast.ExprString, ast.ExprMultiline:
		return String(expr.Value)
	case ast.ExprInteger:
		value, err := strconv.Atoi(expr.Value)
		if err != nil {
			t.addError("E0501", expr.Span, fmt.Sprintf("invalid integer %s", expr.Value))
			return Int(0)
		}
		return Int(value)
	case ast.ExprBool:
		return Bool(expr.Value == "true")
	case ast.ExprList:
		elemType := elementType(expected)
		values := make([]Value, 0, len(expr.Children))
		for _, child := range expr.Children {
			values = append(values, t.valueFromExpr(env, child, elemType))
		}
		return List(values)
	case ast.ExprSet:
		elemType := elementType(expected)
		values := make([]Value, 0, len(expr.Children))
		for _, child := range expr.Children {
			values = append(values, t.valueFromExpr(env, child, elemType))
		}
		return Set(values)
	case ast.ExprRef, ast.ExprPath:
		if expected.Kind == model.TypeSymbol || expected.Kind == model.TypeEnum || expected.Kind == model.TypeClaim {
			if symbol, _ := t.Resolved.ResolveName(env, expr.Value, expr.Span, false); symbol != nil {
				return Ref(model.SymbolIDFor(symbol.Namespace, symbol.Name))
			}
			return Symbol(expr.Value)
		}
		symbol, diagnostics := t.Resolved.ResolveName(env, expr.Value, expr.Span, true)
		if len(diagnostics) > 0 {
			t.diagnostics = append(t.diagnostics, diagnostics...)
			return Missing()
		}
		return Ref(model.SymbolIDFor(symbol.Namespace, symbol.Name))
	case ast.ExprLength:
		if len(expr.Children) > 0 {
			return t.valueFromExpr(env, expr.Children[0], model.TypeRef{Kind: model.TypeInt})
		}
		return Missing()
	default:
		return Symbol(expr.Value)
	}
}

func (t *Timeline) sortedChapterCodes() []string {
	codes := make([]string, 0, len(t.Project.Chapters))
	for code := range t.Project.Chapters {
		codes = append(codes, code)
	}
	sort.Slice(codes, func(i, j int) bool {
		left := t.Project.Chapters[codes[i]].Code
		right := t.Project.Chapters[codes[j]].Code
		return left.Compare(right) < 0
	})
	return codes
}

func (t *Timeline) orderedBeatsForChapter(code string) []*model.Beat {
	chapter := t.Project.Chapters[code]
	if chapter == nil {
		return nil
	}
	for _, field := range chapter.Fields {
		if field.Name != "beats" || field.Value == nil {
			continue
		}
		var beats []*model.Beat
		for _, child := range field.Value.Children {
			beat := t.beatFromExpr(t.envForDecl(chapter.Decl), child)
			if beat != nil {
				beats = append(beats, beat)
			}
		}
		return beats
	}
	return nil
}

func (t *Timeline) beatFromExpr(env resolve.FileEnv, expr *ast.Expr) *model.Beat {
	if expr == nil {
		return nil
	}
	symbol, diagnostics := t.Resolved.ResolveName(env, expr.Value, expr.Span, true)
	if len(diagnostics) > 0 || symbol == nil {
		t.diagnostics = append(t.diagnostics, diagnostics...)
		return nil
	}
	return t.Project.Beats[model.SymbolIDFor(symbol.Namespace, symbol.Name)]
}

func (t *Timeline) effectField(env resolve.FileEnv, expr *ast.Expr) (FieldKey, model.TypeRef, bool) {
	key, fieldType, ok := t.lookupStateFieldLoose(env, expr)
	if ok {
		return key, fieldType, true
	}
	if expr != nil {
		t.addError("E0502", expr.Span, fmt.Sprintf("effect target %s does not resolve to state field", expr.Value))
	}
	return FieldKey{}, model.TypeRef{}, false
}

func (t *Timeline) lookupStateField(env resolve.FileEnv, expr *ast.Expr) (FieldKey, model.TypeRef, bool) {
	if expr == nil {
		return FieldKey{}, model.TypeRef{}, false
	}
	key, fieldType, ok := t.lookupStateFieldLoose(env, expr)
	if !ok {
		return FieldKey{}, model.TypeRef{}, false
	}
	if fieldType.Kind == model.TypeInvalid {
		return FieldKey{}, model.TypeRef{}, false
	}
	return key, fieldType, true
}

func (t *Timeline) lookupStateFieldLoose(env resolve.FileEnv, expr *ast.Expr) (FieldKey, model.TypeRef, bool) {
	if expr == nil {
		return FieldKey{}, model.TypeRef{}, false
	}
	parts := strings.Split(expr.Value, ".")
	for i := len(parts) - 1; i >= 1; i-- {
		prefix := strings.Join(parts[:i], ".")
		symbol, diagnostics := t.Resolved.ResolveName(env, prefix, expr.Span, true)
		if len(diagnostics) > 0 || symbol == nil {
			continue
		}
		entity := t.Project.Entities.All[model.SymbolIDFor(symbol.Namespace, symbol.Name)]
		if entity == nil {
			continue
		}
		fieldName := strings.Join(parts[i:], ".")
		fieldType, ok := t.lookupFieldType(entity, fieldName)
		if !ok {
			return FieldKey{}, model.TypeRef{}, false
		}
		return FieldKey{Entity: entity.ID, Field: fieldName}, fieldType, true
	}
	return FieldKey{}, model.TypeRef{}, false
}

func (t *Timeline) fieldKeyFromRaw(env resolve.FileEnv, raw string, span source.Span) (FieldKey, bool) {
	expr := &ast.Expr{Kind: ast.ExprPath, Value: raw, Span: span}
	key, _, ok := t.effectField(env, expr)
	return key, ok
}

func (t *Timeline) fieldType(entity *model.Entity, fieldName string) model.TypeRef {
	fieldType, ok := t.lookupFieldType(entity, fieldName)
	if !ok {
		return model.TypeRef{Kind: model.TypeSymbol}
	}
	return fieldType
}

func (t *Timeline) lookupFieldType(entity *model.Entity, fieldName string) (model.TypeRef, bool) {
	if field, ok := entity.Fields[fieldName]; ok {
		return field.Type, true
	}
	if entity.Kind == model.EntityCharacter && entity.ClassName != "" {
		if class := t.classForEntity(entity); class != nil {
			if field, ok := class.Fields[fieldName]; ok {
				return field.Type, true
			}
		}
	}
	return model.TypeRef{Kind: model.TypeInvalid}, false
}

func (t *Timeline) classForEntity(entity *model.Entity) *model.Class {
	env := t.envForDecl(entity.Decl)
	symbol, diagnostics := t.Resolved.ResolveName(env, entity.ClassName, entity.Decl.Span, true)
	if len(diagnostics) > 0 || symbol == nil {
		return nil
	}
	return t.classIndex[model.SymbolIDFor(symbol.Namespace, symbol.Name)]
}

func (t *Timeline) envForDecl(decl *ast.Decl) resolve.FileEnv {
	if decl == nil {
		return t.defaultEnv()
	}
	return t.fileEnvs[decl.Span.Start.File]
}

func (t *Timeline) defaultEnv() resolve.FileEnv {
	if len(t.fileEnvs) == 0 {
		return resolve.FileEnv{Imports: map[string]string{}}
	}
	for _, env := range t.fileEnvs {
		if env.Namespace == t.Project.Config.Name {
			return env
		}
	}
	for _, env := range t.fileEnvs {
		return env
	}
	return resolve.FileEnv{Imports: map[string]string{}}
}

func (t *Timeline) addError(code string, span source.Span, message string) {
	t.diagnostics = append(t.diagnostics, source.Error(code, span.Start.File, span.Start.Line, span.Start.Column, message))
}

func elementType(value model.TypeRef) model.TypeRef {
	if value.Elem != nil {
		return *value.Elem
	}
	return model.TypeRef{Kind: model.TypeSymbol}
}
