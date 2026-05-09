package cli

import (
	"fmt"
	"path/filepath"

	chapterbuild "narr/internal/build"
	"narr/internal/check"
	outformat "narr/internal/format"
	"narr/internal/parser"
	"narr/internal/project"
	"narr/internal/resolve"
	"narr/internal/source"
	statetimeline "narr/internal/state"
	"narr/internal/structure"
)

func (a *App) runBuild(args []string) int {
	parsed, err := parseOptions("build", args)
	if err != nil {
		fmt.Fprintln(a.err, "error:", err)
		return 2
	}
	if len(parsed.Positionals) != 1 {
		fmt.Fprintln(a.err, "error: build requires exactly one chapter_code")
		return 2
	}
	if parsed.Command.LLM && parsed.Global.JSON {
		fmt.Fprintln(a.err, "error: --llm cannot be combined with --json")
		return 2
	}

	loaded, generator, env, diagnostics := loadBuildGenerator(parsed.Global.ProjectDir, "")
	if source.HasErrors(diagnostics) {
		return a.finishPending(parsed.Global, diagnostics)
	}
	chapter, buildDiagnostics := generator.BuildChapter(parsed.Positionals[0], env, source.Span{})
	diagnostics = append(diagnostics, buildDiagnostics...)
	if source.HasErrors(diagnostics) {
		return a.finishPending(parsed.Global, diagnostics)
	}

	llmOutput := !parsed.Global.JSON
	if parsed.Global.JSON {
		_ = outformat.JSON(a.out, map[string]any{
			"ok":      true,
			"dry_run": parsed.Command.DryRun,
			"build":   chapter,
		})
		return 0
	}
	if parsed.Command.DryRun {
		if llmOutput {
			fmt.Fprintf(a.out, "DRY RUN build %s llm ok\n", chapter.Chapter.CanonicalCode)
		} else {
			fmt.Fprintf(a.out, "DRY RUN build %s ok\n", chapter.Chapter.CanonicalCode)
		}
		return 0
	}

	outDir := parsed.Command.OutDir
	if !filepath.IsAbs(outDir) {
		outDir = filepath.Join(loaded.Root, outDir)
	}
	outPath := chapterbuild.OutputPath(outDir, chapter.Chapter.CanonicalCode)
	write := chapterbuild.WriteJSON
	if llmOutput {
		outPath = chapterbuild.LLMOutputPath(outDir, chapter.Chapter.CanonicalCode)
		write = chapterbuild.WriteLLM
	}
	if err := write(outPath, chapter); err != nil {
		diagnostics = append(diagnostics, source.Error("E0806", outPath, 0, 0, err.Error()))
		return a.finishPending(parsed.Global, diagnostics)
	}
	fmt.Fprintf(a.out, "wrote %s\n", outPath)
	fmt.Fprintf(a.out, "chapter: %s\n", chapter.Chapter.CanonicalCode)
	fmt.Fprintf(a.out, "beats: %d\n", len(chapter.Beats.OrderedBeats))
	fmt.Fprintf(a.out, "active_threads: %d\n", len(chapter.Structure.ActiveThreads))
	fmt.Fprintf(a.out, "active_promises: %d\n", len(chapter.Structure.ActivePromises))
	return 0
}

func loadBuildGenerator(projectDir, namespace string) (*project.Project, *chapterbuild.Generator, resolve.FileEnv, []source.Diagnostic) {
	loaded, diagnostics := project.Load(project.LoadOptions{ProjectDir: projectDir})
	if loaded == nil || source.HasErrors(diagnostics) {
		return loaded, nil, resolve.FileEnv{}, diagnostics
	}
	files, parseDiagnostics := parser.ParseProject(loaded)
	diagnostics = append(diagnostics, parseDiagnostics...)
	if source.HasErrors(diagnostics) {
		return loaded, nil, resolve.FileEnv{}, diagnostics
	}
	resolved, resolveDiagnostics := resolve.Build(loaded, files)
	diagnostics = append(diagnostics, resolveDiagnostics...)
	if resolved == nil || source.HasErrors(diagnostics) {
		return loaded, nil, resolve.FileEnv{}, diagnostics
	}
	checked, checkDiagnostics := check.Check(loaded, files, resolved)
	diagnostics = append(diagnostics, checkDiagnostics...)
	if checked == nil || checked.Model == nil || source.HasErrors(diagnostics) {
		return loaded, nil, resolve.FileEnv{}, diagnostics
	}
	timeline, timelineDiagnostics := statetimeline.Build(checked.Model, resolved)
	diagnostics = append(diagnostics, timelineDiagnostics...)
	if timeline == nil || source.HasErrors(diagnostics) {
		return loaded, nil, resolve.FileEnv{}, diagnostics
	}
	structureIndex, structureDiagnostics := structure.Build(checked.Model, resolved, timeline)
	diagnostics = append(diagnostics, structureDiagnostics...)
	if structureIndex == nil || source.HasErrors(diagnostics) {
		return loaded, nil, resolve.FileEnv{}, diagnostics
	}
	env, envDiagnostics := resolved.QueryEnv(loaded, namespace)
	diagnostics = append(diagnostics, envDiagnostics...)
	if source.HasErrors(diagnostics) {
		return loaded, nil, resolve.FileEnv{}, diagnostics
	}
	return loaded, chapterbuild.NewGenerator(checked.Model, resolved, timeline, structureIndex), env, diagnostics
}

func (a *App) finishPending(global GlobalOptions, diagnostics []source.Diagnostic) int {
	if global.JSON {
		_ = outformat.JSON(a.out, map[string]any{
			"ok":          false,
			"diagnostics": diagnosticsForJSON(diagnostics),
		})
	} else {
		outformat.DiagnosticsText(a.err, diagnostics)
	}
	if source.HasErrors(diagnostics) {
		return 1
	}
	return 0
}

func diagnosticsForJSON(diagnostics []source.Diagnostic) []source.Diagnostic {
	if diagnostics == nil {
		return []source.Diagnostic{}
	}
	return diagnostics
}
