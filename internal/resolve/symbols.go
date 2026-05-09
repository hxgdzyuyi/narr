package resolve

import (
	"fmt"
	"path/filepath"
	"sort"
	"strings"

	"narr/internal/ast"
	"narr/internal/model"
	"narr/internal/project"
	"narr/internal/source"
)

type SymbolKind string

const (
	SymbolNovel        SymbolKind = "novel"
	SymbolEnum         SymbolKind = "enum"
	SymbolClass        SymbolKind = "class"
	SymbolVolume       SymbolKind = "volume"
	SymbolChapter      SymbolKind = "chapter"
	SymbolBeat         SymbolKind = "beat"
	SymbolStartPattern SymbolKind = "start_pattern"
	SymbolPlace        SymbolKind = "place"
	SymbolCharacter    SymbolKind = "character"
	SymbolCollective   SymbolKind = "collective"
	SymbolFaction      SymbolKind = "faction"
	SymbolObject       SymbolKind = "object"
	SymbolFact         SymbolKind = "fact"
	SymbolPromise      SymbolKind = "promise"
	SymbolThread       SymbolKind = "thread"
	SymbolArc          SymbolKind = "arc"
	SymbolInvariant    SymbolKind = "invariant"
	SymbolStyleNote    SymbolKind = "style_note"
)

type Symbol struct {
	Namespace string
	Name      string
	Kind      SymbolKind
	Decl      *ast.Decl
	File      *ast.File
	Span      source.Span
}

type Namespace struct {
	Name    string
	Files   []*ast.File
	Symbols map[string]*Symbol
}

type FileEnv struct {
	Namespace string
	Imports   map[string]string
}

type Project struct {
	Config         project.Config
	Root           string
	Files          []*ast.File
	Namespaces     map[string]*Namespace
	FileEnvs       map[*ast.File]FileEnv
	ChapterAliases map[string][]*Symbol
	VolumeAliases  map[string][]*Symbol
	ChapterCodes   map[string]*Symbol
	VolumeCodes    map[string]*Symbol
}

func Build(loaded *project.Project, files []*ast.File) (*Project, []source.Diagnostic) {
	resolved := &Project{
		Config:         loaded.Config,
		Root:           loaded.Root,
		Files:          files,
		Namespaces:     map[string]*Namespace{},
		FileEnvs:       map[*ast.File]FileEnv{},
		ChapterAliases: map[string][]*Symbol{},
		VolumeAliases:  map[string][]*Symbol{},
		ChapterCodes:   map[string]*Symbol{},
		VolumeCodes:    map[string]*Symbol{},
	}
	var diagnostics []source.Diagnostic

	for _, file := range files {
		if file.Namespace == "" {
			continue
		}
		namespace := resolved.ensureNamespace(file.Namespace)
		namespace.Files = append(namespace.Files, file)
	}

	for _, file := range files {
		if file.Mode != ast.ModeNarr {
			continue
		}
		for i := range file.Decls {
			diagnostics = append(diagnostics, resolved.addDeclSymbol(file, &file.Decls[i])...)
		}
	}

	for _, file := range files {
		env, importDiagnostics := resolved.buildFileEnv(file)
		resolved.FileEnvs[file] = env
		diagnostics = append(diagnostics, importDiagnostics...)
	}

	if source.HasErrors(diagnostics) {
		return resolved, diagnostics
	}

	resolver := newRefResolver(resolved)
	for _, file := range files {
		resolver.resolveFile(file)
	}
	diagnostics = append(diagnostics, resolver.diagnostics...)
	return resolved, diagnostics
}

func (p *Project) ensureNamespace(name string) *Namespace {
	namespace, ok := p.Namespaces[name]
	if ok {
		return namespace
	}
	namespace = &Namespace{Name: name, Symbols: map[string]*Symbol{}}
	p.Namespaces[name] = namespace
	return namespace
}

func (p *Project) addDeclSymbol(file *ast.File, decl *ast.Decl) []source.Diagnostic {
	namespace := p.ensureNamespace(file.Namespace)
	symbols, diagnostics := symbolsForDecl(file, decl)
	canonicalChapter := ""
	if decl.Kind == ast.DeclChapter {
		canonicalChapter, _ = canonicalChapterCode(decl.Code)
	}
	canonicalVolume := ""
	if decl.Kind == ast.DeclVolume {
		canonicalVolume, _ = canonicalVolumeCode(decl.Code)
	}
	for _, symbol := range symbols {
		switch symbol.Kind {
		case SymbolChapter:
			if symbol.Name == canonicalChapter && p.ChapterCodes[symbol.Name] != nil {
				diagnostics = append(diagnostics, source.Error("E0302", symbol.Span.Start.File, symbol.Span.Start.Line, symbol.Span.Start.Column, fmt.Sprintf("duplicate chapter code %s", symbol.Name)))
				continue
			}
		case SymbolVolume:
			if symbol.Name == canonicalVolume && p.VolumeCodes[symbol.Name] != nil && p.VolumeCodes[symbol.Name].Namespace == symbol.Namespace {
				diagnostics = append(diagnostics, source.Error("E0303", symbol.Span.Start.File, symbol.Span.Start.Line, symbol.Span.Start.Column, fmt.Sprintf("duplicate volume code %s", symbol.Name)))
				continue
			}
		}
		if existing := namespace.Symbols[symbol.Name]; existing != nil {
			diagnostics = append(diagnostics, source.Error("E0301", symbol.Span.Start.File, symbol.Span.Start.Line, symbol.Span.Start.Column, fmt.Sprintf("duplicate declaration %s.%s", symbol.Namespace, symbol.Name)))
			continue
		}
		namespace.Symbols[symbol.Name] = symbol
		if symbol.Kind == SymbolChapter && symbol.Name == canonicalChapter {
			p.ChapterCodes[symbol.Name] = symbol
		}
		if symbol.Kind == SymbolVolume && symbol.Name == canonicalVolume {
			p.VolumeCodes[symbol.Name] = symbol
		}
	}
	if decl.Kind == ast.DeclChapter && decl.Alias != "" {
		canonical, err := canonicalChapterCode(decl.Code)
		if err == nil {
			p.ChapterAliases[decl.Alias] = append(p.ChapterAliases[decl.Alias], namespace.Symbols[canonical])
		}
	}
	if decl.Kind == ast.DeclVolume && decl.Alias != "" {
		canonical, err := canonicalVolumeCode(decl.Code)
		if err == nil {
			p.VolumeAliases[decl.Alias] = append(p.VolumeAliases[decl.Alias], namespace.Symbols[canonical])
		}
	}
	return diagnostics
}

func symbolsForDecl(file *ast.File, decl *ast.Decl) ([]*Symbol, []source.Diagnostic) {
	var symbols []*Symbol
	var diagnostics []source.Diagnostic
	kind := SymbolKind(decl.Kind)
	switch decl.Kind {
	case ast.DeclVolume:
		canonical, err := canonicalVolumeCode(decl.Code)
		if err != nil {
			diagnostics = append(diagnostics, source.Error("E0304", decl.Span.Start.File, decl.Span.Start.Line, decl.Span.Start.Column, err.Error()))
			return nil, diagnostics
		}
		symbols = append(symbols, newSymbol(file, decl, canonical, SymbolVolume))
		if decl.Alias != "" {
			symbols = append(symbols, newSymbol(file, decl, decl.Alias, SymbolVolume))
		}
	case ast.DeclChapter:
		canonical, err := canonicalChapterCode(decl.Code)
		if err != nil {
			diagnostics = append(diagnostics, source.Error("E0305", decl.Span.Start.File, decl.Span.Start.Line, decl.Span.Start.Column, err.Error()))
			return nil, diagnostics
		}
		symbols = append(symbols, newSymbol(file, decl, canonical, SymbolChapter))
		if decl.Alias != "" {
			symbols = append(symbols, newSymbol(file, decl, decl.Alias, SymbolChapter))
		}
	default:
		name := decl.Name
		if name == "" && decl.Code != "" {
			name = decl.Code
		}
		if name == "" {
			return nil, diagnostics
		}
		symbols = append(symbols, newSymbol(file, decl, name, kind))
	}
	return symbols, diagnostics
}

func newSymbol(file *ast.File, decl *ast.Decl, name string, kind SymbolKind) *Symbol {
	return &Symbol{
		Namespace: file.Namespace,
		Name:      name,
		Kind:      kind,
		Decl:      decl,
		File:      file,
		Span:      decl.Span,
	}
}

func (p *Project) buildFileEnv(file *ast.File) (FileEnv, []source.Diagnostic) {
	env := FileEnv{Namespace: file.Namespace, Imports: map[string]string{}}
	var diagnostics []source.Diagnostic
	for _, importDecl := range file.Imports {
		if _, ok := p.Namespaces[importDecl.Path]; !ok {
			diagnostics = append(diagnostics, source.Error("E0306", importDecl.Span.Start.File, importDecl.Span.Start.Line, importDecl.Span.Start.Column, fmt.Sprintf("unknown namespace %s", importDecl.Path)))
			continue
		}
		alias := importDecl.Alias
		if alias == "" {
			alias = lastNamespacePart(importDecl.Path)
		}
		if existing, ok := env.Imports[alias]; ok && existing != importDecl.Path {
			diagnostics = append(diagnostics, source.Error("E0307", importDecl.Span.Start.File, importDecl.Span.Start.Line, importDecl.Span.Start.Column, fmt.Sprintf("import alias %s conflicts with %s", alias, existing)))
			continue
		}
		env.Imports[alias] = importDecl.Path
	}
	return env, diagnostics
}

func canonicalChapterCode(raw string) (string, error) {
	code, err := model.ParseChapterCode(raw)
	if err != nil {
		return "", err
	}
	return code.Canonical(), nil
}

func canonicalVolumeCode(raw string) (string, error) {
	code, err := model.ParseVolumeCode(raw)
	if err != nil {
		return "", err
	}
	return code.Canonical(), nil
}

func lastNamespacePart(namespace string) string {
	parts := strings.Split(namespace, ".")
	return parts[len(parts)-1]
}

func (p *Project) QueryEnv(loaded *project.Project, namespace string) (FileEnv, []source.Diagnostic) {
	if namespace != "" {
		return p.namespaceQueryEnv(namespace)
	}

	main := loaded.Config.Main
	if main != "" {
		for _, file := range p.Files {
			rel, err := filepath.Rel(loaded.Root, file.Path)
			if err != nil {
				continue
			}
			if filepath.ToSlash(rel) == filepath.ToSlash(main) {
				return p.FileEnvs[file], nil
			}
		}
	}

	projectNamespace := loaded.Config.Name
	if projectNamespace == "" {
		return FileEnv{}, []source.Diagnostic{source.Error("E0308", "", 0, 0, "cannot determine query namespace")}
	}
	if _, ok := p.Namespaces[projectNamespace]; !ok {
		return FileEnv{Namespace: projectNamespace, Imports: map[string]string{}}, nil
	}
	return p.namespaceQueryEnv(projectNamespace)
}

func (p *Project) namespaceQueryEnv(namespace string) (FileEnv, []source.Diagnostic) {
	ns, ok := p.Namespaces[namespace]
	if !ok {
		return FileEnv{}, []source.Diagnostic{source.Error("E0309", "", 0, 0, fmt.Sprintf("unknown query namespace %s", namespace))}
	}
	env := FileEnv{Namespace: namespace, Imports: map[string]string{}}
	var diagnostics []source.Diagnostic
	for _, file := range ns.Files {
		fileEnv := p.FileEnvs[file]
		for alias, path := range fileEnv.Imports {
			if existing, ok := env.Imports[alias]; ok && existing != path {
				diagnostics = append(diagnostics, source.Error("E0310", file.Span.Start.File, file.Span.Start.Line, file.Span.Start.Column, fmt.Sprintf("query namespace import alias %s is ambiguous", alias)))
				continue
			}
			env.Imports[alias] = path
		}
	}
	return env, diagnostics
}

func (p *Project) SortedNamespaces() []string {
	names := make([]string, 0, len(p.Namespaces))
	for name := range p.Namespaces {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}
