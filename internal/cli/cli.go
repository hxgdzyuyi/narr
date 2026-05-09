package cli

import (
	"fmt"
	"io"
)

const Version = "0.1.0-m9"

type App struct {
	out io.Writer
	err io.Writer
}

func Main(args []string, out, err io.Writer) int {
	return (&App{out: out, err: err}).Run(args)
}

func (a *App) Run(args []string) int {
	if len(args) == 0 {
		a.usage(a.err)
		return 2
	}

	switch args[0] {
	case "--version", "version":
		fmt.Fprintln(a.out, Version)
		return 0
	case "--help", "-h", "help":
		a.usage(a.out)
		return 0
	}

	preCommand, command, rest, err := splitCommand(args)
	if err != nil {
		fmt.Fprintln(a.err, "error:", err)
		return 2
	}
	if command == "" {
		a.usage(a.err)
		return 2
	}

	commandArgs := append([]string{}, preCommand...)
	commandArgs = append(commandArgs, rest...)

	switch command {
	case "lint":
		return a.runLint(commandArgs)
	case "build":
		return a.runBuild(commandArgs)
	case "test":
		return a.runTest(commandArgs)
	case "info":
		return a.runInfo(commandArgs)
	case "query":
		return a.runQuery(commandArgs)
	case "install-skill":
		return a.runInstallSkill(commandArgs)
	case "init-project":
		return a.runInitProject(commandArgs)
	default:
		fmt.Fprintf(a.err, "error: unknown command %q\n", command)
		a.usage(a.err)
		return 2
	}
}

func (a *App) usage(w io.Writer) {
	fmt.Fprintln(w, "Usage: narrc [global options] <command> [options] <args>")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "Commands:")
	fmt.Fprintln(w, "  build <chapter_code>|--all  build chapter artifact files")
	fmt.Fprintln(w, "  test [files|--all]    run .test.narr declarations")
	fmt.Fprintln(w, "  lint [files]          load and check a Narr project")
	fmt.Fprintln(w, "  info <chapter_code>   show chapter build/state summary")
	fmt.Fprintln(w, "  query <expr>          evaluate a query expression")
	fmt.Fprintln(w, "  install-skill [name]  install bundled Codex skills")
	fmt.Fprintln(w, "  init-project <name>   create an empty Narr project")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "Global options:")
	fmt.Fprintln(w, "  --project DIR         project root containing narr.toml")
	fmt.Fprintln(w, "  --json                JSON output")
	fmt.Fprintln(w, "  --verbose             verbose output")
	fmt.Fprintln(w, "  --no-color            disable color")
	fmt.Fprintln(w, "  --version             print version")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "Build options:")
	fmt.Fprintln(w, "  --all                 build every chapter in project order")
	fmt.Fprintln(w, "  --out-dir DIR         write build artifacts under DIR")
	fmt.Fprintln(w, "  --dry-run             check build target without writing")
	fmt.Fprintln(w, "  --llm                 explicitly use the default Narr-like LLM output")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "Install-skill options:")
	fmt.Fprintln(w, "  --list                list bundled skills")
	fmt.Fprintln(w, "  --all                 install all bundled skills")
	fmt.Fprintln(w, "  --global              install to ${CODEX_HOME:-$HOME/.codex}/skills")
	fmt.Fprintln(w, "  --local               install to ./skills (default)")
	fmt.Fprintln(w, "  --force               replace an existing installed skill")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "Init-project options:")
	fmt.Fprintln(w, "  --dir DIR             create the project in DIR")
}
