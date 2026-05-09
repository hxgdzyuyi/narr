package cli

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

type buildReport struct {
	OK    bool `json:"ok"`
	Build struct {
		Chapter   goldenChapter   `json:"chapter"`
		Summary   goldenSummary   `json:"summary"`
		Structure goldenStructure `json:"structure"`
		Beats     struct {
			OrderedBeats      []string `json:"ordered_beats"`
			BeatPreconditions []any    `json:"beat_preconditions"`
			BeatEffects       []struct {
				Effects []any `json:"effects"`
			} `json:"beat_effects"`
			BeatRenderHints []any `json:"beat_render_hints"`
		} `json:"beats"`
		State struct {
			StateAtChapterBegin map[string]any `json:"state_at_chapter_begin"`
			ExpectedChanges     []any          `json:"expected_state_changes"`
			StateAtChapterEnd   map[string]any `json:"state_at_chapter_end"`
		} `json:"state"`
	} `json:"build"`
}

type buildGolden struct {
	OK        bool            `json:"ok"`
	Chapter   goldenChapter   `json:"chapter"`
	Summary   goldenSummary   `json:"summary"`
	Structure goldenStructure `json:"structure"`
	Beats     goldenBeats     `json:"beats"`
	State     goldenState     `json:"state"`
}

type goldenChapter struct {
	Code            string `json:"code"`
	CanonicalCode   string `json:"canonical_code"`
	Alias           string `json:"alias,omitempty"`
	Title           string `json:"title,omitempty"`
	Purpose         string `json:"purpose,omitempty"`
	TargetLength    string `json:"target_length,omitempty"`
	VolumeCode      string `json:"volume_code"`
	PreviousChapter string `json:"previous_chapter,omitempty"`
	NextChapter     string `json:"next_chapter,omitempty"`
}

type goldenSummary struct {
	NovelSummary   string `json:"novel_summary,omitempty"`
	VolumeSummary  string `json:"volume_summary,omitempty"`
	ChapterSummary string `json:"chapter_summary,omitempty"`
}

type goldenStructure struct {
	StartPatterns  []string `json:"start_patterns"`
	ActiveThreads  []string `json:"active_threads"`
	ActivePromises []string `json:"active_promises"`
	ActiveArcs     []string `json:"active_arcs"`
	ServedThreads  []string `json:"served_threads"`
	ServedPromises []string `json:"served_promises"`
	ServedArcs     []string `json:"served_arcs"`
}

type goldenBeats struct {
	OrderedCount      int    `json:"ordered_count"`
	FirstOrderedBeat  string `json:"first_ordered_beat"`
	LastOrderedBeat   string `json:"last_ordered_beat"`
	PreconditionCount int    `json:"precondition_count"`
	EffectGroupCount  int    `json:"effect_group_count"`
	EffectCount       int    `json:"effect_count"`
	RenderHintCount   int    `json:"render_hint_count"`
}

type goldenState struct {
	BeginFieldCount     int `json:"begin_field_count"`
	ExpectedChangeCount int `json:"expected_change_count"`
	EndFieldCount       int `json:"end_field_count"`
}

func TestExamplesBuildGoldenJSON(t *testing.T) {
	code, stdout, stderr := runCLI(t, "build", "vol01.ch01", "--project", examplesRoot(t), "--json")
	if code != 0 {
		t.Fatalf("build exited %d\nstdout:\n%s\nstderr:\n%s", code, stdout, stderr)
	}
	var report buildReport
	if err := json.Unmarshal([]byte(stdout), &report); err != nil {
		t.Fatalf("failed to decode build JSON: %v\n%s", err, stdout)
	}
	got := marshalGolden(t, summarizeBuild(report))
	want := readTestdata(t, "golden", "vol01.ch01.build-summary.json")
	if got != want {
		t.Fatalf("build golden mismatch\nwant:\n%s\ngot:\n%s", want, got)
	}
}

func TestExamplesBuildDefaultWritesNarrLikeFile(t *testing.T) {
	outDir := t.TempDir()
	code, stdout, stderr := runCLI(t, "build", "vol01.ch01", "--project", examplesRoot(t), "--out-dir", outDir)
	if code != 0 {
		t.Fatalf("build exited %d\nstdout:\n%s\nstderr:\n%s", code, stdout, stderr)
	}
	outPath := filepath.Join(outDir, "vol01.ch01.build.narr")
	data, err := os.ReadFile(outPath)
	if err != nil {
		t.Fatalf("failed to read LLM output %s: %v\nstdout:\n%s", outPath, err, stdout)
	}
	text := string(data)
	for _, want := range []string{
		"// LLM 使用说明",
		"build vol01.ch01 {",
		"chapter_summary",
		"ordered_beats",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("LLM output did not contain %q:\n%s", want, text)
		}
	}
	if !strings.Contains(stdout, "wrote "+outPath) {
		t.Fatalf("stdout did not mention written file\nstdout:\n%s", stdout)
	}
}

func TestExamplesInfoGoldenText(t *testing.T) {
	code, stdout, stderr := runCLI(t, "info", "vol01.ch01", "--project", examplesRoot(t))
	if code != 0 {
		t.Fatalf("info exited %d\nstdout:\n%s\nstderr:\n%s", code, stdout, stderr)
	}
	want := readTestdata(t, "golden", "vol01.ch01.info.txt")
	if stdout != want {
		t.Fatalf("info golden mismatch\nwant:\n%s\ngot:\n%s", want, stdout)
	}
}

func TestInvalidTestdataDiagnostics(t *testing.T) {
	tests := []struct {
		dir  string
		code string
	}{
		{dir: "duplicate_chapter", code: "E0302"},
		{dir: "invalid_import", code: "E0306"},
		{dir: "effect_unknown_field", code: "E0404"},
		{dir: "test_declares_chapter", code: "E0203"},
	}
	for _, tt := range tests {
		t.Run(tt.dir, func(t *testing.T) {
			projectDir := filepath.Join("testdata", "invalid", tt.dir)
			code, stdout, stderr := runCLI(t, "lint", "--project", projectDir)
			if code == 0 {
				t.Fatalf("lint unexpectedly passed\nstdout:\n%s\nstderr:\n%s", stdout, stderr)
			}
			output := stdout + stderr
			if !strings.Contains(output, "["+tt.code+"]") {
				t.Fatalf("diagnostics did not contain %s\nstdout:\n%s\nstderr:\n%s", tt.code, stdout, stderr)
			}
		})
	}
}

func summarizeBuild(report buildReport) buildGolden {
	effectCount := 0
	for _, group := range report.Build.Beats.BeatEffects {
		effectCount += len(group.Effects)
	}
	ordered := report.Build.Beats.OrderedBeats
	golden := buildGolden{
		OK:        report.OK,
		Chapter:   report.Build.Chapter,
		Summary:   report.Build.Summary,
		Structure: report.Build.Structure,
		Beats: goldenBeats{
			OrderedCount:      len(ordered),
			PreconditionCount: len(report.Build.Beats.BeatPreconditions),
			EffectGroupCount:  len(report.Build.Beats.BeatEffects),
			EffectCount:       effectCount,
			RenderHintCount:   len(report.Build.Beats.BeatRenderHints),
		},
		State: goldenState{
			BeginFieldCount:     len(report.Build.State.StateAtChapterBegin),
			ExpectedChangeCount: len(report.Build.State.ExpectedChanges),
			EndFieldCount:       len(report.Build.State.StateAtChapterEnd),
		},
	}
	if len(ordered) > 0 {
		golden.Beats.FirstOrderedBeat = ordered[0]
		golden.Beats.LastOrderedBeat = ordered[len(ordered)-1]
	}
	return golden
}

func runCLI(t *testing.T, args ...string) (int, string, string) {
	t.Helper()
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Main(args, &stdout, &stderr)
	return code, stdout.String(), stderr.String()
}

func examplesRoot(t *testing.T) string {
	t.Helper()
	root, err := filepath.Abs(filepath.Join("..", "..", "examples", "红楼梦"))
	if err != nil {
		t.Fatalf("failed to resolve examples root: %v", err)
	}
	return root
}

func readTestdata(t *testing.T, parts ...string) string {
	t.Helper()
	pathParts := append([]string{"testdata"}, parts...)
	data, err := os.ReadFile(filepath.Join(pathParts...))
	if err != nil {
		t.Fatalf("failed to read testdata %v: %v", parts, err)
	}
	return string(data)
}

func marshalGolden(t *testing.T, value any) string {
	t.Helper()
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		t.Fatalf("failed to marshal golden value: %v", err)
	}
	return string(data) + "\n"
}
