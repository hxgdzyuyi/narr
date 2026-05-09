# narrc CLI

在本仓库中使用本地 Go 入口运行命令：

```bash
go run ./cmd/narrc --help
go run ./cmd/narrc --version
```

当前命令形态：

```text
Usage: narrc [global options] <command> [options] <args>

Commands:
  build <chapter_code>|--all  build chapter artifact files
  test [files|--all]    run .test.narr declarations
  lint [files]          load and check a Narr project
  info <chapter_code>   show chapter build/state summary
  query <expr>          evaluate a query expression
  install-skill [name]  install bundled Codex skills
  init-project <name>   create an empty Narr project

Global options:
  --project DIR         project root containing narr.toml
  --json                JSON output
  --verbose             verbose output
  --no-color            disable color
  --version             print version

Build options:
  --all                 build every chapter in project order
  --out-dir DIR         write build artifacts under DIR
  --dry-run             check build target without writing
  --llm                 explicitly use the default Narr-like LLM output

Install-skill options:
  --list                list bundled skills
  --all                 install all bundled skills
  --global              install to ${CODEX_HOME:-$HOME/.codex}/skills
  --local               install to ./skills (default)
  --force               replace an existing installed skill

Init-project options:
  --dir DIR             create the project in DIR
```

## 常用配方

当前目录不是工程根目录时，显式传入 `--project`：

```bash
go run ./cmd/narrc lint --project examples/红楼梦
go run ./cmd/narrc test --project examples/红楼梦 --all
go run ./cmd/narrc build --project examples/红楼梦 vol01.ch01 --dry-run
go run ./cmd/narrc build --project examples/红楼梦 --all --dry-run
go run ./cmd/narrc info --project examples/红楼梦 vol01.ch01
go run ./cmd/narrc query --project examples/红楼梦 'active_threads(s.vol01.ch01)'
```

构建输出：

```bash
go run ./cmd/narrc build --project examples/红楼梦 vol01.ch01 --out-dir build
go run ./cmd/narrc build --project examples/红楼梦 --all --out-dir build
go run ./cmd/narrc build --project examples/红楼梦 vol01.ch01 --llm --out-dir build
go run ./cmd/narrc build --project examples/红楼梦 vol01.ch01 --json
```

安装二进制内嵌的 skill：

```bash
go run ./cmd/narrc install-skill --list
go run ./cmd/narrc install-skill use-narrc-cli
go run ./cmd/narrc install-skill use-narr-lang --global
go run ./cmd/narrc install-skill --all --force
```

本地安装会把真实 skill 写到 `./skills/<name>`，并在 `./.agents/skills/<name>` 与 `./.claude/skills/<name>` 创建指向真实目录的软链接。全局安装只写 `${CODEX_HOME:-$HOME/.codex}/skills`。

创建空工程并自动安装内置 skill：

```bash
go run ./cmd/narrc init-project 长夜之城
go run ./cmd/narrc init-project "My Novel" --dir my-novel
```

## 处理诊断

命令失败时：

- 解释中保留诊断码和源码位置。
- 优先修复底层 `.narr` 或 `.test.narr`，不要削弱检查。
- 修复后重跑覆盖失败路径的最小命令。
- 下游工具需要结构化结果时，使用 `--json`。

改动 Go 实现时，运行 `AGENTS.md` 要求的检查：

```bash
gofmt <changed-go-files>
go test ./...
go run ./cmd/narrc --version
go run ./cmd/narrc lint --project examples/红楼梦
```
