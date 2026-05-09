package project

import (
	"os"
	"path/filepath"
	"testing"

	"narr/internal/source"
)

func TestDiscoverRootSearchesUpward(t *testing.T) {
	root := t.TempDir()
	writeConfig(t, root, `[project]
name = "Example"
version = "0.3.5"
`)
	nested := filepath.Join(root, "a", "b")
	if err := os.MkdirAll(nested, 0o755); err != nil {
		t.Fatalf("failed to create nested dir: %v", err)
	}

	discovered, diagnostics := DiscoverRoot(nested, "", nil)
	if source.HasErrors(diagnostics) {
		t.Fatalf("DiscoverRoot returned errors: %#v", diagnostics)
	}
	if discovered != root {
		t.Fatalf("discovered = %q, want %q", discovered, root)
	}
}

func TestDiscoverRootUsesDirectoryCandidate(t *testing.T) {
	cwd := t.TempDir()
	root := filepath.Join(cwd, "story")
	if err := os.MkdirAll(root, 0o755); err != nil {
		t.Fatalf("failed to create project dir: %v", err)
	}
	writeConfig(t, root, `[project]
name = "Example"
version = "0.3.5"
`)

	discovered, diagnostics := DiscoverRoot(cwd, "", []string{"story"})
	if source.HasErrors(diagnostics) {
		t.Fatalf("DiscoverRoot returned errors: %#v", diagnostics)
	}
	if discovered != root {
		t.Fatalf("discovered = %q, want %q", discovered, root)
	}
}
