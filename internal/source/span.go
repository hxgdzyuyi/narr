package source

type Position struct {
	File   string `json:"file,omitempty"`
	Line   int    `json:"line,omitempty"`
	Column int    `json:"column,omitempty"`
}

type Span struct {
	Start Position `json:"start"`
	End   Position `json:"end"`
}
