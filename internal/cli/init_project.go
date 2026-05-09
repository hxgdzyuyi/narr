package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"unicode"

	outformat "narr/internal/format"
	"narr/internal/project"
)

type initProjectResult struct {
	OK        bool     `json:"ok"`
	Name      string   `json:"name,omitempty"`
	Namespace string   `json:"namespace,omitempty"`
	Root      string   `json:"root,omitempty"`
	Files     []string `json:"files,omitempty"`
	Skills    []string `json:"skills,omitempty"`
	Error     string   `json:"error,omitempty"`
}

func (a *App) runInitProject(args []string) int {
	parsed, err := parseOptions("init-project", args)
	if err != nil {
		fmt.Fprintln(a.err, "error:", err)
		return 2
	}
	if len(parsed.Positionals) != 1 {
		return a.finishInitProjectError(parsed.Global, "init-project requires exactly one novel name", 2)
	}

	name := strings.TrimSpace(parsed.Positionals[0])
	if name == "" {
		return a.finishInitProjectError(parsed.Global, "novel name must not be empty", 2)
	}

	root := parsed.Command.Dir
	if root == "" {
		root = projectDirName(name)
	}
	root, err = filepath.Abs(root)
	if err != nil {
		return a.finishInitProjectError(parsed.Global, fmt.Sprintf("failed to resolve project directory: %v", err), 1)
	}

	namespace := namespaceFromName(name)
	skills, err := bundledSkillNames()
	if err != nil {
		return a.finishInitProjectError(parsed.Global, err.Error(), 1)
	}

	files, err := createNarrProject(root, name, namespace)
	if err != nil {
		return a.finishInitProjectError(parsed.Global, err.Error(), 1)
	}
	if len(skills) > 0 {
		if err := installBundledSkills(filepath.Join(root, "skills"), skills, false); err != nil {
			return a.finishInitProjectError(parsed.Global, err.Error(), 1)
		}
		if _, err := linkLocalSkillEntrypoints(root, skills, false); err != nil {
			return a.finishInitProjectError(parsed.Global, err.Error(), 1)
		}
		for _, skill := range skills {
			files = append(files, filepath.Join(root, "skills", skill))
		}
	}

	if parsed.Global.JSON {
		_ = outformat.JSON(a.out, initProjectResult{
			OK:        true,
			Name:      name,
			Namespace: namespace,
			Root:      root,
			Files:     files,
			Skills:    skills,
		})
	} else {
		fmt.Fprintf(a.out, "created Narr project %s\n", root)
		fmt.Fprintf(a.out, "project: %s\n", name)
		fmt.Fprintf(a.out, "namespace: %s\n", namespace)
		for _, file := range files {
			fmt.Fprintf(a.out, "wrote %s\n", file)
		}
		fmt.Fprintf(a.out, "next: cd %s && narrc lint --project .\n", root)
	}
	return 0
}

func (a *App) finishInitProjectError(global GlobalOptions, message string, code int) int {
	if global.JSON {
		_ = outformat.JSON(a.out, initProjectResult{OK: false, Error: message})
	} else {
		fmt.Fprintln(a.err, "error:", message)
	}
	return code
}

func createNarrProject(root, name, namespace string) ([]string, error) {
	if err := os.MkdirAll(root, 0o755); err != nil {
		return nil, fmt.Errorf("failed to create %s: %w", root, err)
	}
	dirs := []string{
		filepath.Join(root, "structure"),
		filepath.Join(root, "world"),
		filepath.Join(root, "characters"),
		filepath.Join(root, "threads"),
		filepath.Join(root, "promises"),
		filepath.Join(root, "tests"),
	}
	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return nil, fmt.Errorf("failed to create %s: %w", dir, err)
		}
	}

	files := []struct {
		path string
		text string
	}{
		{
			path: filepath.Join(root, "narr.toml"),
			text: narrToml(name),
		},
		{
			path: filepath.Join(root, "main.narr"),
			text: mainNarr(namespace),
		},
		{
			path: filepath.Join(root, "AGENTS.md"),
			text: agentsMD(name),
		},
	}

	written := make([]string, 0, len(files))
	for _, file := range files {
		if err := writeNewFile(file.path, file.text); err != nil {
			return nil, err
		}
		written = append(written, file.path)
	}
	return written, nil
}

func writeNewFile(path, text string) error {
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644)
	if err != nil {
		if os.IsExist(err) {
			return fmt.Errorf("%s already exists", path)
		}
		return fmt.Errorf("failed to create %s: %w", path, err)
	}
	defer file.Close()
	if _, err := file.WriteString(text); err != nil {
		return fmt.Errorf("failed to write %s: %w", path, err)
	}
	return nil
}

func narrToml(name string) string {
	return fmt.Sprintf(`[project]
name = %s
version = %s
language = "zh-CN"
main = "main.narr"
`, strconv.Quote(name), strconv.Quote(project.SupportedProjectVersion))
}

func mainNarr(namespace string) string {
	return fmt.Sprintf(`namespace %s

// 在这里添加 novel / volume / chapter / beat 等 Narr 蓝图声明。
`, namespace)
}

func agentsMD(name string) string {
	return strings.Join([]string{
		fmt.Sprintf("# AGENTS.md instructions for %s", name),
		"",
		"# Narr Project Guidelines",
		"",
		"## Build And Test",
		"",
		"- Run `narrc lint --project .` after editing `.narr` or `.test.narr` files.",
		"- Run `narrc test --project . --all` after editing project design tests.",
		"- Use `narrc build --project . <chapter_code> --dry-run` before generating chapter build output.",
		"- Use `narrc install-skill --local` from the project root to refresh bundled skills into `./skills`.",
		"",
		"## Project Layout",
		"",
		"- Project config: `narr.toml`.",
		"- Main entry file: `main.narr`.",
		"- Blueprint files: `*.narr`.",
		"- Design test files: `*.test.narr`.",
		"- Local Codex skills: `skills/`.",
		"- Agent skill links: `.agents/skills/` and `.claude/skills/` point back to `skills/`.",
		"",
		"## Implementation Notes",
		"",
		"- Keep Narr syntax aligned with the installed `skills/use-narr-lang/references/syntax.md`.",
		"- Do not add undocumented Narr syntax.",
		"- Only `beat.effect` changes narrative state.",
		"- Keep generated `build/` output out of source control unless explicitly requested.",
		"",
	}, "\n")
}

func projectDirName(name string) string {
	dir := safeIdentifier(name)
	if dir == "" {
		return "narr-project"
	}
	return dir
}

func namespaceFromName(name string) string {
	namespace := safeIdentifier(name)
	if namespace == "" {
		return "novel"
	}
	if first, _ := utf8FirstRune(namespace); unicode.IsDigit(first) {
		return "n" + namespace
	}
	return namespace
}

func safeIdentifier(text string) string {
	var builder strings.Builder
	lastUnderscore := false
	for _, r := range strings.TrimSpace(text) {
		switch {
		case r == '_' || unicode.IsLetter(r) || unicode.IsDigit(r):
			builder.WriteRune(r)
			lastUnderscore = false
		default:
			if builder.Len() > 0 && !lastUnderscore {
				builder.WriteRune('_')
				lastUnderscore = true
			}
		}
	}
	return strings.Trim(builder.String(), "_")
}

func utf8FirstRune(text string) (rune, bool) {
	for _, r := range text {
		return r, true
	}
	return 0, false
}
