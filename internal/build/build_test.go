package build

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"narr/internal/check"
	"narr/internal/parser"
	"narr/internal/project"
	"narr/internal/resolve"
	"narr/internal/source"
	statetimeline "narr/internal/state"
	"narr/internal/structure"
)

func TestBuildExamplesChapter(t *testing.T) {
	generator, env := loadExamplesGenerator(t)
	build, diagnostics := generator.BuildChapter("vol01.ch01", env, source.Span{})
	if source.HasErrors(diagnostics) {
		t.Fatalf("BuildChapter diagnostics:\n%s", diagnosticsText(diagnostics))
	}
	if build.Chapter.CanonicalCode != "vol01.ch01" {
		t.Fatalf("canonical code = %q, want vol01.ch01", build.Chapter.CanonicalCode)
	}
	if build.Chapter.Title == "" || build.Summary.ChapterSummary == "" {
		t.Fatalf("chapter metadata was not populated: %+v %+v", build.Chapter, build.Summary)
	}
	if got := len(build.Beats.OrderedBeats); got != 14 {
		t.Fatalf("len(ordered beats) = %d, want 14", got)
	}
	if len(build.Structure.ActiveThreads) == 0 || len(build.Structure.ServedPromises) == 0 {
		t.Fatalf("structure view was not populated: %+v", build.Structure)
	}
	if len(build.State.ExpectedChanges) == 0 {
		t.Fatalf("expected state changes were not populated")
	}
}

func TestWriteJSON(t *testing.T) {
	generator, env := loadExamplesGenerator(t)
	build, diagnostics := generator.BuildChapter("vol01.ch01", env, source.Span{})
	if source.HasErrors(diagnostics) {
		t.Fatalf("BuildChapter diagnostics:\n%s", diagnosticsText(diagnostics))
	}
	path := OutputPath(t.TempDir(), build.Chapter.CanonicalCode)
	if err := WriteJSON(path, build); err != nil {
		t.Fatalf("WriteJSON returned error: %v", err)
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("expected JSON file at %s: %v", path, err)
	}
}

func TestWriteLLM(t *testing.T) {
	generator, env := loadExamplesGenerator(t)
	build, diagnostics := generator.BuildChapter("vol01.ch01", env, source.Span{})
	if source.HasErrors(diagnostics) {
		t.Fatalf("BuildChapter diagnostics:\n%s", diagnosticsText(diagnostics))
	}
	path := LLMOutputPath(t.TempDir(), build.Chapter.CanonicalCode)
	if err := WriteLLM(path, build); err != nil {
		t.Fatalf("WriteLLM returned error: %v", err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("expected LLM file at %s: %v", path, err)
	}
	text := string(data)
	for _, want := range []string{
		"// LLM 使用说明",
		"// build.narr 简易语法与属性说明",
		"// chapter.canonical_code：编译器归一化后的章节号，作为 build 主键。",
		"// beats.beat_effects：每个 beat 必须造成的状态变化；effect 行使用 target op value。",
		"build vol01.ch01 {",
		"state_at_chapter_begin",
		"ordered_beats",
		"prose:",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("LLM output did not contain %q:\n%s", want, text)
		}
	}
}

func loadExamplesGenerator(t *testing.T) (*Generator, resolve.FileEnv) {
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
	index, diagnostics := structure.Build(checked.Model, resolved, timeline)
	if source.HasErrors(diagnostics) {
		t.Fatalf("structure diagnostics:\n%s", diagnosticsText(diagnostics))
	}
	env, diagnostics := resolved.QueryEnv(loaded, "")
	if source.HasErrors(diagnostics) {
		t.Fatalf("QueryEnv diagnostics:\n%s", diagnosticsText(diagnostics))
	}
	return NewGenerator(checked.Model, resolved, timeline, index), env
}

func diagnosticsText(diagnostics []source.Diagnostic) string {
	out := ""
	for _, diagnostic := range diagnostics {
		out += diagnostic.Code + " " + diagnostic.File + " " + diagnostic.Message + "\n"
	}
	return out
}
