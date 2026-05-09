package source

type Reporter struct {
	Diagnostics []Diagnostic
}

func (r *Reporter) Add(diagnostic Diagnostic) {
	r.Diagnostics = append(r.Diagnostics, diagnostic)
}

func (r *Reporter) Error(code, file string, line, column int, message string) {
	r.Add(Error(code, file, line, column, message))
}

func (r *Reporter) Warning(code, file string, line, column int, message string) {
	r.Add(Warning(code, file, line, column, message))
}

func (r *Reporter) HasErrors() bool {
	return HasErrors(r.Diagnostics)
}
