package cli

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestInstallSkillListIncludesBundledSkill(t *testing.T) {
	code, stdout, stderr := runCLI(t, "install-skill", "--list")
	if code != 0 {
		t.Fatalf("install-skill --list exited %d\nstdout:\n%s\nstderr:\n%s", code, stdout, stderr)
	}
	for _, want := range []string{"use-narr-lang", "use-narrc-cli"} {
		if !strings.Contains(stdout, want) {
			t.Fatalf("list output did not include %s\nstdout:\n%s", want, stdout)
		}
	}
}

func TestInstallSkillInstallsBundledSkillLocally(t *testing.T) {
	withWorkingDir(t, t.TempDir())

	code, stdout, stderr := runCLI(t, "install-skill", "use-narr-lang")
	if code != 0 {
		t.Fatalf("install-skill exited %d\nstdout:\n%s\nstderr:\n%s", code, stdout, stderr)
	}
	wantPath := filepath.Join(mustGetwd(t), "skills", "use-narr-lang", "SKILL.md")
	data, err := os.ReadFile(wantPath)
	if err != nil {
		t.Fatalf("failed to read installed SKILL.md: %v\nstdout:\n%s", err, stdout)
	}
	if !strings.Contains(string(data), "name: use-narr-lang") {
		t.Fatalf("installed SKILL.md did not look valid:\n%s", string(data))
	}
	syntaxPath := filepath.Join(mustGetwd(t), "skills", "use-narr-lang", "references", "syntax.md")
	syntax, err := os.ReadFile(syntaxPath)
	if err != nil {
		t.Fatalf("failed to read embedded syntax reference: %v", err)
	}
	if !strings.Contains(string(syntax), "# Narr 语言规范") {
		t.Fatalf("embedded syntax reference did not contain Narr syntax header")
	}
	if !strings.Contains(stdout, "installed use-narr-lang to ") {
		t.Fatalf("stdout did not mention installation\nstdout:\n%s", stdout)
	}
	assertSkillSymlink(t, mustGetwd(t), ".agents", "use-narr-lang")
	assertSkillSymlink(t, mustGetwd(t), ".claude", "use-narr-lang")
	if !strings.Contains(stdout, "linked ") {
		t.Fatalf("stdout did not mention skill links\nstdout:\n%s", stdout)
	}
}

func TestInstallSkillRefusesOverwriteUnlessForced(t *testing.T) {
	withWorkingDir(t, t.TempDir())

	code, stdout, stderr := runCLI(t, "install-skill", "use-narrc-cli")
	if code != 0 {
		t.Fatalf("initial install exited %d\nstdout:\n%s\nstderr:\n%s", code, stdout, stderr)
	}

	code, stdout, stderr = runCLI(t, "install-skill", "use-narrc-cli")
	if code == 0 {
		t.Fatalf("second install unexpectedly passed\nstdout:\n%s\nstderr:\n%s", stdout, stderr)
	}
	if !strings.Contains(stderr, "already exists") {
		t.Fatalf("stderr did not explain overwrite refusal\nstderr:\n%s", stderr)
	}

	code, stdout, stderr = runCLI(t, "install-skill", "use-narrc-cli", "--force")
	if code != 0 {
		t.Fatalf("forced install exited %d\nstdout:\n%s\nstderr:\n%s", code, stdout, stderr)
	}
}

func TestInstallSkillGlobalUsesCodexHome(t *testing.T) {
	withWorkingDir(t, t.TempDir())
	codexHome := t.TempDir()
	t.Setenv("CODEX_HOME", codexHome)

	code, stdout, stderr := runCLI(t, "install-skill", "use-narrc-cli", "--global")
	if code != 0 {
		t.Fatalf("global install exited %d\nstdout:\n%s\nstderr:\n%s", code, stdout, stderr)
	}
	wantPath := filepath.Join(codexHome, "skills", "use-narrc-cli", "SKILL.md")
	if _, err := os.Stat(wantPath); err != nil {
		t.Fatalf("global install did not write %s: %v", wantPath, err)
	}
	localPath := filepath.Join(mustGetwd(t), "skills", "use-narrc-cli", "SKILL.md")
	if _, err := os.Stat(localPath); !os.IsNotExist(err) {
		t.Fatalf("global install unexpectedly wrote local path %s", localPath)
	}
	if _, err := os.Lstat(filepath.Join(mustGetwd(t), ".agents", "skills", "use-narrc-cli")); !os.IsNotExist(err) {
		t.Fatalf("global install unexpectedly wrote .agents skill link")
	}
	if _, err := os.Lstat(filepath.Join(mustGetwd(t), ".claude", "skills", "use-narrc-cli")); !os.IsNotExist(err) {
		t.Fatalf("global install unexpectedly wrote .claude skill link")
	}
}

func TestInstallSkillJSON(t *testing.T) {
	withWorkingDir(t, t.TempDir())

	code, stdout, stderr := runCLI(t, "install-skill", "use-narrc-cli", "--json")
	if code != 0 {
		t.Fatalf("json install exited %d\nstdout:\n%s\nstderr:\n%s", code, stdout, stderr)
	}
	var result installSkillResult
	if err := json.Unmarshal([]byte(stdout), &result); err != nil {
		t.Fatalf("failed to parse install JSON: %v\nstdout:\n%s", err, stdout)
	}
	if !result.OK || result.Target != "local" || len(result.Installed) != 1 || result.Installed[0] != "use-narrc-cli" {
		t.Fatalf("unexpected install JSON: %+v", result)
	}
	if len(result.Links) != 2 {
		t.Fatalf("expected .agents and .claude links in JSON, got %+v", result)
	}
}

func withWorkingDir(t *testing.T, dir string) {
	t.Helper()
	previous, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get working directory: %v", err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("failed to chdir to %s: %v", dir, err)
	}
	t.Cleanup(func() {
		if err := os.Chdir(previous); err != nil {
			t.Fatalf("failed to restore working directory: %v", err)
		}
	})
}

func mustGetwd(t *testing.T) string {
	t.Helper()
	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get working directory: %v", err)
	}
	return wd
}

func assertSkillSymlink(t *testing.T, projectRoot, agentDir, skill string) {
	t.Helper()
	linkPath := filepath.Join(projectRoot, agentDir, "skills", skill)
	info, err := os.Lstat(linkPath)
	if err != nil {
		t.Fatalf("failed to lstat skill link %s: %v", linkPath, err)
	}
	if info.Mode()&os.ModeSymlink == 0 {
		t.Fatalf("%s is not a symlink", linkPath)
	}
	target, err := os.Readlink(linkPath)
	if err != nil {
		t.Fatalf("failed to read skill link %s: %v", linkPath, err)
	}
	want := filepath.Join("..", "..", "skills", skill)
	if target != want {
		t.Fatalf("skill link %s target = %q, want %q", linkPath, target, want)
	}
}
