package cli

import (
	"fmt"
	"strings"
)

type GlobalOptions struct {
	ProjectDir string
	JSON       bool
	Verbose    bool
	NoColor    bool
}

type CommandOptions struct {
	All           bool
	Dir           string
	DryRun        bool
	Force         bool
	InstallGlobal bool
	InstallLocal  bool
	List          bool
	LLM           bool
	OutDir        string
	Namespace     string
}

type ParsedOptions struct {
	Global      GlobalOptions
	Command     CommandOptions
	Positionals []string
}

func parseOptions(command string, args []string) (ParsedOptions, error) {
	var parsed ParsedOptions
	for i := 0; i < len(args); i++ {
		arg := args[i]
		if arg == "--" {
			parsed.Positionals = append(parsed.Positionals, args[i+1:]...)
			break
		}
		if !strings.HasPrefix(arg, "--") || arg == "--" {
			parsed.Positionals = append(parsed.Positionals, arg)
			continue
		}

		name, value, hasValue := splitOption(arg)
		switch name {
		case "--project":
			text, next, err := optionValue(name, value, hasValue, args, i)
			if err != nil {
				return parsed, err
			}
			parsed.Global.ProjectDir = text
			i = next
		case "--json":
			if hasValue {
				return parsed, fmt.Errorf("%s does not accept a value", name)
			}
			parsed.Global.JSON = true
		case "--verbose":
			if hasValue {
				return parsed, fmt.Errorf("%s does not accept a value", name)
			}
			parsed.Global.Verbose = true
		case "--no-color":
			if hasValue {
				return parsed, fmt.Errorf("%s does not accept a value", name)
			}
			parsed.Global.NoColor = true
		case "--all":
			if command != "test" && command != "install-skill" {
				return parsed, fmt.Errorf("%s is not valid for %s", name, command)
			}
			if hasValue {
				return parsed, fmt.Errorf("%s does not accept a value", name)
			}
			parsed.Command.All = true
		case "--force":
			if command != "install-skill" {
				return parsed, fmt.Errorf("%s is not valid for %s", name, command)
			}
			if hasValue {
				return parsed, fmt.Errorf("%s does not accept a value", name)
			}
			parsed.Command.Force = true
		case "--global":
			if command != "install-skill" {
				return parsed, fmt.Errorf("%s is not valid for %s", name, command)
			}
			if hasValue {
				return parsed, fmt.Errorf("%s does not accept a value", name)
			}
			parsed.Command.InstallGlobal = true
		case "--local":
			if command != "install-skill" {
				return parsed, fmt.Errorf("%s is not valid for %s", name, command)
			}
			if hasValue {
				return parsed, fmt.Errorf("%s does not accept a value", name)
			}
			parsed.Command.InstallLocal = true
		case "--list":
			if command != "install-skill" {
				return parsed, fmt.Errorf("%s is not valid for %s", name, command)
			}
			if hasValue {
				return parsed, fmt.Errorf("%s does not accept a value", name)
			}
			parsed.Command.List = true
		case "--dir":
			if command != "init-project" {
				return parsed, fmt.Errorf("%s is not valid for %s", name, command)
			}
			text, next, err := optionValue(name, value, hasValue, args, i)
			if err != nil {
				return parsed, err
			}
			parsed.Command.Dir = text
			i = next
		case "--dry-run":
			if command != "build" {
				return parsed, fmt.Errorf("%s is not valid for %s", name, command)
			}
			if hasValue {
				return parsed, fmt.Errorf("%s does not accept a value", name)
			}
			parsed.Command.DryRun = true
		case "--llm":
			if command != "build" {
				return parsed, fmt.Errorf("%s is not valid for %s", name, command)
			}
			if hasValue {
				return parsed, fmt.Errorf("%s does not accept a value", name)
			}
			parsed.Command.LLM = true
		case "--out-dir":
			if command != "build" {
				return parsed, fmt.Errorf("%s is not valid for %s", name, command)
			}
			text, next, err := optionValue(name, value, hasValue, args, i)
			if err != nil {
				return parsed, err
			}
			parsed.Command.OutDir = text
			i = next
		case "--namespace":
			if command != "query" {
				return parsed, fmt.Errorf("%s is not valid for %s", name, command)
			}
			text, next, err := optionValue(name, value, hasValue, args, i)
			if err != nil {
				return parsed, err
			}
			parsed.Command.Namespace = text
			i = next
		default:
			return parsed, fmt.Errorf("unknown option %s", name)
		}
	}
	if parsed.Command.OutDir == "" {
		parsed.Command.OutDir = "build"
	}
	return parsed, nil
}

func splitOption(arg string) (name, value string, hasValue bool) {
	before, after, found := strings.Cut(arg, "=")
	if !found {
		return arg, "", false
	}
	return before, after, true
}

func optionValue(name, value string, hasValue bool, args []string, index int) (string, int, error) {
	if hasValue {
		if value == "" {
			return "", index, fmt.Errorf("%s requires a non-empty value", name)
		}
		return value, index, nil
	}
	next := index + 1
	if next >= len(args) {
		return "", index, fmt.Errorf("%s requires a value", name)
	}
	if strings.HasPrefix(args[next], "--") {
		return "", index, fmt.Errorf("%s requires a value", name)
	}
	return args[next], next, nil
}

func splitCommand(args []string) ([]string, string, []string, error) {
	var preCommand []string
	for i := 0; i < len(args); i++ {
		arg := args[i]
		if isRootBoolOption(arg) {
			preCommand = append(preCommand, arg)
			continue
		}
		if isRootValueOption(arg) {
			preCommand = append(preCommand, arg)
			if !strings.Contains(arg, "=") {
				if i+1 >= len(args) {
					return nil, "", nil, fmt.Errorf("%s requires a value", arg)
				}
				i++
				preCommand = append(preCommand, args[i])
			}
			continue
		}
		if strings.HasPrefix(arg, "--") {
			return nil, "", nil, fmt.Errorf("unknown option %s", arg)
		}
		return preCommand, arg, args[i+1:], nil
	}
	return preCommand, "", nil, nil
}

func isRootBoolOption(arg string) bool {
	switch arg {
	case "--json", "--verbose", "--no-color":
		return true
	default:
		return false
	}
}

func isRootValueOption(arg string) bool {
	return arg == "--project" || strings.HasPrefix(arg, "--project=")
}
