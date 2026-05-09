package project

import (
	"os"
	"path/filepath"
	"testing"

	"narr/internal/source"
)

func TestLoadConfigValidProject(t *testing.T) {
	root := t.TempDir()
	writeConfig(t, root, `[project]
name = "Example"
version = "0.3.5"
language = "zh-CN"
main = "main.narr"
`)

	config, diagnostics := LoadConfig(root)
	if source.HasErrors(diagnostics) {
		t.Fatalf("LoadConfig returned errors: %#v", diagnostics)
	}
	if config.Name != "Example" {
		t.Fatalf("Name = %q, want Example", config.Name)
	}
	if config.Version != SupportedProjectVersion {
		t.Fatalf("Version = %q, want %s", config.Version, SupportedProjectVersion)
	}
	if config.Main != "main.narr" {
		t.Fatalf("Main = %q, want main.narr", config.Main)
	}
}

func TestLoadConfigRejectsUnsupportedVersion(t *testing.T) {
	root := t.TempDir()
	writeConfig(t, root, `[project]
name = "Example"
version = "9.9.9"
`)

	_, diagnostics := LoadConfig(root)
	if !source.HasErrors(diagnostics) {
		t.Fatalf("LoadConfig returned no errors for unsupported version: %#v", diagnostics)
	}
}

func writeConfig(t *testing.T, root, text string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(root, "narr.toml"), []byte(text), 0o644); err != nil {
		t.Fatalf("failed to write narr.toml: %v", err)
	}
}
