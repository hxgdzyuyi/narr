package check

import (
	"fmt"
	"strings"

	"narr/internal/ast"
	"narr/internal/model"
	"narr/internal/project"
	"narr/internal/resolve"
	"narr/internal/source"
)

type Result struct {
	Model *model.Project
}

type Checker struct {
	loaded       *project.Project
	files        []*ast.File
	resolved     *resolve.Project
	model        *model.Project
	fileEnvs     map[string]resolve.FileEnv
	diagnostics  []source.Diagnostic
	classByID    map[model.SymbolID]*model.Class
	enumByID     map[model.SymbolID]*model.Enum
	entityByID   map[model.SymbolID]*model.Entity
	entityByDecl map[*ast.Decl]*model.Entity
}

func Check(loaded *project.Project, files []*ast.File, resolved *resolve.Project) (*Result, []source.Diagnostic) {
	modelProject, diagnostics := model.Build(loaded, files)
	checker := &Checker{
		loaded:       loaded,
		files:        files,
		resolved:     resolved,
		model:        modelProject,
		fileEnvs:     map[string]resolve.FileEnv{},
		classByID:    modelProject.Entities.Classes,
		enumByID:     modelProject.Entities.Enums,
		entityByID:   modelProject.Entities.All,
		entityByDecl: map[*ast.Decl]*model.Entity{},
	}
	diagnostics = append(diagnostics, checker.prepare()...)
	if source.HasErrors(diagnostics) {
		return &Result{Model: modelProject}, diagnostics
	}
	checker.checkSchemas()
	checker.checkTestSemantics()
	checker.checkTypes()
	checker.checkEntities()
	checker.checkEffects()
	diagnostics = append(diagnostics, checker.diagnostics...)
	return &Result{Model: modelProject}, diagnostics
}

func (c *Checker) prepare() []source.Diagnostic {
	for _, file := range c.files {
		c.fileEnvs[file.Path] = c.resolved.FileEnvs[file]
	}
	for _, entity := range c.model.Entities.All {
		c.entityByDecl[entity.Decl] = entity
	}
	return nil
}

func (c *Checker) checkTypes() {
	for _, class := range c.model.Entities.Classes {
		env := c.envForDecl(class.Decl)
		for _, field := range class.Fields {
			c.checkTypeExists(env, field.Type, field.Span)
			if field.Default != nil {
				c.checkValueAssignable(env, field.Type, field.Default)
			}
		}
	}
	for _, entity := range c.model.Entities.All {
		env := c.envForDecl(entity.Decl)
		for _, field := range entity.Fields {
			c.checkTypeExists(env, field.Type, field.Span)
			if field.Default != nil {
				c.checkValueAssignable(env, field.Type, field.Default)
			}
		}
	}
}

func (c *Checker) checkEntities() {
	for _, entity := range c.model.Entities.All {
		env := c.envForDecl(entity.Decl)
		for _, stmt := range entity.Initializers {
			if stmt.Target == nil {
				continue
			}
			field, ok := c.fieldForEntity(entity, stmt.Target.Value)
			if !ok {
				c.addError("E0404", stmt.Target.Span, fmt.Sprintf("%s %s has no field %s", entity.Kind, entity.Name, stmt.Target.Value))
				continue
			}
			c.checkStatementValue(env, field.Type, stmt)
		}
	}
}

func (c *Checker) checkEffects() {
	for _, beat := range c.model.Beats {
		env := c.envForDecl(beat.Decl)
		for _, stmt := range beat.Effects {
			target, ok := c.effectTarget(env, stmt.Target)
			if !ok {
				continue
			}
			c.checkStatementValue(env, target.Field.Type, stmt)
		}
	}
}

func (c *Checker) checkStatementValue(env resolve.FileEnv, fieldType model.TypeRef, stmt ast.Stmt) {
	switch stmt.Kind {
	case ast.StmtAssignment, ast.StmtInit:
		c.checkValueAssignable(env, fieldType, stmt.Value)
	case ast.StmtSetAdd, ast.StmtSetRemove:
		if fieldType.Kind != model.TypeSet || fieldType.Elem == nil {
			c.addError("E0405", stmt.Span, fmt.Sprintf("operator %s requires Set field, got %s", stmt.Op, fieldType.String()))
			return
		}
		c.checkValueAssignable(env, *fieldType.Elem, stmt.Value)
	case ast.StmtListAppend:
		if fieldType.Kind != model.TypeList || fieldType.Elem == nil {
			c.addError("E0406", stmt.Span, fmt.Sprintf("operator append requires List field, got %s", fieldType.String()))
			return
		}
		c.checkValueAssignable(env, *fieldType.Elem, stmt.Value)
	}
}

type effectTarget struct {
	Entity *model.Entity
	Field  model.FieldDef
}

func (c *Checker) effectTarget(env resolve.FileEnv, expr *ast.Expr) (effectTarget, bool) {
	if expr == nil {
		return effectTarget{}, false
	}
	parts := strings.Split(expr.Value, ".")
	if len(parts) < 2 {
		c.addError("E0407", expr.Span, fmt.Sprintf("effect target %s must include entity and field", expr.Value))
		return effectTarget{}, false
	}
	for i := len(parts) - 1; i >= 1; i-- {
		prefix := strings.Join(parts[:i], ".")
		symbol, diagnostics := c.resolved.ResolveName(env, prefix, expr.Span, true)
		if len(diagnostics) > 0 {
			continue
		}
		entity := c.entityForSymbol(symbol)
		if entity == nil {
			continue
		}
		fieldName := strings.Join(parts[i:], ".")
		field, ok := c.fieldForEntity(entity, fieldName)
		if !ok {
			c.addError("E0404", expr.Span, fmt.Sprintf("%s %s has no field %s", entity.Kind, entity.Name, fieldName))
			return effectTarget{}, false
		}
		return effectTarget{Entity: entity, Field: field}, true
	}
	c.addError("E0408", expr.Span, fmt.Sprintf("effect target %s does not resolve to an entity", expr.Value))
	return effectTarget{}, false
}

func (c *Checker) fieldForEntity(entity *model.Entity, fieldName string) (model.FieldDef, bool) {
	if field, ok := entity.Fields[fieldName]; ok {
		return field, true
	}
	if entity.Kind != model.EntityCharacter || entity.ClassName == "" {
		return model.FieldDef{}, false
	}
	class := c.classForRef(c.envForDecl(entity.Decl), entity.ClassName, entity.Decl.Span)
	if class == nil {
		return model.FieldDef{}, false
	}
	field, ok := class.Fields[fieldName]
	return field, ok
}

func (c *Checker) checkValueAssignable(env resolve.FileEnv, target model.TypeRef, expr *ast.Expr) {
	if expr == nil {
		return
	}
	switch target.Kind {
	case model.TypeInvalid:
		return
	case model.TypeSymbol:
		return
	case model.TypeClaim:
		if expr.Kind == ast.ExprString || expr.Kind == ast.ExprMultiline || expr.Kind == ast.ExprRef || expr.Kind == ast.ExprPath {
			return
		}
		c.addError("E0409", expr.Span, fmt.Sprintf("value is not compatible with %s", target.String()))
	case model.TypeBool:
		if expr.Kind != ast.ExprBool {
			c.addError("E0409", expr.Span, "value is not compatible with Bool")
		}
	case model.TypeInt:
		if expr.Kind != ast.ExprInteger {
			c.addError("E0409", expr.Span, "value is not compatible with Int")
		}
	case model.TypeString, model.TypeText:
		if expr.Kind != ast.ExprString && expr.Kind != ast.ExprMultiline {
			c.addError("E0409", expr.Span, fmt.Sprintf("value is not compatible with %s", target.String()))
		}
	case model.TypeSet:
		c.checkCollectionValue(env, target, expr, ast.ExprSet)
	case model.TypeList:
		c.checkCollectionValue(env, target, expr, ast.ExprList)
	case model.TypeEnum:
		c.checkEnumValue(env, target, expr)
	default:
		c.checkRefValue(env, target, expr)
	}
}

func (c *Checker) checkCollectionValue(env resolve.FileEnv, target model.TypeRef, expr *ast.Expr, kind ast.ExprKind) {
	if expr.Kind != kind {
		c.addError("E0409", expr.Span, fmt.Sprintf("value is not compatible with %s", target.String()))
		return
	}
	if target.Elem == nil {
		return
	}
	for _, child := range expr.Children {
		c.checkValueAssignable(env, *target.Elem, child)
	}
}

func (c *Checker) checkEnumValue(env resolve.FileEnv, target model.TypeRef, expr *ast.Expr) {
	enum := c.enumForType(env, target, expr.Span)
	if enum == nil {
		return
	}
	if expr.Kind != ast.ExprRef || strings.Contains(expr.Value, ".") {
		c.addError("E0410", expr.Span, fmt.Sprintf("enum %s requires a bare member", enum.Name))
		return
	}
	if !enum.Members[expr.Value] {
		c.addError("E0411", expr.Span, fmt.Sprintf("%s is not a member of enum %s", expr.Value, enum.Name))
	}
}

func (c *Checker) checkRefValue(env resolve.FileEnv, target model.TypeRef, expr *ast.Expr) {
	if expr.Kind != ast.ExprRef && expr.Kind != ast.ExprPath {
		c.addError("E0409", expr.Span, fmt.Sprintf("value is not compatible with %s", target.String()))
		return
	}
	symbol, diagnostics := c.resolved.ResolveName(env, expr.Value, expr.Span, true)
	if len(diagnostics) > 0 {
		c.diagnostics = append(c.diagnostics, diagnostics...)
		return
	}
	if symbol == nil {
		c.addError("E0412", expr.Span, fmt.Sprintf("value %s does not resolve", expr.Value))
		return
	}
	if !symbolMatchesType(symbol, target) {
		c.addError("E0409", expr.Span, fmt.Sprintf("%s is %s, not compatible with %s", expr.Value, symbol.Kind, target.String()))
	}
}

func (c *Checker) checkTypeExists(env resolve.FileEnv, target model.TypeRef, span source.Span) {
	switch target.Kind {
	case model.TypeInvalid:
		c.addError("E0413", span, "invalid type expression")
	case model.TypeSet, model.TypeList:
		if target.Elem != nil {
			c.checkTypeExists(env, *target.Elem, span)
		}
	case model.TypeEnum:
		if c.enumForType(env, target, span) != nil {
			return
		}
		if c.classForRef(env, target.Name, span) != nil {
			return
		}
		c.addError("E0414", span, fmt.Sprintf("unknown type %s", target.Name))
	}
}

func (c *Checker) enumForType(env resolve.FileEnv, target model.TypeRef, span source.Span) *model.Enum {
	symbol, diagnostics := c.resolved.ResolveName(env, target.Name, span, true)
	if len(diagnostics) > 0 || symbol == nil || symbol.Kind != resolve.SymbolEnum {
		return nil
	}
	return c.enumByID[model.SymbolIDFor(symbol.Namespace, symbol.Name)]
}

func (c *Checker) classForRef(env resolve.FileEnv, raw string, span source.Span) *model.Class {
	symbol, diagnostics := c.resolved.ResolveName(env, raw, span, true)
	if len(diagnostics) > 0 || symbol == nil || symbol.Kind != resolve.SymbolClass {
		return nil
	}
	return c.classByID[model.SymbolIDFor(symbol.Namespace, symbol.Name)]
}

func (c *Checker) entityForSymbol(symbol *resolve.Symbol) *model.Entity {
	if symbol == nil {
		return nil
	}
	return c.entityByID[model.SymbolIDFor(symbol.Namespace, symbol.Name)]
}

func (c *Checker) envForDecl(decl *ast.Decl) resolve.FileEnv {
	if decl == nil {
		return resolve.FileEnv{Imports: map[string]string{}}
	}
	return c.fileEnvs[decl.Span.Start.File]
}

func (c *Checker) addError(code string, span source.Span, message string) {
	c.diagnostics = append(c.diagnostics, source.Error(code, span.Start.File, span.Start.Line, span.Start.Column, message))
}

func symbolMatchesType(symbol *resolve.Symbol, target model.TypeRef) bool {
	switch target.Kind {
	case model.TypePlace:
		return symbol.Kind == resolve.SymbolPlace
	case model.TypeCharacter:
		return symbol.Kind == resolve.SymbolCharacter
	case model.TypeObject:
		return symbol.Kind == resolve.SymbolObject
	case model.TypeFaction:
		return symbol.Kind == resolve.SymbolFaction
	case model.TypeFact:
		return symbol.Kind == resolve.SymbolFact
	case model.TypeNovel:
		return symbol.Kind == resolve.SymbolNovel
	case model.TypeVolume:
		return symbol.Kind == resolve.SymbolVolume
	case model.TypeChapter:
		return symbol.Kind == resolve.SymbolChapter
	case model.TypeBeat:
		return symbol.Kind == resolve.SymbolBeat
	case model.TypeThread:
		return symbol.Kind == resolve.SymbolThread
	case model.TypePromise:
		return symbol.Kind == resolve.SymbolPromise
	case model.TypeArc:
		return symbol.Kind == resolve.SymbolArc
	case model.TypeStartPattern:
		return symbol.Kind == resolve.SymbolStartPattern
	case model.TypeInvariant:
		return symbol.Kind == resolve.SymbolInvariant
	default:
		return false
	}
}
