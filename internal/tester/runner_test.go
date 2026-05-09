package tester

import (
	"path/filepath"
	"strings"
	"testing"

	"narr/internal/ast"
	"narr/internal/check"
	"narr/internal/parser"
	"narr/internal/project"
	"narr/internal/resolve"
	"narr/internal/source"
	statetimeline "narr/internal/state"
	"narr/internal/structure"
)

func TestRunExamplesProject(t *testing.T) {
	loaded, files, resolved, index := loadTestProject(t)

	report := Run(Options{Root: loaded.Root, Files: testFiles(files), Resolved: resolved, Index: index})

	if !report.OK {
		t.Fatalf("report failed: %#v", report)
	}
	if report.Total != 13 {
		t.Fatalf("report.Total = %d, want 13", report.Total)
	}
}

func TestFailureIncludesExpressionAndBindings(t *testing.T) {
	loaded, files, _, _ := loadTestProject(t)
	failingFile, diagnostics := parser.ParseTestFile(filepath.Join(loaded.Root, "tests", "m9_failure.test.narr"), `namespace 红楼梦.tests

import 红楼梦.structure as structure

test "binding failure" {
  forall chapter ch in chapters_in(structure.vol01):
    assert ch.alias missing
}
`)
	if source.HasErrors(diagnostics) {
		t.Fatalf("ParseTestFile diagnostics: %#v", diagnostics)
	}
	files = append(files, failingFile)
	resolved, index := buildTestIndex(t, loaded, files)

	report := Run(Options{Root: loaded.Root, Files: []*ast.File{failingFile}, Resolved: resolved, Index: index})

	if report.OK {
		t.Fatalf("report unexpectedly passed")
	}
	if report.Total != 1 || report.Failed != 1 {
		t.Fatalf("report counts = total %d failed %d, want 1 failed", report.Total, report.Failed)
	}
	if len(report.Tests) != 1 || len(report.Tests[0].Failures) == 0 {
		t.Fatalf("missing test failure: %#v", report.Tests)
	}
	failure := report.Tests[0].Failures[0]
	if !strings.Contains(failure.Expression, "ch.alias missing") {
		t.Fatalf("failure expression = %q, want ch.alias missing", failure.Expression)
	}
	if failure.File == "" || failure.Line == 0 || failure.Column == 0 {
		t.Fatalf("failure span missing: %#v", failure)
	}
	if _, ok := failure.Bindings["ch"]; !ok {
		t.Fatalf("failure bindings = %#v, want ch", failure.Bindings)
	}
}

func TestRunRejectsNonBoolAssert(t *testing.T) {
	loaded, _, resolved, index := loadTestProject(t)
	failingFile, diagnostics := parser.ParseTestFile(filepath.Join(loaded.Root, "tests", "non_bool_assert.test.narr"), `namespace 红楼梦.tests

test "non bool assert" {
  assert "text"
}
`)
	if source.HasErrors(diagnostics) {
		t.Fatalf("ParseTestFile diagnostics: %#v", diagnostics)
	}

	report := Run(Options{Root: loaded.Root, Files: []*ast.File{failingFile}, Resolved: resolved, Index: index})

	if report.OK {
		t.Fatalf("report unexpectedly passed")
	}
	if len(report.Tests) != 1 || len(report.Tests[0].Failures) != 1 {
		t.Fatalf("failures = %#v, want one failure", report.Tests)
	}
	if !strings.Contains(report.Tests[0].Failures[0].Message, "Bool") {
		t.Fatalf("failure message = %q, want Bool", report.Tests[0].Failures[0].Message)
	}
}

func loadTestProject(t *testing.T) (*project.Project, []*ast.File, *resolve.Project, *structure.Index) {
	t.Helper()
	root := filepath.Join("..", "..", "examples", "红楼梦")
	loaded, diagnostics := project.Load(project.LoadOptions{ProjectDir: root})
	if source.HasErrors(diagnostics) {
		t.Fatalf("project.Load diagnostics: %#v", diagnostics)
	}
	files, diagnostics := parser.ParseProject(loaded)
	if source.HasErrors(diagnostics) {
		t.Fatalf("ParseProject diagnostics: %#v", diagnostics)
	}
	resolved, index := buildTestIndex(t, loaded, files)
	return loaded, files, resolved, index
}

func buildTestIndex(t *testing.T, loaded *project.Project, files []*ast.File) (*resolve.Project, *structure.Index) {
	t.Helper()
	resolved, diagnostics := resolve.Build(loaded, files)
	if source.HasErrors(diagnostics) {
		t.Fatalf("resolve.Build diagnostics:\n%s", diagnosticsText(diagnostics))
	}
	checked, diagnostics := check.Check(loaded, files, resolved)
	if source.HasErrors(diagnostics) {
		t.Fatalf("check.Check diagnostics:\n%s", diagnosticsText(diagnostics))
	}
	timeline, diagnostics := statetimeline.Build(checked.Model, resolved)
	if source.HasErrors(diagnostics) {
		t.Fatalf("state.Build diagnostics:\n%s", diagnosticsText(diagnostics))
	}
	index, diagnostics := structure.Build(checked.Model, resolved, timeline)
	if source.HasErrors(diagnostics) {
		t.Fatalf("structure.Build diagnostics:\n%s", diagnosticsText(diagnostics))
	}
	return resolved, index
}

func testFiles(files []*ast.File) []*ast.File {
	var out []*ast.File
	for _, file := range files {
		if file.Mode == ast.ModeTest {
			out = append(out, file)
		}
	}
	return out
}

func diagnosticsText(diagnostics []source.Diagnostic) string {
	var builder strings.Builder
	for _, diagnostic := range diagnostics {
		builder.WriteString(diagnostic.Code)
		builder.WriteByte(' ')
		builder.WriteString(diagnostic.File)
		builder.WriteByte(' ')
		builder.WriteString(diagnostic.Message)
		builder.WriteByte('\n')
	}
	return builder.String()
}
