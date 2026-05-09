package model

import (
	"narr/internal/ast"
	"narr/internal/project"
	"narr/internal/source"
)

type astSpan = source.Span

type Namespace struct {
	Name string
}

type Project struct {
	Config        project.Config
	Namespaces    map[string]*Namespace
	Novel         *Novel
	Volumes       map[string]*Volume
	Chapters      map[string]*Chapter
	Beats         map[SymbolID]*Beat
	StartPatterns map[SymbolID]*StartPattern
	Threads       map[SymbolID]*Thread
	Promises      map[SymbolID]*Promise
	Arcs          map[SymbolID]*Arc
	Invariants    map[SymbolID]*Invariant
	Entities      *Entities
}

func Build(loaded *project.Project, files []*ast.File) (*Project, []source.Diagnostic) {
	modelProject := &Project{
		Config:        loaded.Config,
		Namespaces:    map[string]*Namespace{},
		Volumes:       map[string]*Volume{},
		Chapters:      map[string]*Chapter{},
		Beats:         map[SymbolID]*Beat{},
		StartPatterns: map[SymbolID]*StartPattern{},
		Threads:       map[SymbolID]*Thread{},
		Promises:      map[SymbolID]*Promise{},
		Arcs:          map[SymbolID]*Arc{},
		Invariants:    map[SymbolID]*Invariant{},
		Entities:      NewEntities(),
	}
	var diagnostics []source.Diagnostic

	for _, file := range files {
		if file.Namespace != "" {
			modelProject.Namespaces[file.Namespace] = &Namespace{Name: file.Namespace}
		}
		if file.Mode != ast.ModeNarr {
			continue
		}
		for i := range file.Decls {
			diagnostics = append(diagnostics, modelProject.addDecl(file, &file.Decls[i])...)
		}
	}
	return modelProject, diagnostics
}

func (p *Project) addDecl(file *ast.File, decl *ast.Decl) []source.Diagnostic {
	id := SymbolIDFor(file.Namespace, decl.Name)
	switch decl.Kind {
	case ast.DeclNovel:
		name := decl.Name
		if name == "" {
			name = p.Config.Name
		}
		p.Novel = &Novel{ID: SymbolIDFor(file.Namespace, name), Name: name, Fields: fieldValues(decl.Fields), Decl: decl}
	case ast.DeclVolume:
		code, err := ParseVolumeCode(decl.Code)
		if err != nil {
			return []source.Diagnostic{source.Error("E0401", decl.Span.Start.File, decl.Span.Start.Line, decl.Span.Start.Column, err.Error())}
		}
		p.Volumes[code.Canonical()] = &Volume{ID: SymbolIDFor(file.Namespace, code.Canonical()), Code: code, Alias: decl.Alias, Fields: fieldValues(decl.Fields), Decl: decl}
	case ast.DeclChapter:
		code, err := ParseChapterCode(decl.Code)
		if err != nil {
			return []source.Diagnostic{source.Error("E0402", decl.Span.Start.File, decl.Span.Start.Line, decl.Span.Start.Column, err.Error())}
		}
		p.Chapters[code.Canonical()] = &Chapter{ID: SymbolIDFor(file.Namespace, code.Canonical()), Code: code, Alias: decl.Alias, Fields: fieldValues(decl.Fields), Decl: decl}
	case ast.DeclBeat:
		p.Beats[id] = &Beat{ID: id, Name: decl.Name, Anchor: decl.Anchor, Fields: fieldValues(decl.Fields), Effects: effectStatements(decl.Fields), Decl: decl}
	case ast.DeclStartPattern:
		p.StartPatterns[id] = &StartPattern{ID: id, Name: decl.Name, Fields: fieldValues(decl.Fields), Decl: decl}
	case ast.DeclThread:
		p.Threads[id] = &Thread{ID: id, Name: decl.Name, Fields: fieldValues(decl.Fields), Decl: decl}
	case ast.DeclPromise:
		p.Promises[id] = &Promise{ID: id, Name: decl.Name, Fields: fieldValues(decl.Fields), Decl: decl}
	case ast.DeclArc:
		p.Arcs[id] = &Arc{ID: id, Name: decl.Name, Fields: fieldValues(decl.Fields), Decl: decl}
	case ast.DeclInvariant:
		p.Invariants[id] = &Invariant{ID: id, Name: decl.Name, Fields: fieldValues(decl.Fields), Decl: decl}
	case ast.DeclEnum:
		enum := &Enum{ID: id, Name: decl.Name, Members: map[string]bool{}, Decl: decl}
		for _, member := range decl.Members {
			enum.Members[member] = true
		}
		p.Entities.Enums[id] = enum
	case ast.DeclClass:
		p.Entities.Classes[id] = &Class{ID: id, Name: decl.Name, Fields: fieldDefs(decl.Fields), Decl: decl}
	case ast.DeclPlace, ast.DeclCharacter, ast.DeclCollective, ast.DeclFaction, ast.DeclObject, ast.DeclFact:
		p.Entities.AddEntity(entityFromDecl(id, decl))
	}
	return nil
}

func fieldValues(fields []ast.Field) []FieldValue {
	out := make([]FieldValue, 0, len(fields))
	for _, field := range fields {
		out = append(out, FieldValue{Name: field.Name, Value: field.Value, Statements: field.Statements})
	}
	return out
}

func effectStatements(fields []ast.Field) []ast.Stmt {
	var out []ast.Stmt
	for _, field := range fields {
		if field.Name == "effect" {
			out = append(out, field.Statements...)
		}
	}
	return out
}

func fieldDefs(fields []ast.Field) map[string]FieldDef {
	defs := map[string]FieldDef{}
	for _, field := range fields {
		if field.Name == "" || field.Value == nil {
			continue
		}
		def := FieldDef{Name: field.Name, Type: TypeFromExpr(field.Value), Span: field.Span}
		for _, stmt := range field.Statements {
			if stmt.Kind == ast.StmtDefault {
				def.Default = stmt.Value
				break
			}
		}
		defs[field.Name] = def
	}
	return defs
}

func entityFromDecl(id SymbolID, decl *ast.Decl) *Entity {
	entity := &Entity{ID: id, Name: decl.Name, Fields: map[string]FieldDef{}, Value: decl.Value, Decl: decl}
	switch decl.Kind {
	case ast.DeclPlace:
		entity.Kind = EntityPlace
	case ast.DeclCharacter:
		entity.Kind = EntityCharacter
		if decl.Class != nil {
			entity.ClassName = decl.Class.Value
		}
	case ast.DeclCollective:
		entity.Kind = EntityCollective
	case ast.DeclFaction:
		entity.Kind = EntityFaction
	case ast.DeclObject:
		entity.Kind = EntityObject
	case ast.DeclFact:
		entity.Kind = EntityFact
	}
	for _, field := range decl.Fields {
		if field.Name != "" && field.Value != nil {
			def := FieldDef{Name: field.Name, Type: TypeFromExpr(field.Value), Span: field.Span}
			for _, stmt := range field.Statements {
				if stmt.Kind == ast.StmtDefault {
					def.Default = stmt.Value
					break
				}
			}
			entity.Fields[field.Name] = def
			continue
		}
		entity.Initializers = append(entity.Initializers, field.Statements...)
	}
	return entity
}

func TypeFromExpr(expr *ast.Expr) TypeRef {
	if expr == nil {
		return TypeRef{Kind: TypeInvalid}
	}
	if expr.Kind == ast.ExprCall && (expr.Value == "Set" || expr.Value == "List") {
		kind := TypeSet
		if expr.Value == "List" {
			kind = TypeList
		}
		elem := TypeRef{Kind: TypeInvalid}
		if len(expr.Args) > 0 {
			elem = TypeFromExpr(expr.Args[0])
		}
		return TypeRef{Kind: kind, Name: expr.Value, Elem: &elem}
	}
	if expr.Kind == ast.ExprRef || expr.Kind == ast.ExprPath {
		if builtin, ok := BuiltinType(expr.Value); ok {
			return builtin
		}
		return TypeRef{Kind: TypeEnum, Name: expr.Value}
	}
	return TypeRef{Kind: TypeInvalid}
}
