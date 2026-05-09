package state

import (
	"path/filepath"
	"testing"

	"narr/internal/ast"
	"narr/internal/check"
	"narr/internal/parser"
	"narr/internal/project"
	"narr/internal/resolve"
	"narr/internal/source"
)

func TestTimelineExamplesChapterEndState(t *testing.T) {
	loaded, _, resolved, checked := loadCheckedProject(t, filepath.Join("..", "..", "examples", "红楼梦"))
	timeline, diagnostics := Build(checked.Model, resolved)
	if source.HasErrors(diagnostics) {
		t.Fatalf("Build diagnostics:\n%s", diagnosticsText(diagnostics))
	}
	env, diagnostics := resolved.QueryEnv(loaded, "")
	if source.HasErrors(diagnostics) {
		t.Fatalf("QueryEnv diagnostics:\n%s", diagnosticsText(diagnostics))
	}

	assertStateExpr(t, timeline, env, "state(chars.甄士隐.状态, at structure.vol01.ch01.end)", Symbol("出家"))
	assertStateExpr(t, timeline, env, "state(chars.英莲.状态, at structure.vol01.ch01.end)", Symbol("失踪"))
	assertStateExpr(t, timeline, env, "state(chars.贾雨村.状态, at structure.vol01.ch02.end)", Symbol("将有转机"))
	assertStateExpr(t, timeline, env, "state(chars.林黛玉.位置, at structure.vol01.ch02.end)", Ref("红楼梦.world.林府"))

	if len(timeline.OrderedCodes) != 2 {
		t.Fatalf("len(OrderedCodes) = %d, want 2", len(timeline.OrderedCodes))
	}
	if len(timeline.OrderedBeats) != 27 {
		t.Fatalf("len(OrderedBeats) = %d, want 27", len(timeline.OrderedBeats))
	}
}

func TestTimelineAppliesAssignmentSetRemoveAndListAppend(t *testing.T) {
	loaded := testProject(t)
	files := []*ast.File{
		parseNarr(t, "world.narr", `namespace x.world
place 未明
place 王都
fact 旧事实 = "旧"
fact 新事实 = "新"
`),
		parseNarr(t, "characters.narr", `namespace x.characters
import x.world as world

class 叙事人物 {
  field 位置: Place = world.未明
  field 知道: Set<Fact> = { world.旧事实 }
  field 记录: List<Claim> = []
}

character 沈夜 : 叙事人物
`),
		parseNarr(t, "structure.narr", `namespace x.structure
import x.characters as chars
import x.world as world

chapter vol01.ch01 {
  beats: [入城, 忘记]
}

beat 入城 @ vol01.ch01 {
  effect:
    chars.沈夜.位置 = world.王都
    chars.沈夜.知道 += world.新事实
    chars.沈夜.记录 append "入城"
}

beat 忘记 @ vol01.ch01 {
  effect:
    chars.沈夜.知道 -= world.旧事实
}
`),
	}
	_, resolved, checked := resolveAndCheck(t, loaded, files)
	timeline, diagnostics := Build(checked.Model, resolved)
	if source.HasErrors(diagnostics) {
		t.Fatalf("Build diagnostics:\n%s", diagnosticsText(diagnostics))
	}
	env := resolve.FileEnv{Namespace: "x.structure", Imports: map[string]string{"chars": "x.characters", "world": "x.world"}}

	assertStateExpr(t, timeline, env, "state(chars.沈夜.位置, at vol01.ch01.end)", Ref("x.world.王都"))
	assertStateExpr(t, timeline, env, "state(chars.沈夜.知道, at vol01.ch01.end)", Set([]Value{Ref("x.world.新事实")}))
	assertStateExpr(t, timeline, env, "state(chars.沈夜.记录, at vol01.ch01.end)", List([]Value{String("入城")}))

	if timeline.ChapterBegin["vol01.ch01"].Len() == 0 || timeline.ChapterEnd["vol01.ch01"].Len() == 0 {
		t.Fatalf("chapter boundary checkpoints were not saved")
	}
	if len(timeline.BeatBefore) != 2 || len(timeline.BeatAfter) != 2 {
		t.Fatalf("beat checkpoints before/after = %d/%d, want 2/2", len(timeline.BeatBefore), len(timeline.BeatAfter))
	}
}

func TestTimelineAllowsEmptyVolumeAnchors(t *testing.T) {
	loaded := testProject(t)
	file := parseNarr(t, "structure.narr", `namespace x.structure
volume vol01
`)
	_, resolved, checked := resolveAndCheck(t, loaded, []*ast.File{file})
	timeline, diagnostics := Build(checked.Model, resolved)
	if source.HasErrors(diagnostics) {
		t.Fatalf("Build diagnostics:\n%s", diagnosticsText(diagnostics))
	}
	env := resolve.FileEnv{Namespace: "x.structure"}

	begin, diagnostics := timeline.AnchorFromExpr(env, &ast.Expr{Kind: ast.ExprRef, Value: "vol01"})
	if source.HasErrors(diagnostics) {
		t.Fatalf("AnchorFromExpr(vol01) diagnostics:\n%s", diagnosticsText(diagnostics))
	}
	if begin.Kind != AnchorBeginning {
		t.Fatalf("vol01 anchor = %s, want %s", begin.Kind, AnchorBeginning)
	}

	end, diagnostics := timeline.AnchorFromExpr(env, &ast.Expr{Kind: ast.ExprPath, Value: "vol01.end"})
	if source.HasErrors(diagnostics) {
		t.Fatalf("AnchorFromExpr(vol01.end) diagnostics:\n%s", diagnosticsText(diagnostics))
	}
	if end.Kind != AnchorEndOfStory {
		t.Fatalf("vol01.end anchor = %s, want %s", end.Kind, AnchorEndOfStory)
	}
}

func assertStateExpr(t *testing.T, timeline *Timeline, env resolve.FileEnv, text string, want Value) {
	t.Helper()
	expr, diagnostics := parser.ParseExpression("<test>", text)
	if source.HasErrors(diagnostics) {
		t.Fatalf("ParseExpression(%q) diagnostics:\n%s", text, diagnosticsText(diagnostics))
	}
	got, diagnostics := timeline.StateExpr(env, expr)
	if source.HasErrors(diagnostics) {
		t.Fatalf("StateExpr(%q) diagnostics:\n%s", text, diagnosticsText(diagnostics))
	}
	if got.StableKey() != want.StableKey() {
		t.Fatalf("StateExpr(%q) = %s (%s), want %s (%s)", text, got.String(), got.StableKey(), want.String(), want.StableKey())
	}
}

func loadCheckedProject(t *testing.T, root string) (*project.Project, []*ast.File, *resolve.Project, *check.Result) {
	t.Helper()
	loaded, diagnostics := project.Load(project.LoadOptions{ProjectDir: root})
	if source.HasErrors(diagnostics) {
		t.Fatalf("project.Load diagnostics:\n%s", diagnosticsText(diagnostics))
	}
	files, diagnostics := parser.ParseProject(loaded)
	if source.HasErrors(diagnostics) {
		t.Fatalf("ParseProject diagnostics:\n%s", diagnosticsText(diagnostics))
	}
	_, resolved, checked := resolveAndCheck(t, loaded, files)
	return loaded, files, resolved, checked
}

func resolveAndCheck(t *testing.T, loaded *project.Project, files []*ast.File) ([]*ast.File, *resolve.Project, *check.Result) {
	t.Helper()
	resolved, diagnostics := resolve.Build(loaded, files)
	if source.HasErrors(diagnostics) {
		t.Fatalf("resolve diagnostics:\n%s", diagnosticsText(diagnostics))
	}
	checked, diagnostics := check.Check(loaded, files, resolved)
	if source.HasErrors(diagnostics) {
		t.Fatalf("check diagnostics:\n%s", diagnosticsText(diagnostics))
	}
	return files, resolved, checked
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
