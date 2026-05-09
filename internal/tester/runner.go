package tester

import (
	"fmt"
	"sort"

	"narr/internal/ast"
	"narr/internal/resolve"
	"narr/internal/source"
	"narr/internal/structure"
)

type Options struct {
	Root     string
	Files    []*ast.File
	Resolved *resolve.Project
	Index    *structure.Index
}

type runner struct {
	root        string
	resolved    *resolve.Project
	index       *structure.Index
	diagnostics []source.Diagnostic
}

func Run(options Options) Report {
	r := &runner{
		root:     options.Root,
		resolved: options.Resolved,
		index:    options.Index,
	}
	report := Report{}
	seenFiles := map[string]bool{}
	for _, file := range options.Files {
		if file == nil || file.Mode != ast.ModeTest {
			continue
		}
		rel := relFile(options.Root, file.Path)
		if !seenFiles[rel] {
			report.Files = append(report.Files, rel)
			seenFiles[rel] = true
		}
		env := options.Resolved.FileEnvs[file]
		for i := range file.Tests {
			result := r.runTest(file, env, &file.Tests[i])
			report.Tests = append(report.Tests, result)
			report.Total++
			if result.Status == "pass" {
				report.Passed++
			} else {
				report.Failed++
			}
		}
	}
	report.Diagnostics = r.diagnostics
	report.OK = report.Failed == 0 && !source.HasErrors(report.Diagnostics)
	return report
}

func (r *runner) runTest(file *ast.File, env resolve.FileEnv, test *ast.TestDecl) TestResult {
	locals := map[string]structure.EvalValue{}
	failures := r.execStatements(env, test.Statements, locals)
	status := "pass"
	if len(failures) > 0 {
		status = "fail"
	}
	return TestResult{
		Name:     test.Name,
		File:     relFile(r.root, file.Path),
		Line:     test.Span.Start.Line,
		Column:   test.Span.Start.Column,
		Status:   status,
		Failures: failures,
	}
}

func (r *runner) execStatements(env resolve.FileEnv, statements []ast.Stmt, locals map[string]structure.EvalValue) []Failure {
	var failures []Failure
	for i := range statements {
		failures = append(failures, r.execStatement(env, &statements[i], locals)...)
	}
	return failures
}

func (r *runner) execStatement(env resolve.FileEnv, stmt *ast.Stmt, locals map[string]structure.EvalValue) []Failure {
	if stmt == nil {
		return nil
	}
	switch stmt.Kind {
	case ast.StmtAssert:
		return r.execAssert(env, stmt, locals)
	case ast.StmtLet:
		return r.execLet(env, stmt, locals)
	case ast.StmtForall:
		return r.execForall(env, stmt, locals)
	case ast.StmtExists:
		return r.execExists(env, stmt, locals)
	default:
		return []Failure{r.failure(stmt.Span, string(stmt.Kind), "", fmt.Sprintf("unsupported test statement %s", stmt.Kind), locals)}
	}
}

func (r *runner) execAssert(env resolve.FileEnv, stmt *ast.Stmt, locals map[string]structure.EvalValue) []Failure {
	value, failures := r.eval(env, stmt.Expr, locals)
	if len(failures) > 0 {
		return failures
	}
	if value.Kind != structure.EvalBool {
		return []Failure{r.failure(stmt.Span, "assert", exprText(stmt.Expr), "assert expression must evaluate to Bool", locals)}
	}
	if structure.EvalTruth(value) {
		return nil
	}
	message := "assertion failed"
	if stmt.Message != nil {
		if text, ok := r.messageText(env, stmt.Message, locals); ok {
			message = text
		}
	}
	return []Failure{r.failure(stmt.Span, "assert", exprText(stmt.Expr), message, locals)}
}

func (r *runner) execLet(env resolve.FileEnv, stmt *ast.Stmt, locals map[string]structure.EvalValue) []Failure {
	value, failures := r.eval(env, stmt.Value, locals)
	if len(failures) > 0 {
		return failures
	}
	locals[stmt.Name] = value
	return nil
}

func (r *runner) execForall(env resolve.FileEnv, stmt *ast.Stmt, locals map[string]structure.EvalValue) []Failure {
	items, failures := r.binderItems(env, stmt, locals)
	if len(failures) > 0 {
		return failures
	}
	for _, item := range items {
		scoped := cloneBindings(locals)
		scoped[stmt.Binder.Name] = item
		include, whereFailures := r.binderWhere(env, stmt, scoped)
		failures = append(failures, whereFailures...)
		if !include {
			continue
		}
		failures = append(failures, r.execStatements(env, stmt.Body, scoped)...)
	}
	return failures
}

func (r *runner) execExists(env resolve.FileEnv, stmt *ast.Stmt, locals map[string]structure.EvalValue) []Failure {
	items, failures := r.binderItems(env, stmt, locals)
	if len(failures) > 0 {
		return failures
	}
	var firstMatch map[string]structure.EvalValue
	var firstBodyFailures []Failure
	for _, item := range items {
		scoped := cloneBindings(locals)
		scoped[stmt.Binder.Name] = item
		include, whereFailures := r.binderWhere(env, stmt, scoped)
		failures = append(failures, whereFailures...)
		if !include {
			continue
		}
		if firstMatch == nil {
			firstMatch = cloneBindings(scoped)
		}
		if len(stmt.Body) == 0 {
			return failures
		}
		bodyFailures := r.execStatements(env, stmt.Body, scoped)
		if len(bodyFailures) == 0 {
			return failures
		}
		if firstBodyFailures == nil {
			firstBodyFailures = bodyFailures
		}
	}
	if len(failures) > 0 {
		return failures
	}
	bindings := locals
	message := "no matching value satisfies exists"
	if stmt.Binder != nil {
		message = fmt.Sprintf("no %s %s satisfies exists", stmt.Binder.Domain, stmt.Binder.Name)
	}
	if firstMatch != nil {
		bindings = firstMatch
		message = "no matching value satisfies exists block"
		if stmt.Binder != nil {
			message = fmt.Sprintf("no %s %s satisfies exists block", stmt.Binder.Domain, stmt.Binder.Name)
		}
	}
	existsFailure := r.failure(stmt.Span, "exists", existsText(stmt), message, bindings)
	if len(firstBodyFailures) == 0 {
		return []Failure{existsFailure}
	}
	return append([]Failure{existsFailure}, firstBodyFailures...)
}

func (r *runner) binderItems(env resolve.FileEnv, stmt *ast.Stmt, locals map[string]structure.EvalValue) ([]structure.EvalValue, []Failure) {
	if stmt.Binder == nil {
		return nil, []Failure{r.failure(stmt.Span, string(stmt.Kind), "", "missing binder", locals)}
	}
	items, diagnostics := structure.EvalBinder(r.index, env, stmt.Binder, locals)
	if len(diagnostics) == 0 {
		return items, nil
	}
	r.diagnostics = append(r.diagnostics, diagnostics...)
	return nil, r.failuresFromDiagnostics(diagnostics, stmt.Span, binderText(stmt.Binder), locals)
}

func (r *runner) binderWhere(env resolve.FileEnv, stmt *ast.Stmt, locals map[string]structure.EvalValue) (bool, []Failure) {
	if stmt.Binder == nil || stmt.Binder.Where == nil {
		return true, nil
	}
	value, failures := r.eval(env, stmt.Binder.Where, locals)
	if len(failures) > 0 {
		return false, failures
	}
	return structure.EvalTruth(value), nil
}

func (r *runner) eval(env resolve.FileEnv, expr *ast.Expr, locals map[string]structure.EvalValue) (structure.EvalValue, []Failure) {
	result, diagnostics := structure.EvalQueryWithLocals(r.index, env, expr, locals)
	if len(diagnostics) == 0 {
		return result.Value, nil
	}
	r.diagnostics = append(r.diagnostics, diagnostics...)
	return result.Value, r.failuresFromDiagnostics(diagnostics, exprSpan(expr), exprText(expr), locals)
}

func (r *runner) failuresFromDiagnostics(diagnostics []source.Diagnostic, span source.Span, expression string, locals map[string]structure.EvalValue) []Failure {
	failures := make([]Failure, 0, len(diagnostics))
	for _, diagnostic := range diagnostics {
		diagSpan := span
		if diagnostic.File != "" {
			diagSpan.Start.File = diagnostic.File
		}
		if diagnostic.Line > 0 {
			diagSpan.Start.Line = diagnostic.Line
		}
		if diagnostic.Column > 0 {
			diagSpan.Start.Column = diagnostic.Column
		}
		failures = append(failures, r.failure(diagSpan, "eval", expression, diagnostic.Message, locals))
	}
	return failures
}

func (r *runner) messageText(env resolve.FileEnv, expr *ast.Expr, locals map[string]structure.EvalValue) (string, bool) {
	value, failures := r.eval(env, expr, locals)
	if len(failures) > 0 {
		return "", false
	}
	if value.Kind == structure.EvalString || value.Kind == structure.EvalSymbol {
		return value.Text, true
	}
	return value.String(), true
}

func (r *runner) failure(span source.Span, statement, expression, message string, locals map[string]structure.EvalValue) Failure {
	return Failure{
		File:       relFile(r.root, span.Start.File),
		Line:       span.Start.Line,
		Column:     span.Start.Column,
		Statement:  statement,
		Expression: expression,
		Message:    message,
		Bindings:   captureBindings(locals),
	}
}

func captureBindings(locals map[string]structure.EvalValue) map[string]BindingValue {
	if len(locals) == 0 {
		return nil
	}
	names := make([]string, 0, len(locals))
	for name := range locals {
		names = append(names, name)
	}
	sort.Strings(names)
	bindings := map[string]BindingValue{}
	for _, name := range names {
		value := locals[name]
		bindings[name] = BindingValue{Text: value.String(), Value: value}
	}
	return bindings
}

func cloneBindings(locals map[string]structure.EvalValue) map[string]structure.EvalValue {
	out := map[string]structure.EvalValue{}
	for name, value := range locals {
		out[name] = value
	}
	return out
}

func existsText(stmt *ast.Stmt) string {
	if stmt == nil || stmt.Binder == nil {
		return "exists"
	}
	return "exists " + binderText(stmt.Binder)
}

func exprSpan(expr *ast.Expr) source.Span {
	if expr == nil {
		return source.Span{}
	}
	return expr.Span
}
