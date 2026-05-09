package project

import "narr/internal/source"

type LoadOptions struct {
	CWD               string
	ProjectDir        string
	ProjectCandidates []string
}

type Project struct {
	Root   string `json:"root"`
	Config Config `json:"config"`
	Files  []File `json:"files"`
}

func Load(options LoadOptions) (*Project, []source.Diagnostic) {
	root, diagnostics := DiscoverRoot(options.CWD, options.ProjectDir, options.ProjectCandidates)
	if source.HasErrors(diagnostics) {
		return nil, diagnostics
	}

	config, configDiagnostics := LoadConfig(root)
	diagnostics = append(diagnostics, configDiagnostics...)

	files, fileDiagnostics := CollectFiles(root)
	diagnostics = append(diagnostics, fileDiagnostics...)

	return &Project{
		Root:   root,
		Config: config,
		Files:  files,
	}, diagnostics
}

func (p *Project) CountFiles(kind FileKind) int {
	count := 0
	for _, file := range p.Files {
		if file.Kind == kind {
			count++
		}
	}
	return count
}
