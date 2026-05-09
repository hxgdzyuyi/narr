package resolve

import (
	"fmt"
	"strings"

	"narr/internal/ast"
	"narr/internal/source"
)

type refResolver struct {
	project     *Project
	diagnostics []source.Diagnostic
	scopes      []map[string]bool
}

func ResolveQueryExpr(project *Project, env FileEnv, expr *ast.Expr) []source.Diagnostic {
	resolver := newRefResolver(project)
	resolver.resolveExprPaths(env, expr)
	return resolver.diagnostics
}

func newRefResolver(project *Project) *refResolver {
	return &refResolver{project: project}
}

func (r *refResolver) resolveFile(file *ast.File) {
	env := r.project.FileEnvs[file]
	switch file.Mode {
	case ast.ModeNarr:
		for i := range file.Decls {
			r.resolveDecl(env, &file.Decls[i])
		}
	case ast.ModeTest:
		for i := range file.Tests {
			r.resolveTestDecl(env, &file.Tests[i])
		}
	}
}

func (r *refResolver) resolveDecl(env FileEnv, decl *ast.Decl) {
	r.resolveReferenceExpr(env, decl.Anchor, true)
	r.resolveReferenceExpr(env, decl.In, true)
	r.resolveReferenceExpr(env, decl.Class, true)
	r.resolveExprPaths(env, decl.Value)
	for i := range decl.Fields {
		r.resolveField(env, decl.Kind, &decl.Fields[i])
	}
}

func (r *refResolver) resolveField(env FileEnv, declKind ast.DeclKind, field *ast.Field) {
	switch field.Name {
	case "start_pattern", "pov", "location", "setup_at", "payoff_by", "payoff_at",
		"starts_at", "expected_resolution", "resolved_at", "subject", "active_until", "at":
		r.resolveReferenceExpr(env, field.Value, true)
	case "beats", "observers", "sets_up", "pays_off", "advances", "resolves", "reveals", "mentions":
		r.resolveReferenceList(env, field.Value, true)
	case "hidden":
		r.resolveHiddenRule(env, field.Value)
	default:
		r.resolveExprPaths(env, field.Value)
	}

	for i := range field.Statements {
		r.resolveStmt(env, declKind, &field.Statements[i])
	}
}

func (r *refResolver) resolveHiddenRule(env FileEnv, expr *ast.Expr) {
	if expr == nil || expr.Kind != ast.ExprBinary || expr.Op != "until" || len(expr.Children) != 2 {
		r.resolveExprPaths(env, expr)
		return
	}
	r.resolveReferenceExpr(env, expr.Children[0], true)
	r.resolveReferenceExpr(env, expr.Children[1], true)
}

func (r *refResolver) resolveStmt(env FileEnv, declKind ast.DeclKind, stmt *ast.Stmt) {
	switch stmt.Kind {
	case ast.StmtCondition, ast.StmtAssert:
		r.resolveExprPaths(env, stmt.Expr)
		r.resolveExprPaths(env, stmt.Message)
	case ast.StmtAssignment, ast.StmtSetAdd, ast.StmtSetRemove, ast.StmtListAppend:
		if declKind == ast.DeclBeat && stmt.Target != nil && strings.Contains(stmt.Target.Value, ".") {
			r.resolveReferenceExpr(env, stmt.Target, true)
		} else if stmt.Target != nil && strings.Contains(stmt.Target.Value, ".") {
			r.resolveReferenceExpr(env, stmt.Target, false)
		}
		r.resolveExprPaths(env, stmt.Value)
	case ast.StmtStartTarget:
		r.resolveReferenceExpr(env, stmt.Value, true)
	case ast.StmtDefault, ast.StmtLength, ast.StmtInit:
		r.resolveExprPaths(env, stmt.Value)
	case ast.StmtLet, ast.StmtForall, ast.StmtExists:
		r.resolveTestStmt(env, stmt)
	default:
		r.resolveExprPaths(env, stmt.Target)
		r.resolveExprPaths(env, stmt.Value)
		r.resolveExprPaths(env, stmt.Expr)
	}
}

func (r *refResolver) resolveTestDecl(env FileEnv, test *ast.TestDecl) {
	r.pushScope()
	defer r.popScope()
	for i := range test.Statements {
		r.resolveTestStmt(env, &test.Statements[i])
	}
}

func (r *refResolver) resolveTestStmt(env FileEnv, stmt *ast.Stmt) {
	switch stmt.Kind {
	case ast.StmtAssert:
		r.resolveExprPaths(env, stmt.Expr)
		r.resolveExprPaths(env, stmt.Message)
	case ast.StmtLet:
		r.resolveExprPaths(env, stmt.Value)
		r.defineLocal(stmt.Name)
	case ast.StmtForall, ast.StmtExists:
		if stmt.Binder != nil {
			r.resolveExprPaths(env, stmt.Binder.In)
			r.pushScope()
			r.defineLocal(stmt.Binder.Name)
			r.resolveExprPaths(env, stmt.Binder.Where)
			for i := range stmt.Body {
				r.resolveTestStmt(env, &stmt.Body[i])
			}
			r.popScope()
		}
	default:
		r.resolveStmt(env, "", stmt)
	}
}

func (r *refResolver) resolveExprPaths(env FileEnv, expr *ast.Expr) {
	if expr == nil {
		return
	}
	switch expr.Kind {
	case ast.ExprInvalid, ast.ExprString, ast.ExprMultiline, ast.ExprInteger, ast.ExprBool, ast.ExprSymbol, ast.ExprLanguage, ast.ExprCollection:
		return
	case ast.ExprRef:
		if r.isLocalPath(expr.Value) {
			return
		}
	case ast.ExprPath:
		r.resolveReferenceExpr(env, expr, true)
	case ast.ExprList, ast.ExprSet, ast.ExprLength, ast.ExprUnary, ast.ExprBinary, ast.ExprPostfix, ast.ExprParen:
		for _, child := range expr.Children {
			r.resolveExprPaths(env, child)
		}
	case ast.ExprCall:
		for _, arg := range expr.Args {
			r.resolveExprPaths(env, arg)
		}
	case ast.ExprCount:
		if expr.Binder != nil {
			r.resolveExprBinder(env, expr.Binder)
			return
		}
		for _, child := range expr.Children {
			r.resolveExprPaths(env, child)
		}
	case ast.ExprCollect:
		if expr.Binder != nil {
			r.resolveExprPaths(env, expr.Binder.In)
			r.pushScope()
			r.defineLocal(expr.Binder.Name)
			for _, child := range expr.Children {
				r.resolveExprPaths(env, child)
			}
			r.resolveExprPaths(env, expr.Binder.Where)
			r.popScope()
		} else {
			for _, child := range expr.Children {
				r.resolveExprPaths(env, child)
			}
		}
	case ast.ExprState:
		if len(expr.Children) > 0 {
			r.resolveReferenceExpr(env, expr.Children[0], true)
		}
		for _, child := range expr.Children[1:] {
			r.resolveExprPaths(env, child)
		}
	default:
		for _, child := range expr.Children {
			r.resolveExprPaths(env, child)
		}
		for _, arg := range expr.Args {
			r.resolveExprPaths(env, arg)
		}
	}
}

func (r *refResolver) resolveExprBinder(env FileEnv, binder *ast.Binder) {
	r.resolveExprPaths(env, binder.In)
	r.pushScope()
	r.defineLocal(binder.Name)
	r.resolveExprPaths(env, binder.Where)
	r.popScope()
}

func (r *refResolver) resolveReferenceList(env FileEnv, expr *ast.Expr, required bool) {
	if expr == nil {
		return
	}
	switch expr.Kind {
	case ast.ExprList, ast.ExprSet:
		for _, child := range expr.Children {
			r.resolveReferenceExpr(env, child, required)
		}
	default:
		r.resolveReferenceExpr(env, expr, required)
	}
}

func (r *refResolver) resolveReferenceExpr(env FileEnv, expr *ast.Expr, required bool) {
	if expr == nil {
		return
	}
	switch expr.Kind {
	case ast.ExprRef, ast.ExprPath:
		if isBuiltinAnchorRef(expr.Value) {
			return
		}
		if r.isLocalPath(expr.Value) {
			return
		}
		if _, diagnostics := r.project.ResolveName(env, expr.Value, expr.Span, required); len(diagnostics) > 0 {
			r.diagnostics = append(r.diagnostics, diagnostics...)
		}
	case ast.ExprList, ast.ExprSet:
		for _, child := range expr.Children {
			r.resolveReferenceExpr(env, child, required)
		}
	case ast.ExprParen:
		for _, child := range expr.Children {
			r.resolveReferenceExpr(env, child, required)
		}
	default:
		r.resolveExprPaths(env, expr)
	}
}

func (r *refResolver) pushScope() {
	r.scopes = append(r.scopes, map[string]bool{})
}

func (r *refResolver) popScope() {
	if len(r.scopes) > 0 {
		r.scopes = r.scopes[:len(r.scopes)-1]
	}
}

func (r *refResolver) defineLocal(name string) {
	if name == "" {
		return
	}
	if len(r.scopes) == 0 {
		r.pushScope()
	}
	r.scopes[len(r.scopes)-1][name] = true
}

func (r *refResolver) isLocalPath(path string) bool {
	first := firstPathSegment(path)
	for i := len(r.scopes) - 1; i >= 0; i-- {
		if r.scopes[i][first] {
			return true
		}
	}
	return false
}

func (p *Project) ResolveName(env FileEnv, raw string, span source.Span, required bool) (*Symbol, []source.Diagnostic) {
	if raw == "" {
		return nil, nil
	}
	parts := strings.Split(raw, ".")
	if len(parts) == 0 {
		return nil, nil
	}

	if namespace, ok := env.Imports[parts[0]]; ok {
		symbol := p.resolveLocalPath(namespace, parts[1:])
		if symbol != nil {
			return symbol, nil
		}
		return nil, []source.Diagnostic{unknownReference(span, fmt.Sprintf("%s in namespace %s", raw, namespace))}
	}

	if symbol := p.resolveLocalPath(env.Namespace, parts); symbol != nil {
		return symbol, nil
	}

	if symbol := p.resolveFullyQualified(parts); symbol != nil {
		return symbol, nil
	}

	if len(parts) == 1 {
		if symbol, diagnostics := p.resolveGlobalAlias(parts[0], span); symbol != nil || len(diagnostics) > 0 {
			return symbol, diagnostics
		}
	}

	if !required && len(parts) == 1 {
		return nil, nil
	}
	return nil, []source.Diagnostic{unknownReference(span, raw)}
}

func (p *Project) resolveLocalPath(namespace string, parts []string) *Symbol {
	ns := p.Namespaces[namespace]
	if ns == nil || len(parts) == 0 {
		return nil
	}
	parts = trimAnchorSuffix(parts)
	for i := len(parts); i >= 1; i-- {
		candidate := strings.Join(parts[:i], ".")
		if symbol := ns.Symbols[candidate]; symbol != nil {
			return symbol
		}
		if i == 2 {
			if canonical, err := canonicalChapterCode(candidate); err == nil {
				if symbol := ns.Symbols[canonical]; symbol != nil {
					return symbol
				}
			}
		}
		if i == 1 {
			if canonical, err := canonicalVolumeCode(candidate); err == nil {
				if symbol := ns.Symbols[canonical]; symbol != nil {
					return symbol
				}
			}
		}
	}
	return nil
}

func (p *Project) resolveFullyQualified(parts []string) *Symbol {
	for i := len(parts) - 1; i >= 1; i-- {
		namespace := strings.Join(parts[:i], ".")
		if _, ok := p.Namespaces[namespace]; !ok {
			continue
		}
		if symbol := p.resolveLocalPath(namespace, parts[i:]); symbol != nil {
			return symbol
		}
	}
	return nil
}

func (p *Project) resolveGlobalAlias(name string, span source.Span) (*Symbol, []source.Diagnostic) {
	if matches := compactSymbols(p.ChapterAliases[name]); len(matches) > 0 {
		if len(matches) == 1 {
			return matches[0], nil
		}
		return nil, []source.Diagnostic{source.Error("E0312", span.Start.File, span.Start.Line, span.Start.Column, fmt.Sprintf("chapter alias %s is ambiguous", name))}
	}
	if matches := compactSymbols(p.VolumeAliases[name]); len(matches) > 0 {
		if len(matches) == 1 {
			return matches[0], nil
		}
		return nil, []source.Diagnostic{source.Error("E0313", span.Start.File, span.Start.Line, span.Start.Column, fmt.Sprintf("volume alias %s is ambiguous", name))}
	}
	return nil, nil
}

func trimAnchorSuffix(parts []string) []string {
	if len(parts) <= 1 {
		return parts
	}
	switch parts[len(parts)-1] {
	case "begin", "end", "before", "after":
		return parts[:len(parts)-1]
	default:
		return parts
	}
}

func isBuiltinAnchorRef(value string) bool {
	return value == "beginning" || value == "end_of_story"
}

func firstPathSegment(path string) string {
	before, _, ok := strings.Cut(path, ".")
	if !ok {
		return path
	}
	return before
}

func compactSymbols(symbols []*Symbol) []*Symbol {
	out := symbols[:0]
	for _, symbol := range symbols {
		if symbol != nil {
			out = append(out, symbol)
		}
	}
	return out
}

func unknownReference(span source.Span, name string) source.Diagnostic {
	return source.Error("E0311", span.Start.File, span.Start.Line, span.Start.Column, fmt.Sprintf("unknown reference %s", name))
}
