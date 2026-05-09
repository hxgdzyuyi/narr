package format

import (
	"fmt"
	"io"
	"strings"

	"narr/internal/source"
)

func DiagnosticsText(w io.Writer, diagnostics []source.Diagnostic) {
	for _, diagnostic := range diagnostics {
		fmt.Fprintln(w, FormatDiagnostic(diagnostic))
	}
}

func FormatDiagnostic(diagnostic source.Diagnostic) string {
	var builder strings.Builder
	builder.WriteString(strings.ToUpper(string(diagnostic.Severity)))
	if diagnostic.Code != "" {
		builder.WriteString(" [")
		builder.WriteString(diagnostic.Code)
		builder.WriteString("]")
	}
	if diagnostic.File != "" {
		builder.WriteString(" ")
		builder.WriteString(diagnostic.File)
		if diagnostic.Line > 0 {
			builder.WriteString(fmt.Sprintf(":%d", diagnostic.Line))
			if diagnostic.Column > 0 {
				builder.WriteString(fmt.Sprintf(":%d", diagnostic.Column))
			}
		}
	}
	builder.WriteString(" ")
	builder.WriteString(diagnostic.Message)
	if diagnostic.Hint != "" {
		builder.WriteString("\n  hint: ")
		builder.WriteString(diagnostic.Hint)
	}
	return builder.String()
}
