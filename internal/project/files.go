package project

import (
	"io/fs"
	"path/filepath"
	"sort"
	"strings"

	"narr/internal/source"
)

type FileKind string

const (
	FileKindNarr FileKind = "narr"
	FileKindTest FileKind = "test"
)

type File struct {
	Path string   `json:"path"`
	Rel  string   `json:"rel"`
	Kind FileKind `json:"kind"`
}

func CollectFiles(root string) ([]File, []source.Diagnostic) {
	var files []File
	var diagnostics []source.Diagnostic
	ignoredDirectories := map[string]bool{
		".git":         true,
		"build":        true,
		"vendor":       true,
		"node_modules": true,
	}

	err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			diagnostics = append(diagnostics, source.Error("E0011", path, 0, 0, "failed to read project path: "+err.Error()))
			return nil
		}
		if entry.IsDir() {
			if ignoredDirectories[entry.Name()] {
				return filepath.SkipDir
			}
			return nil
		}

		kind, ok := narrFileKind(entry.Name())
		if !ok {
			return nil
		}

		rel, err := filepath.Rel(root, path)
		if err != nil {
			diagnostics = append(diagnostics, source.Error("E0012", path, 0, 0, "failed to compute project-relative path: "+err.Error()))
			return nil
		}
		files = append(files, File{
			Path: path,
			Rel:  filepath.ToSlash(rel),
			Kind: kind,
		})
		return nil
	})
	if err != nil {
		diagnostics = append(diagnostics, source.Error("E0013", root, 0, 0, "failed to scan project: "+err.Error()))
	}

	sort.Slice(files, func(i, j int) bool {
		return files[i].Rel < files[j].Rel
	})
	return files, diagnostics
}

func narrFileKind(name string) (FileKind, bool) {
	if strings.HasSuffix(name, ".test.narr") {
		return FileKindTest, true
	}
	if strings.HasSuffix(name, ".narr") {
		return FileKindNarr, true
	}
	return "", false
}
