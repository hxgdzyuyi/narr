# Narr / narrc

Narr 是一门把小说写作变成代码的形式化语言；`narrc` 是它的 Go 版命令行工具。

Narr 当前的目标，是将小说写作转化为一种**可编程**、**可测试**的创作流程，从而为小说提供类似软件开发中的 feedback testing 机制。借此，AI 小说创作将进入 **Ralph Loop** 时代。

Narr 会将小说拆分为“剧情”和“文笔”两个层面：

剧情负责结构、逻辑与可测试性；

而最终的文笔表达，则由 LLM 生成。

语言规范以 [docs/syntax.md](docs/syntax.md) 为准；

## 快速开始

从源码运行：

```bash
go run ./cmd/narrc init-project 长夜之城
go run ./cmd/narrc lint --project 长夜之城
```

构建二进制后运行：

```bash
go build -o narrc ./cmd/narrc
./narrc init-project 长夜之城
./narrc lint --project 长夜之城
```

`init-project` 会创建一个空 Narr 工程，并把二进制内嵌的 Codex skills 安装到项目内：

- `use-narrc-cli`：引导使用 `narrc` 命令工具、解释输出和诊断。
- `use-narr-lang`：引导在项目内编写 `.narr` 和 `.test.narr`，并内置完整语法参考。

也可以把内置 skills 安装到当前目录或全局 Codex skills 目录。本地安装时，真实文件位于 `./skills/`，并额外在 `./.agents/skills/` 与 `./.claude/skills/` 下创建软链接：

```bash
narrc install-skill --list
narrc install-skill --all
narrc install-skill use-narr-lang --global
```

## 一、Narr 项目：通过结构蓝图生成长篇小说

长篇小说的问题通常不在单章，而在跨章节一致性：人物知道什么、伏笔是否兑现、主线何时推进、地点与世界状态是否冲突、章节是否偏离全书结构。Narr 用结构化文件描述这些信息，让工具在写作前先提供 feedback 测试，并生成可靠的单章上下文。

在 Narr 中，小说被拆成两部分：

- 剧情：由 `.narr` 和 `.test.narr` 描述，是可编译、可查询、可测试的代码。
- 文笔：由作者或 LLM 根据 `narrc build` 的上下文生成，是最终给读者阅读的散文表达。

这意味着 AI 不再独自决定故事结构。AI 负责在明确约束下生成文笔；Narr 和 `narrc` 负责让剧情结构能被检查、被反馈、被迭代。

Narr 项目分为两层：

- 全局蓝图层：`novel`、`volume`、`thread`、`promise`、`arc`、`invariant`、`start_pattern`、角色、地点、势力、物件、事实。
- 局部章节层：`chapter`、`beat`、`effect`。章节由一组有顺序的 beat 组成，beat 负责推动状态变化、设置或兑现伏笔、推进线索和人物弧线。

`narrc` 会基于这些声明派生出章节关系、活跃线索、活跃伏笔、章节起止状态和测试结果。生成正文时，你不需要让模型重新猜全书设定，而是把当前章节需要继承和完成的内容交给它。

### 推荐工作流

1. 用 `narrc init-project <小说名>` 新建一个 Narr 项目，自动生成 `narr.toml`、`main.narr`、`AGENTS.md` 和本地 `skills/`。
2. 在 `world/`、`characters/`、`structure/`、`threads/`、`promises/`、`arcs/` 等目录里维护蓝图。
3. 用 `novel` 和 `volume` 定义全书规模、卷结构、摘要和文风提示。
4. 用 `chapter` 定义每章标题、目的、字数、摘要、视角、地点和 beat 顺序。
5. 用 `beat` 和 `effect` 描述章内必须发生的状态变化。
6. 用 `promise`、`thread`、`arc` 管理伏笔、线索和人物弧线。
7. 用 `.test.narr` 写项目自己的 feedback 测试，例如“关键章必须有 alias”“某条主线在一卷内至少推进 N 次”。
8. 反复运行 `lint` 和 `test`，根据反馈修改剧情蓝图。
9. 对目标章节运行 `narrc build`，得到单章写作上下文，再交给作者或 LLM 写正文。

一个项目通常长这样：

```text
my-novel/
  AGENTS.md
  narr.toml
  main.narr
  skills/
    use-narrc-cli/
    use-narr-lang/
  .agents/
    skills/
      use-narrc-cli -> ../../skills/use-narrc-cli
      use-narr-lang -> ../../skills/use-narr-lang
  .claude/
    skills/
      use-narrc-cli -> ../../skills/use-narrc-cli
      use-narr-lang -> ../../skills/use-narr-lang
  world/
    places.narr
    facts.narr
  characters/
    main.narr
  structure/
    novel.narr
    volumes.narr
    chapters.narr
    beats_vol01_ch01.narr
  threads/
    main.narr
  promises/
    main.narr
  arcs/
    main.narr
  tests/
    structure.test.narr
```

最小配置：

```toml
[project]
name = "长夜之城"
version = "0.3.5"
language = "zh-CN"
main = "main.narr"
```

全书蓝图示例：

```narr
namespace 长夜之城.structure

novel 长夜之城 {
  title: "长夜之城"
  language: zh-CN
  summary: "少年在王都暗潮中发现旧王血秘密，并被卷入城邦、宗门与旧神遗迹的长期冲突。"

  length:
    volumes = 3
    chapters_per_volume = 40
    chapter = 3000 字

  prose_hint: "正文以冷静、细密的第三人称叙述展开，重视权力关系、场面调度和人物心理。"
}
```

章节与 beat 示例：

```narr
namespace 长夜之城.structure

import 长夜之城.world as world
import 长夜之城.characters as chars
import 长夜之城.promises as promises

chapter vol01.ch01 alias 城门异火 {
  title: "夜入王都，火印初醒"
  purpose: entry
  target_length: 3000 字
  summary: "沈夜抵达王都南门，火印在守城石前异常发热，引出旧王血伏笔。"
  pov: chars.沈夜
  location: world.王都南门
  beats: [
    沈夜抵达城门,
    火印异常
  ]
}

beat 火印异常 @ vol01.ch01 {
  effect:
    chars.沈夜.知道 += world.火印会响应王都石
    chars.沈夜.状态 = 被城门校尉注意

  location: world.王都南门
  pov: chars.沈夜
  on_screen: true
  sets_up: promises.旧王血伏笔
  render_hint: "不要解释火印来源，只写沈夜身体反应与守门者的警觉。"
}
```

生成章节上下文：

```bash
go run ./cmd/narrc build vol01.ch01 --project examples/红楼梦 --out-dir build
go run ./cmd/narrc build --all --project examples/红楼梦 --out-dir build
```

默认输出类似：

```text
build/vol01.ch01.build.narr
```

这个文件是给作者或 LLM 使用的写作上下文，通常包含：

- `chapter`：章节标题、目的、目标字数、前后章节。
- `summary`：全书、卷、本章摘要。
- `context`：本章相关角色、地点、物件和事实。
- `state`：章节开始状态、本章预期变化、章节结束状态。
- `structure`：活跃线索、活跃伏笔、活跃弧线，以及本章实际服务的结构关系。
- `beats`：本章必须按顺序展开的 beat、前置条件、状态效果和呈现提示。
- `prose`：目标长度和文风提示。

拿到 build 文件后，可以把它作为写作约束交给 LLM，例如：

```text
请根据下面的 narrc build 上下文撰写 vol01.ch01 正文。
要求：遵守 ordered_beats 顺序、完成 effect 中的状态变化、不要新增未出现的设定、正文不要输出 Narr 标签。
```

正文建议保存为 `.md` 或其他散文文件；它不是 Narr 编译对象。Narr 源文件继续负责结构，散文文件负责最终文本。每写完一章，再继续构建下一章，循环执行“蓝图调整 -> lint/test -> build -> 正文扩写”。

可以参考内置示例：

```bash
go run ./cmd/narrc lint --project examples/红楼梦
go run ./cmd/narrc test --all --project examples/红楼梦
go run ./cmd/narrc build vol01.ch01 --project examples/红楼梦 --out-dir build
go run ./cmd/narrc info vol01.ch01 --project examples/红楼梦
go run ./cmd/narrc query 'active_threads(structure.vol01.ch01)' --project examples/红楼梦
```

## 二、编译与代码使用

### 环境要求

- Go 1.22 或更新版本。
- 依赖由 Go module 管理，当前模块名为 `narr`。

### 编译与测试

```bash
go test ./...
go run ./cmd/narrc --version
go build -o narrc ./cmd/narrc
```

项目约定：

- Go 代码修改后运行 `gofmt`。
- Go 变更交付前运行 `go test ./...`。
- 最小 CLI 烟测：`go run ./cmd/narrc --version`。
- 项目加载路径烟测：`go run ./cmd/narrc lint --project examples/红楼梦`。
- 不要把生成的 build 输出纳入源码管理。

### CLI 命令

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
```

全局选项：

```text
--project DIR   project root containing narr.toml
--json          JSON output
--verbose       verbose output where supported
--no-color      disable color
--version       print version
```

`build` 选项：

```text
--out-dir DIR   write build artifacts under DIR
--all           build every chapter in project order
--dry-run       check build target without writing
--llm           explicitly use the default Narr-like LLM output
```

`install-skill` 选项：

```text
--list      list bundled skills
--all       install all bundled skills
--global    install to ${CODEX_HOME:-$HOME/.codex}/skills
--local     install to ./skills (default)
--force     replace an existing installed skill
```

本地安装会把真实 skill 写到 `./skills/<name>`，并在 `./.agents/skills/<name>` 与 `./.claude/skills/<name>` 创建指向真实目录的软链接；全局安装只写 `${CODEX_HOME:-$HOME/.codex}/skills`。

`init-project` 选项：

```text
--dir DIR   create the project in DIR
```

常用命令：

```bash
# 创建空 Narr 工程，并自动安装内置 skills
go run ./cmd/narrc init-project 长夜之城
go run ./cmd/narrc init-project "My Novel" --dir my-novel

# 查看和安装二进制内嵌 skills
go run ./cmd/narrc install-skill --list
go run ./cmd/narrc install-skill --all
go run ./cmd/narrc install-skill use-narr-lang --global

# 检查项目
go run ./cmd/narrc lint --project examples/红楼梦

# 运行全部 .test.narr
go run ./cmd/narrc test --all --project examples/红楼梦

# 生成默认 LLM 写作上下文
go run ./cmd/narrc build vol01.ch01 --project examples/红楼梦 --out-dir build
go run ./cmd/narrc build --all --project examples/红楼梦 --out-dir build

# 只检查构建目标，不写文件
go run ./cmd/narrc build vol01.ch01 --project examples/红楼梦 --dry-run
go run ./cmd/narrc build --all --project examples/红楼梦 --dry-run

# 输出 JSON 到 stdout
go run ./cmd/narrc --json build vol01.ch01 --project examples/红楼梦

# 查看单章 build / 状态摘要
go run ./cmd/narrc info vol01.ch01 --project examples/红楼梦

# 查询派生视图
go run ./cmd/narrc query 'active_threads(structure.vol01.ch01)' --project examples/红楼梦
go run ./cmd/narrc --json query 'count(served_promises(structure.vol01.ch01))' --project examples/红楼梦
```

### 代码结构

```text
cmd/narrc/main.go        CLI 入口
bundled_skills.go        将顶层 skills/ 内嵌到 narrc 二进制
internal/cli/            CLI 参数解析与命令分发
internal/project/        项目发现、narr.toml 加载、文件收集
internal/source/         诊断、源码位置、错误报告
internal/lexer/          Narr 词法分析
internal/parser/         Narr / .test.narr 解析
internal/ast/            语法树
internal/resolve/        namespace、import 与引用解析
internal/check/          语义检查
internal/model/          规范化后的小说模型
internal/state/          状态时间线与查询基础
internal/structure/      派生结构视图与查询求值
internal/tester/         .test.narr 测试执行
internal/build/          单章 build 生成与写出
internal/format/         文本与 JSON 输出
skills/use-narrc-cli/    narrc 命令工具使用指南
skills/use-narr-lang/    Narr 语言与 .narr / .test.narr 写作指南
```

开发时优先阅读：

- [docs/syntax.md](docs/syntax.md)：Narr 语法与语义来源。
- [docs/cli-docs.md](docs/cli-docs.md)：CLI 行为设计。
- [docs/error-codes.md](docs/error-codes.md)：诊断码目录。
- [internal/parser/coverage.md](internal/parser/coverage.md)：语法覆盖矩阵。
