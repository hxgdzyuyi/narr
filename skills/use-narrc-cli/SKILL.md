---
name: use-narrc-cli
description: 当用户需要使用 `narrc` 命令工具时使用：查看帮助、初始化项目、安装内置 skill、lint、test、build、info、query、处理命令参数、解释 CLI 输出或诊断、设计命令行工作流。不要用此 skill 生成 `.narr` 或 `.test.narr` 内容；写 Narr 源文件时使用 `use-narr-lang`。
---

# Use Narrc CLI

## 概览

引导用户正确运行 Go 版 `narrc` 命令工具，并解释命令输出。此 skill 专注 CLI 行为；Narr 语言建模、`.narr`、`.test.narr` 写作规则交给 `use-narr-lang`。

## 起手检查

- 命令参数有疑问时，优先运行 `narrc --help` 或本仓库内的 `go run ./cmd/narrc --help`。
- 运行工程命令前，确认目标目录包含 `narr.toml`；从其他目录运行时使用 `--project <dir>`。
- 解释失败时保留诊断码、文件路径和位置，给出最小可重跑命令。
- 如果用户要创建或修改 Narr 源文件，切换到 `use-narr-lang` 的规则，再回到本 skill 跑验证命令。

## 引用路由

- 处理命令、参数、输出格式、初始化工程和安装 skill 时，读取 `references/cli.md`。
- 需要解释 `.narr` 或 `.test.narr` 语法时，不要猜测；使用 `use-narr-lang`。

## 常用工作流

查看工具能力：

```bash
narrc --help
narrc --version
narrc install-skill --list
```

创建新项目：

```bash
narrc init-project 长夜之城
narrc init-project "My Novel" --dir my-novel
```

在已有项目中验证：

```bash
narrc lint --project <project-dir>
narrc test --project <project-dir> --all
narrc build --project <project-dir> <chapter_code> --dry-run
narrc build --project <project-dir> --all --dry-run
```

安装内置 skill：

```bash
narrc install-skill --list
narrc install-skill --all
narrc install-skill use-narr-lang --global
```

## 仓库内验证

如果改动 `narrc` Go 代码，运行：

```bash
gofmt <changed-go-files>
go test ./...
go run ./cmd/narrc --version
go run ./cmd/narrc lint --project examples/红楼梦
```
