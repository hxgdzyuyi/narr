package tester

import (
	"fmt"
	"io"
	"path/filepath"
	"sort"
	"strings"

	"narr/internal/source"
	"narr/internal/structure"
)

type Report struct {
	OK          bool                `json:"ok"`
	Total       int                 `json:"total"`
	Passed      int                 `json:"passed"`
	Failed      int                 `json:"failed"`
	Files       []string            `json:"files"`
	Tests       []TestResult        `json:"tests"`
	Diagnostics []source.Diagnostic `json:"diagnostics,omitempty"`
}

type TestResult struct {
	Name     string    `json:"name"`
	File     string    `json:"file"`
	Line     int       `json:"line"`
	Column   int       `json:"column"`
	Status   string    `json:"status"`
	Failures []Failure `json:"failures,omitempty"`
}

type Failure struct {
	File       string                  `json:"file"`
	Line       int                     `json:"line"`
	Column     int                     `json:"column"`
	Statement  string                  `json:"statement"`
	Expression string                  `json:"expression,omitempty"`
	Message    string                  `json:"message,omitempty"`
	Bindings   map[string]BindingValue `json:"bindings,omitempty"`
}

type BindingValue struct {
	Text  string              `json:"text"`
	Value structure.EvalValue `json:"value"`
}

func WriteText(w io.Writer, report Report) {
	for _, diagnostic := range report.Diagnostics {
		fmt.Fprintf(w, "ERROR [%s] %s:%d:%d %s\n", diagnostic.Code, diagnostic.File, diagnostic.Line, diagnostic.Column, diagnostic.Message)
	}
	for _, result := range report.Tests {
		switch result.Status {
		case "pass":
			fmt.Fprintf(w, "PASS %s\n", result.Name)
		default:
			fmt.Fprintf(w, "FAIL %s\n", result.Name)
			for _, failure := range result.Failures {
				fmt.Fprintf(w, "  %s:%d:%d %s\n", failure.File, failure.Line, failure.Column, failure.Message)
				if failure.Expression != "" {
					fmt.Fprintf(w, "    expr: %s\n", failure.Expression)
				}
				if len(failure.Bindings) > 0 {
					fmt.Fprintf(w, "    bindings: %s\n", bindingsText(failure.Bindings))
				}
			}
		}
	}
	fmt.Fprintf(w, "%d tests, %d passed, %d failed\n", report.Total, report.Passed, report.Failed)
}

func relFile(root, path string) string {
	if root == "" || path == "" {
		return filepath.ToSlash(path)
	}
	rel, err := filepath.Rel(root, path)
	if err != nil || strings.HasPrefix(rel, "..") {
		return filepath.ToSlash(path)
	}
	return filepath.ToSlash(rel)
}

func bindingsText(bindings map[string]BindingValue) string {
	names := make([]string, 0, len(bindings))
	for name := range bindings {
		names = append(names, name)
	}
	sort.Strings(names)
	parts := make([]string, 0, len(names))
	for _, name := range names {
		parts = append(parts, name+"="+bindings[name].Text)
	}
	return strings.Join(parts, ", ")
}
