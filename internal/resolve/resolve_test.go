package resolve

import (
	"path/filepath"
	"testing"

	"narr/internal/ast"
	"narr/internal/parser"
	"narr/internal/project"
	"narr/internal/source"
)

func TestBuildExamplesResolvesCrossNamespaceReferences(t *testing.T) {
	root := filepath.Join("..", "..", "examples", "红楼梦")
	loaded, diagnostics := project.Load(project.LoadOptions{ProjectDir: root})
	if source.HasErrors(diagnostics) {
		t.Fatalf("project.Load diagnostics: %#v", diagnostics)
	}
	files, diagnostics := parser.ParseProject(loaded)
	if source.HasErrors(diagnostics) {
		t.Fatalf("parser diagnostics: %#v", diagnostics)
	}

	resolved, diagnostics := Build(loaded, files)
	if source.HasErrors(diagnostics) {
		t.Fatalf("resolve diagnostics:\n%s", diagnosticsText(diagnostics))
	}
	if resolved.Namespaces["红楼梦.structure"] == nil {
		t.Fatalf("missing merged structure namespace")
	}
	if resolved.Namespaces["红楼梦.structure"].Symbols["vol01.ch01"] == nil {
		t.Fatalf("missing canonical chapter symbol vol01.ch01")
	}
}

func TestBuildReportsDuplicateDeclaration(t *testing.T) {
	loaded := testProject(t)
	first := parseNarr(t, "a.narr", "namespace x\nfact 重复 = \"one\"\n")
	second := parseNarr(t, "b.narr", "namespace x\nfact 重复 = \"two\"\n")

	_, diagnostics := Build(loaded, []*ast.File{first, second})
	if !source.HasErrors(diagnostics) {
		t.Fatalf("Build returned no diagnostics for duplicate declaration")
	}
	if diagnostics[0].Code != "E0301" {
		t.Fatalf("first diagnostic code = %s, want E0301", diagnostics[0].Code)
	}
}

func TestBuildReportsDuplicateCanonicalChapterCode(t *testing.T) {
	loaded := testProject(t)
	first := parseNarr(t, "a.narr", "namespace x\nchapter vol1.ch001\n")
	second := parseNarr(t, "b.narr", "namespace x\nchapter vol01.ch01\n")

	_, diagnostics := Build(loaded, []*ast.File{first, second})
	if !source.HasErrors(diagnostics) {
		t.Fatalf("Build returned no diagnostics for duplicate chapter code")
	}
	if diagnostics[0].Code != "E0302" {
		t.Fatalf("first diagnostic code = %s, want E0302", diagnostics[0].Code)
	}
}

func TestBuildReportsUnknownReference(t *testing.T) {
	loaded := testProject(t)
	file := parseNarr(t, "a.narr", `namespace x
chapter vol01.ch01 {
  start_pattern: 不存在
}
`)
	_, diagnostics := Build(loaded, []*ast.File{file})
	if !source.HasErrors(diagnostics) {
		t.Fatalf("Build returned no diagnostics for unknown reference")
	}
	if diagnostics[0].Code != "E0311" {
		t.Fatalf("first diagnostic code = %s, want E0311", diagnostics[0].Code)
	}
}

func TestBuildAllowsBuiltinStoryAnchors(t *testing.T) {
	loaded := testProject(t)
	file := parseNarr(t, "a.narr", `namespace x
thread 主线 {
  starts_at: beginning
  expected_resolution: end_of_story
}
`)
	_, diagnostics := Build(loaded, []*ast.File{file})
	if source.HasErrors(diagnostics) {
		t.Fatalf("Build diagnostics:\n%s", diagnosticsText(diagnostics))
	}
}

func TestTestLocalVariablesTakePrecedence(t *testing.T) {
	loaded := testProject(t)
	structure := parseNarr(t, "structure.narr", `namespace x.structure
volume vol01
chapter vol01.ch01
thread 主线 {
  starts_at: vol01.ch01
}
`)
	testFile := parseTest(t, "tests.test.narr", `namespace x.tests
import x.structure as s

test "locals" {
  let 当前 = s.vol01.ch01
  assert 当前 starts s.主线
}
`)

	_, diagnostics := Build(loaded, []*ast.File{structure, testFile})
	if source.HasErrors(diagnostics) {
		t.Fatalf("Build diagnostics:\n%s", diagnosticsText(diagnostics))
	}
}

func TestQueryEnvironmentUsesMainImports(t *testing.T) {
	root := filepath.Join("..", "..", "examples", "红楼梦")
	loaded, diagnostics := project.Load(project.LoadOptions{ProjectDir: root})
	if source.HasErrors(diagnostics) {
		t.Fatalf("project.Load diagnostics: %#v", diagnostics)
	}
	files, diagnostics := parser.ParseProject(loaded)
	if source.HasErrors(diagnostics) {
		t.Fatalf("parser diagnostics: %#v", diagnostics)
	}
	resolved, diagnostics := Build(loaded, files)
	if source.HasErrors(diagnostics) {
		t.Fatalf("resolve diagnostics:\n%s", diagnosticsText(diagnostics))
	}
	env, diagnostics := resolved.QueryEnv(loaded, "")
	if source.HasErrors(diagnostics) {
		t.Fatalf("QueryEnv diagnostics: %#v", diagnostics)
	}
	if env.Namespace != "红楼梦" {
		t.Fatalf("query namespace = %q, want 红楼梦", env.Namespace)
	}
	if env.Imports["structure"] != "红楼梦.structure" {
		t.Fatalf("structure import = %q, want 红楼梦.structure", env.Imports["structure"])
	}
	expr, diagnostics := parser.ParseExpression("<query>", "active_threads(structure.vol01.ch01)")
	if source.HasErrors(diagnostics) {
		t.Fatalf("query parse diagnostics: %#v", diagnostics)
	}
	diagnostics = ResolveQueryExpr(resolved, env, expr)
	if source.HasErrors(diagnostics) {
		t.Fatalf("query resolve diagnostics:\n%s", diagnosticsText(diagnostics))
	}
}

func parseNarr(t *testing.T, path, text string) *ast.File {
	t.Helper()
	file, diagnostics := parser.ParseNarrFile(path, text)
	if source.HasErrors(diagnostics) {
		t.Fatalf("ParseNarrFile(%s) diagnostics: %#v", path, diagnostics)
	}
	return file
}

func parseTest(t *testing.T, path, text string) *ast.File {
	t.Helper()
	file, diagnostics := parser.ParseTestFile(path, text)
	if source.HasErrors(diagnostics) {
		t.Fatalf("ParseTestFile(%s) diagnostics: %#v", path, diagnostics)
	}
	return file
}

func testProject(t *testing.T) *project.Project {
	t.Helper()
	return &project.Project{
		Root: t.TempDir(),
		Config: project.Config{
			Name:    "x",
			Version: project.SupportedProjectVersion,
			Main:    "main.narr",
		},
	}
}

func diagnosticsText(diagnostics []source.Diagnostic) string {
	out := ""
	for _, diagnostic := range diagnostics {
		out += diagnostic.Code + " " + diagnostic.File + " " + diagnostic.Message + "\n"
	}
	return out
}
