package cli

import (
	"fmt"

	"narr/internal/check"
	outformat "narr/internal/format"
	"narr/internal/parser"
	"narr/internal/project"
	"narr/internal/resolve"
	"narr/internal/source"
	statetimeline "narr/internal/state"
	"narr/internal/structure"
)

type lintJSON struct {
	OK          bool                `json:"ok"`
	Project     *project.Project    `json:"project,omitempty"`
	Diagnostics []source.Diagnostic `json:"diagnostics"`
}

func (a *App) runLint(args []string) int {
	parsed, err := parseOptions("lint", args)
	if err != nil {
		fmt.Fprintln(a.err, "error:", err)
		return 2
	}

	loaded, diagnostics := project.Load(project.LoadOptions{
		ProjectDir:        parsed.Global.ProjectDir,
		ProjectCandidates: parsed.Positionals,
	})
	var parsedFiles int
	var resolvedNamespaces int
	var modelEntities int
	var stateCheckpoints int
	var structureViews int
	if loaded != nil && !source.HasErrors(diagnostics) {
		files, parseDiagnostics := parser.ParseProject(loaded)
		parsedFiles = len(files)
		diagnostics = append(diagnostics, parseDiagnostics...)
		if !source.HasErrors(diagnostics) {
			resolved, resolveDiagnostics := resolve.Build(loaded, files)
			diagnostics = append(diagnostics, resolveDiagnostics...)
			if resolved != nil {
				resolvedNamespaces = len(resolved.Namespaces)
			}
			if resolved != nil && !source.HasErrors(diagnostics) {
				result, checkDiagnostics := check.Check(loaded, files, resolved)
				diagnostics = append(diagnostics, checkDiagnostics...)
				if result != nil && result.Model != nil && result.Model.Entities != nil {
					modelEntities = len(result.Model.Entities.All)
				}
				if result != nil && result.Model != nil && !source.HasErrors(diagnostics) {
					timeline, timelineDiagnostics := statetimeline.Build(result.Model, resolved)
					diagnostics = append(diagnostics, timelineDiagnostics...)
					if timeline != nil {
						stateCheckpoints = len(timeline.ChapterBegin) + len(timeline.ChapterEnd) + len(timeline.BeatBefore) + len(timeline.BeatAfter)
					}
					if timeline != nil && !source.HasErrors(diagnostics) {
						structureIndex, structureDiagnostics := structure.Build(result.Model, resolved, timeline)
						diagnostics = append(diagnostics, structureDiagnostics...)
						if structureIndex != nil {
							structureViews = len(structureIndex.Chapters)
						}
					}
				}
			}
		}
	}

	if parsed.Global.JSON {
		_ = outformat.JSON(a.out, lintJSON{
			OK:          !source.HasErrors(diagnostics),
			Project:     loaded,
			Diagnostics: diagnosticsForJSON(diagnostics),
		})
	} else {
		outformat.DiagnosticsText(a.err, diagnostics)
		if loaded != nil && !source.HasErrors(diagnostics) {
			fmt.Fprintf(a.out, "PASS project %s\n", loaded.Root)
			fmt.Fprintf(a.out, "parsed %d files: %d .narr and %d .test.narr\n", parsedFiles, loaded.CountFiles(project.FileKindNarr), loaded.CountFiles(project.FileKindTest))
			fmt.Fprintf(a.out, "resolved %d namespaces\n", resolvedNamespaces)
			fmt.Fprintf(a.out, "built model with %d entities\n", modelEntities)
			fmt.Fprintf(a.out, "built state timeline with %d checkpoints\n", stateCheckpoints)
			fmt.Fprintf(a.out, "built structure views for %d chapters\n", structureViews)
			if parsed.Global.Verbose {
				for _, file := range loaded.Files {
					fmt.Fprintf(a.out, "  %s %s\n", file.Kind, file.Rel)
				}
			}
		}
	}

	if source.HasErrors(diagnostics) {
		return 1
	}
	return 0
}
