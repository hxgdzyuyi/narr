package project

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"

	"github.com/pelletier/go-toml/v2"

	"narr/internal/source"
)

const SupportedProjectVersion = "0.3.5"

type Config struct {
	Name     string `json:"name"`
	Version  string `json:"version"`
	Language string `json:"language,omitempty"`
	Main     string `json:"main,omitempty"`
}

func LoadConfig(root string) (Config, []source.Diagnostic) {
	configPath := filepath.Join(root, "narr.toml")
	data, err := os.ReadFile(configPath)
	if err != nil {
		return Config{}, []source.Diagnostic{
			source.Error("E0002", configPath, 1, 1, "failed to read narr.toml: "+err.Error()),
		}
	}

	var raw map[string]any
	if err := toml.Unmarshal(data, &raw); err != nil {
		return Config{}, []source.Diagnostic{
			source.Error("E0003", configPath, 1, 1, "failed to parse narr.toml: "+err.Error()),
		}
	}

	var diagnostics []source.Diagnostic
	forbiddenTopLevel := map[string]bool{"mode": true, "type": true, "generation": true}
	for key := range raw {
		switch {
		case key == "project":
			continue
		case forbiddenTopLevel[key]:
			diagnostics = append(diagnostics, source.Error("E0006", configPath, 1, 1, fmt.Sprintf("top-level %q is not supported", key)))
		default:
			diagnostics = append(diagnostics, source.Warning("W0001", configPath, 1, 1, fmt.Sprintf("unknown top-level table or key %q", key)))
		}
	}

	projectValue, ok := raw["project"]
	if !ok {
		diagnostics = append(diagnostics, source.Error("E0004", configPath, 1, 1, "missing required [project] table"))
		return Config{}, diagnostics
	}

	projectTable, ok := projectValue.(map[string]any)
	if !ok {
		diagnostics = append(diagnostics, source.Error("E0005", configPath, 1, 1, "[project] must be a TOML table"))
		return Config{}, diagnostics
	}

	config := Config{
		Name:     stringField(projectTable, "name"),
		Version:  stringField(projectTable, "version"),
		Language: stringField(projectTable, "language"),
		Main:     stringField(projectTable, "main"),
	}

	allowedProjectKeys := []string{"name", "version", "language", "main"}
	for key := range projectTable {
		switch {
		case key == "mode" || key == "type" || key == "generation":
			diagnostics = append(diagnostics, source.Error("E0007", configPath, 1, 1, fmt.Sprintf("[project].%s is not supported", key)))
		case !slices.Contains(allowedProjectKeys, key):
			diagnostics = append(diagnostics, source.Warning("W0002", configPath, 1, 1, fmt.Sprintf("unknown [project] field %q", key)))
		}
	}

	if config.Name == "" {
		diagnostics = append(diagnostics, source.Error("E0008", configPath, 1, 1, "[project].name is required"))
	}
	if config.Version == "" {
		diagnostics = append(diagnostics, source.Error("E0009", configPath, 1, 1, "[project].version is required"))
	} else if config.Version != SupportedProjectVersion {
		diagnostics = append(diagnostics, source.Error("E0010", configPath, 1, 1, fmt.Sprintf("[project].version %q is not compatible with %s", config.Version, SupportedProjectVersion)))
	}

	return config, diagnostics
}

func stringField(table map[string]any, key string) string {
	value, ok := table[key]
	if !ok {
		return ""
	}
	text, ok := value.(string)
	if !ok {
		return ""
	}
	return text
}
