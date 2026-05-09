package source

type Severity string

const (
	SeverityError   Severity = "error"
	SeverityWarning Severity = "warning"
	SeverityInfo    Severity = "info"
)

type Diagnostic struct {
	Severity Severity `json:"severity"`
	Code     string   `json:"code"`
	File     string   `json:"file,omitempty"`
	Line     int      `json:"line,omitempty"`
	Column   int      `json:"column,omitempty"`
	Message  string   `json:"message"`
	Hint     string   `json:"hint,omitempty"`
}

func Error(code, file string, line, column int, message string) Diagnostic {
	return Diagnostic{
		Severity: SeverityError,
		Code:     code,
		File:     file,
		Line:     line,
		Column:   column,
		Message:  message,
	}
}

func Warning(code, file string, line, column int, message string) Diagnostic {
	return Diagnostic{
		Severity: SeverityWarning,
		Code:     code,
		File:     file,
		Line:     line,
		Column:   column,
		Message:  message,
	}
}

func Info(code, file string, line, column int, message string) Diagnostic {
	return Diagnostic{
		Severity: SeverityInfo,
		Code:     code,
		File:     file,
		Line:     line,
		Column:   column,
		Message:  message,
	}
}

func HasErrors(diagnostics []Diagnostic) bool {
	for _, diagnostic := range diagnostics {
		if diagnostic.Severity == SeverityError {
			return true
		}
	}
	return false
}
