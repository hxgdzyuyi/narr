package cli

import (
	"fmt"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"

	bundled "narr"
	outformat "narr/internal/format"
)

type installSkillResult struct {
	OK        bool     `json:"ok"`
	Target    string   `json:"target,omitempty"`
	Root      string   `json:"root,omitempty"`
	Installed []string `json:"installed,omitempty"`
	Links     []string `json:"links,omitempty"`
	Skills    []string `json:"skills,omitempty"`
	Error     string   `json:"error,omitempty"`
}

func (a *App) runInstallSkill(args []string) int {
	parsed, err := parseOptions("install-skill", args)
	if err != nil {
		fmt.Fprintln(a.err, "error:", err)
		return 2
	}
	if parsed.Command.InstallGlobal && parsed.Command.InstallLocal {
		return a.finishInstallSkillError(parsed.Global, "--global and --local cannot be combined", 2)
	}
	if parsed.Command.List && (parsed.Command.All || len(parsed.Positionals) > 0) {
		return a.finishInstallSkillError(parsed.Global, "--list cannot be combined with skill names or --all", 2)
	}
	if parsed.Command.All && len(parsed.Positionals) > 0 {
		return a.finishInstallSkillError(parsed.Global, "install-skill accepts either --all or skill names, not both", 2)
	}

	skills, err := bundledSkillNames()
	if err != nil {
		return a.finishInstallSkillError(parsed.Global, err.Error(), 1)
	}
	if parsed.Command.List {
		if parsed.Global.JSON {
			_ = outformat.JSON(a.out, installSkillResult{OK: true, Skills: skills})
		} else {
			fmt.Fprintln(a.out, "Bundled skills:")
			for _, skill := range skills {
				fmt.Fprintf(a.out, "  %s\n", skill)
			}
		}
		return 0
	}

	selected := parsed.Positionals
	if parsed.Command.All || len(selected) == 0 {
		selected = skills
	}
	if len(selected) == 0 {
		return a.finishInstallSkillError(parsed.Global, "no bundled skills are available", 1)
	}
	if err := validateBundledSkills(skills, selected); err != nil {
		return a.finishInstallSkillError(parsed.Global, err.Error(), 1)
	}

	targetName := "local"
	var projectRoot string
	targetRoot, err := localSkillRoot()
	if parsed.Command.InstallGlobal {
		targetName = "global"
		targetRoot, err = globalSkillRoot()
	} else {
		projectRoot, err = os.Getwd()
	}
	if err != nil {
		return a.finishInstallSkillError(parsed.Global, err.Error(), 1)
	}
	if err := installBundledSkills(targetRoot, selected, parsed.Command.Force); err != nil {
		return a.finishInstallSkillError(parsed.Global, err.Error(), 1)
	}
	var links []string
	if projectRoot != "" {
		links, err = linkLocalSkillEntrypoints(projectRoot, selected, parsed.Command.Force)
		if err != nil {
			return a.finishInstallSkillError(parsed.Global, err.Error(), 1)
		}
	}

	if parsed.Global.JSON {
		_ = outformat.JSON(a.out, installSkillResult{
			OK:        true,
			Target:    targetName,
			Root:      targetRoot,
			Installed: selected,
			Links:     links,
		})
	} else {
		for _, skill := range selected {
			fmt.Fprintf(a.out, "installed %s to %s\n", skill, filepath.Join(targetRoot, skill))
		}
		for _, link := range links {
			fmt.Fprintf(a.out, "linked %s\n", link)
		}
	}
	return 0
}

func (a *App) finishInstallSkillError(global GlobalOptions, message string, code int) int {
	if global.JSON {
		_ = outformat.JSON(a.out, installSkillResult{OK: false, Error: message})
	} else {
		fmt.Fprintln(a.err, "error:", message)
	}
	return code
}

func bundledSkillNames() ([]string, error) {
	entries, err := bundled.BundledSkills.ReadDir("skills")
	if err != nil {
		return nil, fmt.Errorf("failed to read bundled skills: %w", err)
	}
	var names []string
	for _, entry := range entries {
		if entry.IsDir() {
			names = append(names, entry.Name())
		}
	}
	sort.Strings(names)
	return names, nil
}

func validateBundledSkills(available, selected []string) error {
	known := make(map[string]bool, len(available))
	for _, skill := range available {
		known[skill] = true
	}
	for _, skill := range selected {
		if !known[skill] {
			return fmt.Errorf("skill %q is not bundled; run narrc install-skill --list", skill)
		}
	}
	return nil
}

func localSkillRoot() (string, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("failed to resolve current directory: %w", err)
	}
	return filepath.Join(cwd, "skills"), nil
}

func globalSkillRoot() (string, error) {
	codexHome := os.Getenv("CODEX_HOME")
	if codexHome == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("failed to resolve home directory: %w", err)
		}
		codexHome = filepath.Join(home, ".codex")
	}
	return filepath.Join(codexHome, "skills"), nil
}

func installBundledSkills(targetRoot string, skills []string, force bool) error {
	for _, skill := range skills {
		dest := filepath.Join(targetRoot, skill)
		if _, err := os.Stat(dest); err == nil && !force {
			return fmt.Errorf("%s already exists; pass --force to replace it", dest)
		} else if err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("failed to inspect %s: %w", dest, err)
		}
	}
	if err := os.MkdirAll(targetRoot, 0o755); err != nil {
		return fmt.Errorf("failed to create %s: %w", targetRoot, err)
	}
	for _, skill := range skills {
		dest := filepath.Join(targetRoot, skill)
		if force {
			if err := os.RemoveAll(dest); err != nil {
				return fmt.Errorf("failed to replace %s: %w", dest, err)
			}
		}
		if err := copyBundledSkill(skill, dest); err != nil {
			return err
		}
	}
	return nil
}

func linkLocalSkillEntrypoints(projectRoot string, skills []string, force bool) ([]string, error) {
	linkRoots := []string{
		filepath.Join(projectRoot, ".agents", "skills"),
		filepath.Join(projectRoot, ".claude", "skills"),
	}
	var links []string
	for _, linkRoot := range linkRoots {
		if err := os.MkdirAll(linkRoot, 0o755); err != nil {
			return nil, fmt.Errorf("failed to create %s: %w", linkRoot, err)
		}
		for _, skill := range skills {
			linkPath := filepath.Join(linkRoot, skill)
			target := filepath.Join("..", "..", "skills", skill)
			if err := ensureSymlink(linkPath, target, force); err != nil {
				return nil, err
			}
			links = append(links, linkPath)
		}
	}
	return links, nil
}

func ensureSymlink(linkPath, target string, force bool) error {
	info, err := os.Lstat(linkPath)
	if err == nil {
		if info.Mode()&os.ModeSymlink != 0 {
			existing, err := os.Readlink(linkPath)
			if err != nil {
				return fmt.Errorf("failed to read existing symlink %s: %w", linkPath, err)
			}
			if existing == target {
				return nil
			}
		}
		if !force {
			return fmt.Errorf("%s already exists; pass --force to replace it", linkPath)
		}
		if err := os.RemoveAll(linkPath); err != nil {
			return fmt.Errorf("failed to replace %s: %w", linkPath, err)
		}
	} else if !os.IsNotExist(err) {
		return fmt.Errorf("failed to inspect %s: %w", linkPath, err)
	}
	if err := os.Symlink(target, linkPath); err != nil {
		return fmt.Errorf("failed to create symlink %s -> %s: %w", linkPath, target, err)
	}
	return nil
}

func copyBundledSkill(skill, dest string) error {
	srcRoot := path.Join("skills", skill)
	return fs.WalkDir(bundled.BundledSkills, srcRoot, func(src string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		rel := strings.TrimPrefix(strings.TrimPrefix(src, srcRoot), "/")
		target := dest
		if rel != "" {
			target = filepath.Join(dest, filepath.FromSlash(rel))
		}
		if entry.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		data, err := bundled.BundledSkills.ReadFile(src)
		if err != nil {
			return fmt.Errorf("failed to read bundled %s: %w", src, err)
		}
		if err := os.WriteFile(target, data, 0o644); err != nil {
			return fmt.Errorf("failed to write %s: %w", target, err)
		}
		return nil
	})
}
