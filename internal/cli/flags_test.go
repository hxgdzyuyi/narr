package cli

import "testing"

func TestParseOptionsAcceptsGlobalFlagsAfterPositionals(t *testing.T) {
	parsed, err := parseOptions("lint", []string{"structure", "--project", "examples/project", "--json", "--verbose"})
	if err != nil {
		t.Fatalf("parseOptions returned error: %v", err)
	}
	if parsed.Global.ProjectDir != "examples/project" {
		t.Fatalf("ProjectDir = %q, want examples/project", parsed.Global.ProjectDir)
	}
	if !parsed.Global.JSON || !parsed.Global.Verbose {
		t.Fatalf("global bool flags were not parsed: %+v", parsed.Global)
	}
	if got := len(parsed.Positionals); got != 1 {
		t.Fatalf("len(Positionals) = %d, want 1", got)
	}
	if parsed.Positionals[0] != "structure" {
		t.Fatalf("Positionals[0] = %q, want structure", parsed.Positionals[0])
	}
}

func TestSplitCommandAllowsGlobalFlagsBeforeCommand(t *testing.T) {
	pre, command, rest, err := splitCommand([]string{"--project", "examples/project", "--verbose", "lint", "structure"})
	if err != nil {
		t.Fatalf("splitCommand returned error: %v", err)
	}
	if command != "lint" {
		t.Fatalf("command = %q, want lint", command)
	}
	if len(pre) != 3 {
		t.Fatalf("len(pre) = %d, want 3", len(pre))
	}
	if len(rest) != 1 || rest[0] != "structure" {
		t.Fatalf("rest = %#v, want [structure]", rest)
	}
}

func TestParseOptionsAcceptsBuildLLM(t *testing.T) {
	parsed, err := parseOptions("build", []string{"--llm", "vol01.ch01"})
	if err != nil {
		t.Fatalf("parseOptions returned error: %v", err)
	}
	if !parsed.Command.LLM {
		t.Fatalf("LLM = false, want true")
	}
	if len(parsed.Positionals) != 1 || parsed.Positionals[0] != "vol01.ch01" {
		t.Fatalf("Positionals = %#v, want [vol01.ch01]", parsed.Positionals)
	}
}

func TestParseOptionsAcceptsBuildAll(t *testing.T) {
	parsed, err := parseOptions("build", []string{"--all"})
	if err != nil {
		t.Fatalf("parseOptions returned error: %v", err)
	}
	if !parsed.Command.All {
		t.Fatalf("All = false, want true")
	}
	if len(parsed.Positionals) != 0 {
		t.Fatalf("Positionals = %#v, want []", parsed.Positionals)
	}
}
