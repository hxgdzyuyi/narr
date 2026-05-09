package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"narr/internal/ast"
	"narr/internal/check"
	outformat "narr/internal/format"
	"narr/internal/parser"
	"narr/internal/project"
	"narr/internal/resolve"
	"narr/internal/source"
	statetimeline "narr/internal/state"
	"narr/internal/structure"
	"narr/internal/tester"
)

func (a *App) runTest(args []string) int {
	parsed, err := parseOptions("test", args)
	if err != nil {
		fmt.Fprintln(a.err, "error:", err)
		return 2
	}
	if parsed.Command.All && len(parsed.Positionals) > 0 {
		fmt.Fprintln(a.err, "error: test accepts either --all or files, not both")
		return 2
	}

	loaded, files, resolved, structureIndex, diagnostics := loadTestProject(parsed.Global.ProjectDir, parsed.Positionals)
	if source.HasErrors(diagnostics) {
		return a.finishPending(parsed.Global, diagnostics)
	}
	selected, selectDiagnostics := selectTestFiles(loaded.Root, files, parsed.Command.All, parsed.Positionals)
	diagnostics = append(diagnostics, selectDiagnostics...)
	if source.HasErrors(diagnostics) {
		return a.finishPending(parsed.Global, diagnostics)
	}

	report := tester.Run(tester.Options{
		Root:     loaded.Root,
		Files:    selected,
		Resolved: resolved,
		Index:    structureIndex,
	})
	if parsed.Global.JSON {
		_ = outformat.JSON(a.out, report)
	} else {
		tester.WriteText(a.out, report)
	}
	if !report.OK {
		return 1
	}
	return 0
}

func loadTestProject(projectDir string, candidates []string) (*project.Project, []*ast.File, *resolve.Project, *structure.Index, []source.Diagnostic) {
	loaded, diagnostics := project.Load(project.LoadOptions{
		ProjectDir:        projectDir,
		ProjectCandidates: candidates,
	})
	if loaded == nil || source.HasErrors(diagnostics) {
		return loaded, nil, nil, nil, diagnostics
	}
	files, parseDiagnostics := parser.ParseProject(loaded)
	diagnostics = append(diagnostics, parseDiagnostics...)
	if source.HasErrors(diagnostics) {
		return loaded, files, nil, nil, diagnostics
	}
	resolved, resolveDiagnostics := resolve.Build(loaded, files)
	diagnostics = append(diagnostics, resolveDiagnostics...)
	if resolved == nil || source.HasErrors(diagnostics) {
		return loaded, files, resolved, nil, diagnostics
	}
	checked, checkDiagnostics := check.Check(loaded, files, resolved)
	diagnostics = append(diagnostics, checkDiagnostics...)
	if checked == nil || checked.Model == nil || source.HasErrors(diagnostics) {
		return loaded, files, resolved, nil, diagnostics
	}
	timeline, timelineDiagnostics := statetimeline.Build(checked.Model, resolved)
	diagnostics = append(diagnostics, timelineDiagnostics...)
	if timeline == nil || source.HasErrors(diagnostics) {
		return loaded, files, resolved, nil, diagnostics
	}
	structureIndex, structureDiagnostics := structure.Build(checked.Model, resolved, timeline)
	diagnostics = append(diagnostics, structureDiagnostics...)
	if structureIndex == nil || source.HasErrors(diagnostics) {
		return loaded, files, resolved, structureIndex, diagnostics
	}
	return loaded, files, resolved, structureIndex, diagnostics
}

func selectTestFiles(root string, files []*ast.File, all bool, candidates []string) ([]*ast.File, []source.Diagnostic) {
	var tests []*ast.File
	for _, file := range files {
		if file.Mode == ast.ModeTest {
			tests = append(tests, file)
		}
	}
	if all || len(candidates) == 0 {
		return tests, nil
	}

	selected := make([]*ast.File, 0, len(candidates))
	seen := map[*ast.File]bool{}
	var diagnostics []source.Diagnostic
	for _, candidate := range candidates {
		matched := false
		for _, file := range tests {
			if !testFileMatches(root, file.Path, candidate) {
				continue
			}
			matched = true
			if !seen[file] {
				selected = append(selected, file)
				seen[file] = true
			}
		}
		if !matched {
			diagnostics = append(diagnostics, source.Error("E0807", candidate, 0, 0, "no .test.narr file matches test argument"))
		}
	}
	return selected, diagnostics
}

func testFileMatches(root, path, candidate string) bool {
	candidate = filepath.Clean(candidate)
	rel, err := filepath.Rel(root, path)
	if err == nil {
		rel = filepath.ToSlash(rel)
		candidateSlash := filepath.ToSlash(candidate)
		if rel == candidateSlash || strings.HasPrefix(rel, strings.TrimSuffix(candidateSlash, "/")+"/") {
			return true
		}
	}
	for _, expanded := range expandedCandidatePaths(root, candidate) {
		if path == expanded {
			return true
		}
		info, err := os.Stat(expanded)
		if err == nil && info.IsDir() {
			rel, err := filepath.Rel(expanded, path)
			if err == nil && rel != "." && !strings.HasPrefix(rel, "..") {
				return true
			}
		}
	}
	return false
}

func expandedCandidatePaths(root, candidate string) []string {
	var paths []string
	if filepath.IsAbs(candidate) {
		return []string{candidate}
	}
	if abs, err := filepath.Abs(candidate); err == nil {
		paths = append(paths, abs)
	}
	if root != "" {
		paths = append(paths, filepath.Join(root, candidate))
	}
	return paths
}
