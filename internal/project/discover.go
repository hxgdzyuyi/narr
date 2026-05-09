package project

import (
	"os"
	"path/filepath"

	"narr/internal/source"
)

func DiscoverRoot(cwd, explicitProject string, candidates []string) (string, []source.Diagnostic) {
	if cwd == "" {
		var err error
		cwd, err = os.Getwd()
		if err != nil {
			return "", []source.Diagnostic{
				source.Error("E0000", "", 0, 0, "failed to determine current directory: "+err.Error()),
			}
		}
	}

	if explicitProject != "" {
		root, err := filepath.Abs(explicitProject)
		if err != nil {
			return "", []source.Diagnostic{
				source.Error("E0001", explicitProject, 0, 0, "invalid --project path: "+err.Error()),
			}
		}
		if !hasNarrConfig(root) {
			return "", []source.Diagnostic{
				source.Error("E0001", filepath.Join(root, "narr.toml"), 1, 1, "--project directory does not contain narr.toml"),
			}
		}
		return root, nil
	}

	for _, candidate := range candidates {
		root, ok := projectRootFromCandidate(cwd, candidate)
		if ok {
			return root, nil
		}
	}

	root, ok := searchUpward(cwd)
	if !ok {
		return "", []source.Diagnostic{
			source.Error("E0001", cwd, 0, 0, "could not find narr.toml; pass --project DIR or run inside a Narr project"),
		}
	}
	return root, nil
}

func projectRootFromCandidate(cwd, candidate string) (string, bool) {
	if candidate == "" {
		return "", false
	}
	path := candidate
	if !filepath.IsAbs(path) {
		path = filepath.Join(cwd, path)
	}
	info, err := os.Stat(path)
	if err != nil || !info.IsDir() {
		return "", false
	}
	root, err := filepath.Abs(path)
	if err != nil {
		return "", false
	}
	return root, hasNarrConfig(root)
}

func searchUpward(start string) (string, bool) {
	current, err := filepath.Abs(start)
	if err != nil {
		return "", false
	}
	for {
		if hasNarrConfig(current) {
			return current, true
		}
		parent := filepath.Dir(current)
		if parent == current {
			return "", false
		}
		current = parent
	}
}

func hasNarrConfig(root string) bool {
	info, err := os.Stat(filepath.Join(root, "narr.toml"))
	return err == nil && !info.IsDir()
}
