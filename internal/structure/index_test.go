package structure

import (
	"path/filepath"
	"testing"

	"narr/internal/ast"
	"narr/internal/check"
	"narr/internal/model"
	"narr/internal/parser"
	"narr/internal/project"
	"narr/internal/resolve"
	"narr/internal/source"
	statetimeline "narr/internal/state"
)

func TestBuildExamplesDerivedViews(t *testing.T) {
	index := loadStructureIndex(t, filepath.Join("..", "..", "examples", "红楼梦"))

	ch01 := index.View("vol01.ch01")
	if ch01 == nil {
		t.Fatalf("missing vol01.ch01 view")
	}
	assertContainsID(t, ch01.ActiveThreads, "红楼梦.threads.石头入世线")
	assertContainsID(t, ch01.ActiveThreads, "红楼梦.threads.甄士隐兴亡线")
	assertContainsID(t, ch01.ActivePromises, "红楼梦.promises.公差传唤悬念")
	assertContainsID(t, ch01.ServedPromises, "红楼梦.promises.通灵宝玉来历伏笔")
	assertContainsID(t, ch01.ServedPromises, "红楼梦.promises.英莲失踪伏笔")
	assertContainsID(t, ch01.ServedArcs, "红楼梦.arcs.甄士隐悟道弧")
	assertContainsID(t, ch01.Reveals, "红楼梦.world.通灵宝玉源起")
	assertContainsID(t, ch01.Mentions, "红楼梦.world.青埂峰")

	ch02 := index.View("vol01.ch02")
	if ch02 == nil {
		t.Fatalf("missing vol01.ch02 view")
	}
	assertContainsID(t, ch02.ActiveThreads, "红楼梦.threads.贾府引入线")
	assertContainsID(t, ch02.ServedPromises, "红楼梦.promises.荣国府衰象伏笔")
	assertContainsID(t, ch02.ServedPromises, "红楼梦.promises.宝玉衔玉伏笔")
}

func TestBuildReportsFailedBeatPrecondition(t *testing.T) {
	files := []*ast.File{
		parseNarr(t, "main.narr", `namespace x
class 人 {
  field 状态: Symbol = 安居
}
character 主 : 人
chapter vol01.ch01 {
  beats: [出门]
}
beat 出门 @ vol01.ch01 {
  precondition:
    主.状态 == 离家
  effect:
    主.状态 = 离家
}
`),
	}
	diagnostics := buildStructureDiagnostics(t, testProject(t), files)
	assertDiagnosticCode(t, diagnostics, "E0623")
}

func TestBuildReportsPromisePayoffBeforeSetup(t *testing.T) {
	files := []*ast.File{
		parseNarr(t, "main.narr", `namespace x
chapter vol01.ch01
chapter vol01.ch02
promise 伏笔 {
  setup_at: vol01.ch02
  payoff_at: vol01.ch01
}
`),
	}
	diagnostics := buildStructureDiagnostics(t, testProject(t), files)
	assertDiagnosticCode(t, diagnostics, "E0612")
}

func TestBuildReportsArcStateOutsideStates(t *testing.T) {
	files := []*ast.File{
		parseNarr(t, "main.narr", `namespace x
class 人 {
  field arc_state: Symbol = 初
}
character 主 : 人
chapter vol01.ch01 {
  beats: [变化]
}
beat 变化 @ vol01.ch01 {
  effect:
    主.arc_state = 未列出
}
arc 主弧 {
  subject: 主
  starts_at: vol01.ch01
  state_field: arc_state
  initial: 初
  states: [初, 终]
}
`),
	}
	diagnostics := buildStructureDiagnostics(t, testProject(t), files)
	assertDiagnosticCode(t, diagnostics, "E0620")
}

func TestBuildReportsHiddenInvariantEarlyReveal(t *testing.T) {
	files := []*ast.File{
		parseNarr(t, "main.narr", `namespace x
fact 秘密 = "secret"
chapter vol01.ch01 {
  beats: [揭示]
}
chapter vol01.ch02
beat 揭示 @ vol01.ch01 {
  reveals: 秘密
}
invariant 保密 {
  hidden: 秘密 until vol01.ch02
}
`),
	}
	diagnostics := buildStructureDiagnostics(t, testProject(t), files)
	assertDiagnosticCode(t, diagnostics, "E0621")
}

func TestBuildReportsAlwaysInvariantFailure(t *testing.T) {
	files := []*ast.File{
		parseNarr(t, "main.narr", `namespace x
class 人 {
  field 存活: Bool = true
}
character 主 : 人
chapter vol01.ch01 {
  beats: [遇险]
}
chapter vol01.ch02
beat 遇险 @ vol01.ch01 {
  effect:
    主.存活 = false
}
invariant 生存 {
  always:
    主.存活 == true
  active_until: vol01.ch02
}
`),
	}
	diagnostics := buildStructureDiagnostics(t, testProject(t), files)
	assertDiagnosticCode(t, diagnostics, "E0623")
}

func loadStructureIndex(t *testing.T, root string) *Index {
	t.Helper()
	loaded, diagnostics := project.Load(project.LoadOptions{ProjectDir: root})
	if source.HasErrors(diagnostics) {
		t.Fatalf("project.Load diagnostics:\n%s", diagnosticsText(diagnostics))
	}
	files, diagnostics := parser.ParseProject(loaded)
	if source.HasErrors(diagnostics) {
		t.Fatalf("ParseProject diagnostics:\n%s", diagnosticsText(diagnostics))
	}
	resolved, diagnostics := resolve.Build(loaded, files)
	if source.HasErrors(diagnostics) {
		t.Fatalf("resolve diagnostics:\n%s", diagnosticsText(diagnostics))
	}
	checked, diagnostics := check.Check(loaded, files, resolved)
	if source.HasErrors(diagnostics) {
		t.Fatalf("check diagnostics:\n%s", diagnosticsText(diagnostics))
	}
	timeline, diagnostics := statetimeline.Build(checked.Model, resolved)
	if source.HasErrors(diagnostics) {
		t.Fatalf("timeline diagnostics:\n%s", diagnosticsText(diagnostics))
	}
	index, diagnostics := Build(checked.Model, resolved, timeline)
	if source.HasErrors(diagnostics) {
		t.Fatalf("structure diagnostics:\n%s", diagnosticsText(diagnostics))
	}
	return index
}

func buildStructureDiagnostics(t *testing.T, loaded *project.Project, files []*ast.File) []source.Diagnostic {
	t.Helper()
	resolved, diagnostics := resolve.Build(loaded, files)
	if source.HasErrors(diagnostics) {
		return diagnostics
	}
	checked, diagnostics := check.Check(loaded, files, resolved)
	if source.HasErrors(diagnostics) {
		return diagnostics
	}
	timeline, diagnostics := statetimeline.Build(checked.Model, resolved)
	if source.HasErrors(diagnostics) {
		return diagnostics
	}
	_, diagnostics = Build(checked.Model, resolved, timeline)
	return diagnostics
}

func assertContainsID(t *testing.T, got []model.SymbolID, want model.SymbolID) {
	t.Helper()
	for _, id := range got {
		if id == want {
			return
		}
	}
	t.Fatalf("%v does not contain %s", got, want)
}

func assertDiagnosticCode(t *testing.T, diagnostics []source.Diagnostic, code string) {
	t.Helper()
	for _, diagnostic := range diagnostics {
		if diagnostic.Code == code {
			return
		}
	}
	t.Fatalf("diagnostics do not contain %s:\n%s", code, diagnosticsText(diagnostics))
}

func parseNarr(t *testing.T, path, text string) *ast.File {
	t.Helper()
	file, diagnostics := parser.ParseNarrFile(path, text)
	if source.HasErrors(diagnostics) {
		t.Fatalf("ParseNarrFile(%s) diagnostics:\n%s", path, diagnosticsText(diagnostics))
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
