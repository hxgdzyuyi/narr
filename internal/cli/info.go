package cli

import (
	"fmt"

	outformat "narr/internal/format"
	"narr/internal/source"
)

func (a *App) runInfo(args []string) int {
	parsed, err := parseOptions("info", args)
	if err != nil {
		fmt.Fprintln(a.err, "error:", err)
		return 2
	}
	if len(parsed.Positionals) != 1 {
		fmt.Fprintln(a.err, "error: info requires exactly one chapter_code")
		return 2
	}

	_, generator, env, diagnostics := loadBuildGenerator(parsed.Global.ProjectDir, "")
	if source.HasErrors(diagnostics) {
		return a.finishPending(parsed.Global, diagnostics)
	}
	chapter, buildDiagnostics := generator.BuildChapter(parsed.Positionals[0], env, source.Span{})
	diagnostics = append(diagnostics, buildDiagnostics...)
	if source.HasErrors(diagnostics) {
		return a.finishPending(parsed.Global, diagnostics)
	}
	if parsed.Global.JSON {
		_ = outformat.JSON(a.out, map[string]any{
			"ok":      true,
			"chapter": chapter,
		})
		return 0
	}
	fmt.Fprintf(a.out, "chapter %s", chapter.Chapter.CanonicalCode)
	if chapter.Chapter.Title != "" {
		fmt.Fprintf(a.out, " - %s", chapter.Chapter.Title)
	}
	fmt.Fprintln(a.out)
	if chapter.Chapter.Alias != "" {
		fmt.Fprintf(a.out, "alias: %s\n", chapter.Chapter.Alias)
	}
	fmt.Fprintf(a.out, "volume: %s\n", chapter.Chapter.VolumeCode)
	if chapter.Chapter.PreviousChapter != "" {
		fmt.Fprintf(a.out, "previous: %s\n", chapter.Chapter.PreviousChapter)
	}
	if chapter.Chapter.NextChapter != "" {
		fmt.Fprintf(a.out, "next: %s\n", chapter.Chapter.NextChapter)
	}
	if chapter.Summary.ChapterSummary != "" {
		fmt.Fprintf(a.out, "summary: %s\n", chapter.Summary.ChapterSummary)
	}
	fmt.Fprintf(a.out, "beats: %d\n", len(chapter.Beats.OrderedBeats))
	fmt.Fprintf(a.out, "state changes: %d\n", len(chapter.State.ExpectedChanges))
	fmt.Fprintf(a.out, "active threads: %d\n", len(chapter.Structure.ActiveThreads))
	fmt.Fprintf(a.out, "active promises: %d\n", len(chapter.Structure.ActivePromises))
	fmt.Fprintf(a.out, "served promises: %d\n", len(chapter.Structure.ServedPromises))
	if len(chapter.Structure.ActiveThreads) > 0 {
		fmt.Fprintln(a.out, "active_threads:")
		for _, id := range chapter.Structure.ActiveThreads {
			fmt.Fprintf(a.out, "  %s\n", id)
		}
	}
	if len(chapter.Structure.ServedPromises) > 0 {
		fmt.Fprintln(a.out, "served_promises:")
		for _, id := range chapter.Structure.ServedPromises {
			fmt.Fprintf(a.out, "  %s\n", id)
		}
	}
	return 0
}
