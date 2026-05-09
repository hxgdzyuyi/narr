package check

import (
	"path/filepath"
	"testing"

	"narr/internal/ast"
	"narr/internal/parser"
	"narr/internal/project"
	"narr/internal/resolve"
	"narr/internal/source"
)

func TestCheckExamplesProject(t *testing.T) {
	loaded, files, resolved := loadParsedResolvedProject(t, filepath.Join("..", "..", "examples", "红楼梦"))
	_, diagnostics := Check(loaded, files, resolved)
	if source.HasErrors(diagnostics) {
		t.Fatalf("Check diagnostics:\n%s", diagnosticsText(diagnostics))
	}
}

func TestCheckEffectUnknownFieldFails(t *testing.T) {
	loaded := testProject(t)
	characters := parseNarr(t, "characters.narr", `namespace x.characters
import x.world as world

class 叙事人物 {
  field 位置: Place
}
character 沈夜 : 叙事人物 {
  位置 = world.未明
}
`)
	world := parseNarr(t, "world.narr", `namespace x.world
place 未明
place 王都
`)
	structure := parseNarr(t, "structure.narr", `namespace x.structure
import x.characters as chars
import x.world as world

chapter vol01.ch01
beat 入城 @ vol01.ch01 {
  effect:
    chars.沈夜.不存在 = world.王都
}
`)

	diagnostics := parseResolveCheck(t, loaded, []*ast.File{characters, world, structure})
	if !source.HasErrors(diagnostics) {
		t.Fatalf("Check returned no error for unknown effect field")
	}
	if diagnostics[0].Code != "E0404" {
		t.Fatalf("first diagnostic code = %s, want E0404", diagnostics[0].Code)
	}
}

func TestCheckEffectValueTypeMismatchFails(t *testing.T) {
	loaded := testProject(t)
	characters := parseNarr(t, "characters.narr", `namespace x.characters
class 叙事人物 {
  field 位置: Place
}
character 沈夜 : 叙事人物
`)
	structure := parseNarr(t, "structure.narr", `namespace x.structure
import x.characters as chars

chapter vol01.ch01
beat 入城 @ vol01.ch01 {
  effect:
    chars.沈夜.位置 = true
}
`)

	diagnostics := parseResolveCheck(t, loaded, []*ast.File{characters, structure})
	if !source.HasErrors(diagnostics) {
		t.Fatalf("Check returned no error for type mismatch")
	}
	if diagnostics[0].Code != "E0409" {
		t.Fatalf("first diagnostic code = %s, want E0409", diagnostics[0].Code)
	}
}

func TestCheckEnumDefaultValueFailsForUnknownMember(t *testing.T) {
	loaded := testProject(t)
	file := parseNarr(t, "world.narr", `namespace x.world
enum 城市状态 {
  封闭
  开放
}
place 王都 {
  状态: 城市状态 = 起火
}
`)

	diagnostics := parseResolveCheck(t, loaded, []*ast.File{file})
	if !source.HasErrors(diagnostics) {
		t.Fatalf("Check returned no error for invalid enum member")
	}
	if diagnostics[0].Code != "E0411" {
		t.Fatalf("first diagnostic code = %s, want E0411", diagnostics[0].Code)
	}
}

func TestCheckSchemaRejectsUnknownField(t *testing.T) {
	loaded := testProject(t)
	file := parseNarr(t, "structure.narr", `namespace x.structure
chapter vol01.ch01 {
  unknown_field: true
}
`)

	diagnostics := parseResolveCheck(t, loaded, []*ast.File{file})
	assertDiagnosticCode(t, diagnostics, "E0415")
}

func TestCheckSchemaRejectsInvalidEnumValues(t *testing.T) {
	tests := []struct {
		name string
		text string
	}{
		{
			name: "chapter purpose",
			text: `namespace x.structure
chapter vol01.ch01 {
  purpose: nonsense
}
`,
		},
		{
			name: "promise strength",
			text: `namespace x.structure
promise 伏笔 {
  setup_strength: huge
}
`,
		},
		{
			name: "thread kind",
			text: `namespace x.structure
thread 主线 {
  kind: random
}
`,
		},
		{
			name: "volume purpose",
			text: `namespace x.structure
volume vol01 {
  purpose: random
}
`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			loaded := testProject(t)
			file := parseNarr(t, "structure.narr", tt.text)

			diagnostics := parseResolveCheck(t, loaded, []*ast.File{file})
			assertDiagnosticCode(t, diagnostics, "E0417")
		})
	}
}

func TestCheckTestSemanticsRejectsNonBoolAssert(t *testing.T) {
	loaded := testProject(t)
	file := parseTest(t, "bad.test.narr", `namespace x.tests
test "bad assert" {
  assert "text"
}
`)

	diagnostics := parseResolveCheck(t, loaded, []*ast.File{file})
	assertDiagnosticCode(t, diagnostics, "E0903")
}

func TestCheckTestSemanticsRejectsNonCollectionCount(t *testing.T) {
	loaded := testProject(t)
	structure := parseNarr(t, "structure.narr", `namespace x.structure
chapter vol01.ch01
`)
	testFile := parseTest(t, "bad.test.narr", `namespace x.tests
import x.structure as structure

test "bad count" {
  assert count(structure.vol01.ch01) == 1
}
`)

	diagnostics := parseResolveCheck(t, loaded, []*ast.File{structure, testFile})
	assertDiagnosticCode(t, diagnostics, "E0904")
}

func TestCheckTestSemanticsRejectsBinderCollectionMismatch(t *testing.T) {
	loaded := testProject(t)
	structure := parseNarr(t, "structure.narr", `namespace x.structure
thread 主线 {
  starts_at: vol01.ch01
}
chapter vol01.ch01
`)
	testFile := parseTest(t, "bad.test.narr", `namespace x.tests
import x.structure as structure

test "bad binder" {
  forall chapter ch in threads:
    assert ch exists
}
`)

	diagnostics := parseResolveCheck(t, loaded, []*ast.File{structure, testFile})
	assertDiagnosticCode(t, diagnostics, "E0904")
}

func TestCheckTestSemanticsRejectsDuplicateLocal(t *testing.T) {
	loaded := testProject(t)
	file := parseTest(t, "bad.test.narr", `namespace x.tests
test "duplicate local" {
  let 第一回 = 1
  let 第一回 = 2
}
`)

	diagnostics := parseResolveCheck(t, loaded, []*ast.File{file})
	assertDiagnosticCode(t, diagnostics, "E0901")
}

func TestCheckTestSemanticsRejectsNarrativePredicateTargetType(t *testing.T) {
	loaded := testProject(t)
	structure := parseNarr(t, "structure.narr", `namespace x.structure
chapter vol01.ch01
promise 伏笔 {
  setup_at: vol01.ch01
}
`)
	testFile := parseTest(t, "bad.test.narr", `namespace x.tests
import x.structure as structure

test "bad narrative predicate" {
  forall chapter ch:
    assert ch reveals structure.伏笔
}
`)

	diagnostics := parseResolveCheck(t, loaded, []*ast.File{structure, testFile})
	assertDiagnosticCode(t, diagnostics, "E0905")
}

func loadParsedResolvedProject(t *testing.T, root string) (*project.Project, []*ast.File, *resolve.Project) {
	t.Helper()
	loaded, diagnostics := project.Load(project.LoadOptions{ProjectDir: root})
	if source.HasErrors(diagnostics) {
		t.Fatalf("project.Load diagnostics: %#v", diagnostics)
	}
	files, diagnostics := parser.ParseProject(loaded)
	if source.HasErrors(diagnostics) {
		t.Fatalf("parser diagnostics: %#v", diagnostics)
	}
	resolved, diagnostics := resolve.Build(loaded, files)
	if source.HasErrors(diagnostics) {
		t.Fatalf("resolve diagnostics:\n%s", diagnosticsText(diagnostics))
	}
	return loaded, files, resolved
}

func parseResolveCheck(t *testing.T, loaded *project.Project, files []*ast.File) []source.Diagnostic {
	t.Helper()
	resolved, diagnostics := resolve.Build(loaded, files)
	if source.HasErrors(diagnostics) {
		return diagnostics
	}
	_, diagnostics = Check(loaded, files, resolved)
	return diagnostics
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

func assertDiagnosticCode(t *testing.T, diagnostics []source.Diagnostic, code string) {
	t.Helper()
	if !source.HasErrors(diagnostics) {
		t.Fatalf("Check returned no error, want %s", code)
	}
	for _, diagnostic := range diagnostics {
		if diagnostic.Code == code {
			return
		}
	}
	t.Fatalf("diagnostics missing %s:\n%s", code, diagnosticsText(diagnostics))
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
