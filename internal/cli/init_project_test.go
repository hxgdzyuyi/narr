package cli

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestInitProjectCreatesEmptyProjectWithBundledSkills(t *testing.T) {
	withWorkingDir(t, t.TempDir())

	code, stdout, stderr := runCLI(t, "init-project", "长夜之城")
	if code != 0 {
		t.Fatalf("init-project exited %d\nstdout:\n%s\nstderr:\n%s", code, stdout, stderr)
	}

	root := filepath.Join(mustGetwd(t), "长夜之城")
	assertFileContains(t, filepath.Join(root, "narr.toml"), `name = "长夜之城"`)
	assertFileContains(t, filepath.Join(root, "main.narr"), "namespace 长夜之城")
	assertFileContains(t, filepath.Join(root, "AGENTS.md"), "narrc install-skill --local")
	assertFileContains(t, filepath.Join(root, "skills", "use-narrc-cli", "SKILL.md"), "name: use-narrc-cli")
	assertFileContains(t, filepath.Join(root, "skills", "use-narr-lang", "SKILL.md"), "name: use-narr-lang")
	assertFileContains(t, filepath.Join(root, "skills", "use-narr-lang", "references", "syntax.md"), "# Narr 语言规范")
	assertSkillSymlink(t, root, ".agents", "use-narrc-cli")
	assertSkillSymlink(t, root, ".agents", "use-narr-lang")
	assertSkillSymlink(t, root, ".claude", "use-narrc-cli")
	assertSkillSymlink(t, root, ".claude", "use-narr-lang")

	code, lintStdout, lintStderr := runCLI(t, "lint", "--project", root)
	if code != 0 {
		t.Fatalf("lint on initialized project exited %d\nstdout:\n%s\nstderr:\n%s", code, lintStdout, lintStderr)
	}
	if !strings.Contains(stdout, "created Narr project") {
		t.Fatalf("stdout did not mention project creation\nstdout:\n%s", stdout)
	}
}

func TestInitProjectUsesExplicitDirAndSafeNamespace(t *testing.T) {
	withWorkingDir(t, t.TempDir())
	root := filepath.Join(mustGetwd(t), "custom-project")

	code, stdout, stderr := runCLI(t, "init-project", "123 My Novel!", "--dir", root)
	if code != 0 {
		t.Fatalf("init-project --dir exited %d\nstdout:\n%s\nstderr:\n%s", code, stdout, stderr)
	}
	assertFileContains(t, filepath.Join(root, "main.narr"), "namespace n123_My_Novel")
	if _, err := os.Stat(filepath.Join(root, "skills", "use-narr-lang", "references", "syntax.md")); err != nil {
		t.Fatalf("explicit dir project did not get installed skill: %v", err)
	}
}

func TestInitProjectRefusesExistingProjectFiles(t *testing.T) {
	withWorkingDir(t, t.TempDir())

	code, stdout, stderr := runCLI(t, "init-project", "长夜之城")
	if code != 0 {
		t.Fatalf("initial init-project exited %d\nstdout:\n%s\nstderr:\n%s", code, stdout, stderr)
	}

	code, stdout, stderr = runCLI(t, "init-project", "长夜之城")
	if code == 0 {
		t.Fatalf("second init-project unexpectedly passed\nstdout:\n%s\nstderr:\n%s", stdout, stderr)
	}
	if !strings.Contains(stderr, "already exists") {
		t.Fatalf("stderr did not explain existing file refusal\nstderr:\n%s", stderr)
	}
}

func TestInitProjectJSON(t *testing.T) {
	withWorkingDir(t, t.TempDir())
	root := filepath.Join(mustGetwd(t), "json-project")

	code, stdout, stderr := runCLI(t, "init-project", "九州别记", "--dir", root, "--json")
	if code != 0 {
		t.Fatalf("json init-project exited %d\nstdout:\n%s\nstderr:\n%s", code, stdout, stderr)
	}
	var result initProjectResult
	if err := json.Unmarshal([]byte(stdout), &result); err != nil {
		t.Fatalf("failed to parse init-project JSON: %v\nstdout:\n%s", err, stdout)
	}
	if !result.OK || result.Name != "九州别记" || result.Namespace != "九州别记" || result.Root != root {
		t.Fatalf("unexpected init-project JSON: %+v", result)
	}
	if !stringSliceContains(result.Skills, "use-narrc-cli") || !stringSliceContains(result.Skills, "use-narr-lang") {
		t.Fatalf("JSON did not include installed skills: %+v", result)
	}
}

func assertFileContains(t *testing.T, path, want string) {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("failed to read %s: %v", path, err)
	}
	if !strings.Contains(string(data), want) {
		t.Fatalf("%s did not contain %q:\n%s", path, want, string(data))
	}
}

func stringSliceContains(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}
