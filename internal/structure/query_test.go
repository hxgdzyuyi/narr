package structure

import (
	"path/filepath"
	"testing"

	"narr/internal/check"
	"narr/internal/parser"
	"narr/internal/project"
	"narr/internal/resolve"
	"narr/internal/source"
	statetimeline "narr/internal/state"
)

func TestEvalQueryAcceptanceCountNarrativePredicate(t *testing.T) {
	_, resolved, index, env := loadExamplesQuery(t)
	result := evalExampleQuery(t, resolved, index, env, `count(chapter 章节 in chapters_in(structure.vol01) where 章节 serves promises.通灵宝玉来历伏笔)`)
	if result.Value.Kind != EvalInt || result.Value.Int != 1 {
		t.Fatalf("query result = %s, want 1", result.Value.String())
	}
}

func TestEvalQueryCollectionsCollectStateAndBuiltins(t *testing.T) {
	_, resolved, index, env := loadExamplesQuery(t)
	tests := []struct {
		expr string
		want string
	}{
		{`collect(章节.code from chapter 章节 in chapters_in(structure.vol01) where 章节 serves promises.通灵宝玉来历伏笔)`, `[vol01.ch01]`},
		{`state(chars.甄士隐.状态, at structure.vol01.ch01.end)`, `出家`},
		{`count(served_promises(structure.vol01.ch01))`, `6`},
		{`chapter_distance(promises.通灵宝玉来历伏笔.setup_at, promises.通灵宝玉来历伏笔.payoff_by)`, `0`},
	}
	for _, tt := range tests {
		result := evalExampleQuery(t, resolved, index, env, tt.expr)
		if got := result.Value.String(); got != tt.want {
			t.Fatalf("EvalQuery(%q) = %s, want %s", tt.expr, got, tt.want)
		}
	}
}

func TestEvalQueryTemporalAndNarrativePredicates(t *testing.T) {
	_, resolved, index, env := loadExamplesQuery(t)
	tests := []string{
		`structure.vol01.ch01 precedes structure.vol01.ch02`,
		`promises.通灵宝玉来历伏笔.setup_at at_or_before promises.通灵宝玉来历伏笔.payoff_by`,
		`structure.vol01.ch01 in_volume structure.vol01`,
		`structure.vol01.ch01 reveals world.通灵宝玉源起`,
		`structure.vol01.ch01 changes chars.甄士隐.状态 to 出家`,
	}
	for _, expr := range tests {
		result := evalExampleQuery(t, resolved, index, env, expr)
		if result.Value.Kind != EvalBool || !result.Value.Bool {
			t.Fatalf("EvalQuery(%q) = %s, want true", expr, result.Value.String())
		}
	}
}

func TestEvalQueryRejectsNonCollectionCount(t *testing.T) {
	_, resolved, index, env := loadExamplesQuery(t)
	expr, diagnostics := parser.ParseExpression("<query>", `count(structure.vol01.ch01)`)
	if source.HasErrors(diagnostics) {
		t.Fatalf("ParseExpression diagnostics:\n%s", diagnosticsText(diagnostics))
	}
	diagnostics = resolve.ResolveQueryExpr(resolved, env, expr)
	if source.HasErrors(diagnostics) {
		t.Fatalf("ResolveQueryExpr diagnostics:\n%s", diagnosticsText(diagnostics))
	}
	_, diagnostics = EvalQuery(index, env, expr)
	assertDiagnosticsContain(t, diagnostics, "E0904")
}

func TestEvalQueryRejectsNarrativePredicateTargetType(t *testing.T) {
	_, resolved, index, env := loadExamplesQuery(t)
	expr, diagnostics := parser.ParseExpression("<query>", `structure.vol01.ch01 reveals promises.通灵宝玉来历伏笔`)
	if source.HasErrors(diagnostics) {
		t.Fatalf("ParseExpression diagnostics:\n%s", diagnosticsText(diagnostics))
	}
	diagnostics = resolve.ResolveQueryExpr(resolved, env, expr)
	if source.HasErrors(diagnostics) {
		t.Fatalf("ResolveQueryExpr diagnostics:\n%s", diagnosticsText(diagnostics))
	}
	_, diagnostics = EvalQuery(index, env, expr)
	assertDiagnosticsContain(t, diagnostics, "E0905")
}

func TestEvalQueryBuildObjectFields(t *testing.T) {
	_, resolved, index, env := loadExamplesQuery(t)
	tests := []string{
		`build(structure.vol01.ch01).summary.chapter_summary exists`,
		`build(structure.vol01.ch01).state.expected_state_changes exists`,
		`build(structure.vol01.ch01).beats.beat_effects exists`,
		`build(structure.vol01.ch01).prose.target_length exists`,
	}
	for _, text := range tests {
		result := evalExampleQuery(t, resolved, index, env, text)
		if result.Value.Kind != EvalBool || !result.Value.Bool {
			t.Fatalf("EvalQuery(%q) = %s, want true", text, result.Value.String())
		}
	}
}

func evalExampleQuery(t *testing.T, resolved *resolve.Project, index *Index, env resolve.FileEnv, text string) QueryResult {
	t.Helper()
	expr, diagnostics := parser.ParseExpression("<query>", text)
	if source.HasErrors(diagnostics) {
		t.Fatalf("ParseExpression(%q) diagnostics:\n%s", text, diagnosticsText(diagnostics))
	}
	diagnostics = resolve.ResolveQueryExpr(resolved, env, expr)
	if source.HasErrors(diagnostics) {
		t.Fatalf("ResolveQueryExpr(%q) diagnostics:\n%s", text, diagnosticsText(diagnostics))
	}
	result, diagnostics := EvalQuery(index, env, expr)
	if source.HasErrors(diagnostics) {
		t.Fatalf("EvalQuery(%q) diagnostics:\n%s", text, diagnosticsText(diagnostics))
	}
	return result
}

func loadExamplesQuery(t *testing.T) (*project.Project, *resolve.Project, *Index, resolve.FileEnv) {
	t.Helper()
	root := filepath.Join("..", "..", "examples", "红楼梦")
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
	env, diagnostics := resolved.QueryEnv(loaded, "")
	if source.HasErrors(diagnostics) {
		t.Fatalf("QueryEnv diagnostics:\n%s", diagnosticsText(diagnostics))
	}
	return loaded, resolved, index, env
}

func assertDiagnosticsContain(t *testing.T, diagnostics []source.Diagnostic, code string) {
	t.Helper()
	for _, diagnostic := range diagnostics {
		if diagnostic.Code == code {
			return
		}
	}
	t.Fatalf("diagnostics missing %s:\n%s", code, diagnosticsText(diagnostics))
}
