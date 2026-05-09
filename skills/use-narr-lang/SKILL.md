---
name: use-narr-lang
description: 当用户需要在 Narr 项目内创建、修改、解释或审查 `.narr`、`.test.narr`、`narr.toml` 文件时使用：构建小说蓝图、章节、beat、effect、实体、thread、promise、arc、invariant，编写项目特异设计测试，使用派生视图和查询谓词，或按完整 `syntax.md` 判断 Narr 语法是否合法。
---

# Use Narr Lang

## 概览

引导用户用 Narr 语言构建小说蓝图和项目设计测试，不要臆造语法。此 skill 内置 `references/syntax.md`，它是从仓库 `docs/syntax.md` 复制的完整语法事实源。

## 起手检查

- 先查看当前项目的 `narr.toml`、namespace、import、目录结构和相邻 `.narr` / `.test.narr` 示例。
- 保持现有命名、namespace 和 import 风格。
- 不添加 `references/syntax.md` 未记录的结构。
- 只在 `beat.effect` 中改变叙事状态；`.test.narr` 只能表达约束和查询。
- 需要运行命令验证时，使用 `use-narrc-cli` 的 CLI 工作流。

## 引用路由

- 写 `.narr`、`narr.toml`、章节、beat、effect、实体和结构声明时，先读 `references/language.md`。
- 写 `.test.narr`、测试断言、派生视图、叙事谓词或 `narrc query` 表达式时，先读 `references/tests-and-query.md`。
- 有任何语法边界、字段合法性或完整规则疑问时，读 `references/syntax.md`。

## `.narr` 工作流

1. 定位目标 namespace 和需要 import 的命名空间。
2. 只使用文档中的顶层声明：`novel`、`volume`、`chapter`、`beat`、`promise`、`thread`、`arc`、`invariant`、实体等。
3. 让 chapter 持有 `beats: [...]` 顺序；顶层 beat 必须 `@` 到所属 chapter。
4. 用显式字段和 `effect` 表达结构关系，不依赖散文语义猜测。
5. 修改后建议运行 `narrc lint --project <project-dir>`。

## `.test.narr` 工作流

1. 测试文件只包含 `namespace`、`import`、`test`。
2. 写项目特异约束，例如节奏、密度、伏笔间隔、关键章元信息；不要重复编译器默认检查。
3. 使用 `chapters_in`、`beats`、`served_promises`、`state(...)`、`count(...)` 等派生视图和聚合。
4. 断言必须求值为 Bool，不写 `effect`。
5. 修改后建议运行 `narrc test --project <project-dir> --all` 或目标测试文件。

## 边界

Narr 不负责判断文本是否精彩、优美、震撼或深刻。用 `.narr` 表达结构骨架与状态变化，用 `.test.narr` 表达可计算的项目约束，散文生成交给 build 输出后的写作流程。
