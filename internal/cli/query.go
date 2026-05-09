package cli

import (
	"fmt"
	"io"

	"narr/internal/check"
	outformat "narr/internal/format"
	"narr/internal/parser"
	"narr/internal/project"
	"narr/internal/resolve"
	"narr/internal/source"
	statetimeline "narr/internal/state"
	"narr/internal/structure"
)

func (a *App) runQuery(args []string) int {
	parsed, err := parseOptions("query", args)
	if err != nil {
		fmt.Fprintln(a.err, "error:", err)
		return 2
	}
	if len(parsed.Positionals) != 1 {
		fmt.Fprintln(a.err, "error: query requires exactly one expression")
		return 2
	}

	loaded, diagnostics := project.Load(project.LoadOptions{ProjectDir: parsed.Global.ProjectDir})
	var result structure.QueryResult
	if loaded != nil && !source.HasErrors(diagnostics) {
		files, parseDiagnostics := parser.ParseProject(loaded)
		diagnostics = append(diagnostics, parseDiagnostics...)
		if !source.HasErrors(diagnostics) {
			resolved, resolveDiagnostics := resolve.Build(loaded, files)
			diagnostics = append(diagnostics, resolveDiagnostics...)
			if resolved != nil && !source.HasErrors(diagnostics) {
				expr, exprDiagnostics := parser.ParseExpression("<query>", parsed.Positionals[0])
				diagnostics = append(diagnostics, exprDiagnostics...)
				if !source.HasErrors(diagnostics) {
					env, envDiagnostics := resolved.QueryEnv(loaded, parsed.Command.Namespace)
					diagnostics = append(diagnostics, envDiagnostics...)
					if !source.HasErrors(diagnostics) {
						diagnostics = append(diagnostics, resolve.ResolveQueryExpr(resolved, env, expr)...)
						if !source.HasErrors(diagnostics) {
							checked, checkDiagnostics := check.Check(loaded, files, resolved)
							diagnostics = append(diagnostics, checkDiagnostics...)
							if checked != nil && checked.Model != nil && !source.HasErrors(diagnostics) {
								timeline, timelineDiagnostics := statetimeline.Build(checked.Model, resolved)
								diagnostics = append(diagnostics, timelineDiagnostics...)
								if timeline != nil && !source.HasErrors(diagnostics) {
									structureIndex, structureDiagnostics := structure.Build(checked.Model, resolved, timeline)
									diagnostics = append(diagnostics, structureDiagnostics...)
									if structureIndex != nil && !source.HasErrors(diagnostics) {
										var queryDiagnostics []source.Diagnostic
										result, queryDiagnostics = structure.EvalQuery(structureIndex, env, expr)
										diagnostics = append(diagnostics, queryDiagnostics...)
									}
								}
							}
						}
					}
				}
			}
		}
	}
	if source.HasErrors(diagnostics) {
		return a.finishPending(parsed.Global, diagnostics)
	}
	if parsed.Global.JSON {
		_ = outformat.JSON(a.out, map[string]any{
			"ok":     true,
			"result": result,
		})
		return 0
	}
	printQueryValue(a.out, result.Value)
	return 0
}

func printQueryValue(w io.Writer, value structure.EvalValue) {
	if value.Kind != structure.EvalList && value.Kind != structure.EvalSet {
		fmt.Fprintf(w, "result: %s\n", value.String())
		return
	}
	fmt.Fprintln(w, "result:")
	if len(value.Items) == 0 {
		fmt.Fprintln(w, "  (none)")
		return
	}
	for _, item := range value.Items {
		fmt.Fprintf(w, "  %s\n", item.String())
	}
}
