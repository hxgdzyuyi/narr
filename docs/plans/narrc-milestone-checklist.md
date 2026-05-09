# narrc 里程碑完成情况

本 checklist 对应 [narrc Go 实现计划](./narrc-go-implementation-plan.md) 中的 M1-M10。状态更新只记录实际完成情况，不用来替代实现计划本身。

状态约定：

```text
[ ] 未开始
[-] 进行中
[x] 已完成
```

## 总览

| 里程碑 | 状态 | 目标 |
| --- | --- | --- |
| M1 | [x] | Go 工程骨架 |
| M2 | [x] | Lexer / Parser |
| M3 | [x] | Symbol / Resolve |
| M4 | [x] | Model / Type Check |
| M5 | [x] | State Timeline |
| M6 | [x] | Structure Check / Derive |
| M7 | [x] | Eval / Query |
| M8 | [x] | Build / Info |
| M9 | [x] | Test Runner |
| M10 | [x] | 稳定性与文档 |

---

## M1: Go 工程骨架

状态：`[x]`

完成项：

- [x] 创建 `go.mod`。
- [x] 创建 `cmd/narrc/main.go`。
- [x] 实现全局 flag 和子命令分发。
- [x] 实现 `--project` 工程发现。
- [x] 实现 `narr.toml` 读取与基础校验。

验收：

- [x] `go run ./cmd/narrc --version` 可运行。
- [x] `go run ./cmd/narrc lint --project examples/红楼梦` 能进入加载流程。

---

## M2: Lexer / Parser

状态：`[x]`

完成项：

- [x] 实现 Unicode lexer。
- [x] 建立 `internal/parser/coverage.md`，逐条映射 `docs/syntax.md` 的 EBNF production。
- [x] 实现 `.narr` 顶层声明解析。
- [x] 实现 `.test.narr` 顶层 test 解析。
- [x] 实现字段专用 parser：length / condition / effect / starts / test statement。
- [x] 实现表达式 parser 与优先级。
- [x] 每个 EBNF production 至少有一个正例测试。
- [x] 所有 AST 节点保留 source span。
- [x] 解析错误可恢复到下一个顶层声明或 test。

验收：

- [x] `internal/parser/coverage.md` 覆盖 `docs/syntax.md` 当前所有 EBNF 组。
- [x] 能解析 `examples/红楼梦` 所有 `.narr` 文件。
- [x] 能解析 `examples/红楼梦` 所有 `.test.narr` 文件。
- [x] 语法错误诊断包含文件、行、列。

---

## M3: Symbol / Resolve

状态：`[x]`

完成项：

- [x] 建立 namespace symbol table。
- [x] 支持同一 namespace 跨文件合并。
- [x] 实现 import alias 解析。
- [x] 实现 chapter_code 解析、归一化、查重。
- [x] 实现 ref / path_expr 基础解析。
- [x] 实现 `.test.narr` 局部变量优先级。
- [x] 实现 `narrc query` 的 main namespace/imports 命名环境。

验收：

- [x] 能解析 `examples/红楼梦` 所有跨 namespace 引用。
- [x] 重复声明能报错。
- [x] 未知引用能报错。

---

## M4: Model / Type Check

状态：`[x]`

完成项：

- [x] AST 转 Model。
- [x] 实现 enum。
- [x] 实现 class field。
- [x] 实现 place / character / collective / faction / object / fact。
- [x] 实现结构类型：Novel / Volume / Chapter / Beat / Thread / Promise / Arc / StartPattern / Invariant。
- [x] 实现 Symbol 宽松求值策略。
- [x] 实现 effect 目标字段检查。
- [x] 实现 effect 值类型兼容检查。

验收：

- [x] `examples/红楼梦` 类型检查通过。
- [x] 负例 `effect unknown field` 失败。

---

## M5: State Timeline

状态：`[x]`

完成项：

- [x] 构建初始状态。
- [x] 按 chapter_code 排序 chapter。
- [x] 按 `chapter.beats` 排序 beat。
- [x] 应用 assignment。
- [x] 应用 set add/remove。
- [x] 应用 list append。
- [x] 保存 chapter / beat 边界 checkpoint。
- [x] 实现 `state(field, at chapter.begin/end)`。

验收：

- [x] 能求出 `examples/红楼梦` 第一回结尾状态。
- [x] 能求出 `examples/红楼梦` 第二回结尾状态。

---

## M6: Structure Check / Derive

状态：`[x]`

完成项：

- [x] 实现 StaticStructureCheck。
- [x] 实现 StateDependentCheck。
- [x] 检查 beat.precondition。
- [x] 检查 start_pattern.requires。
- [x] 检查 thread / promise / arc 起点匹配 start_pattern.at。
- [x] 检查 promise payoff 不早于 setup。
- [x] 检查 arc state_field 与 states。
- [x] 检查 hidden invariant。
- [x] 检查 always invariant。
- [x] 实现 active_threads / active_promises / active_arcs。
- [x] 实现 served_threads / served_promises / served_arcs。
- [x] 实现 reveals_in / mentions_in。

验收：

- [x] `go run ./cmd/narrc lint --project examples/红楼梦` 通过。
- [x] active 视图查询返回预期。
- [x] served 视图查询返回预期。

---

## M7: Eval / Query

状态：`[x]`

完成项：

- [x] 实现 EvalEnv。
- [x] 实现 literal/ref/path 求值。
- [x] 实现 comparison / membership。
- [x] 实现 existence predicate。
- [x] 实现 temporal predicate。
- [x] 实现 narrative predicate。
- [x] 实现 `count(query)`。
- [x] 实现 `count(binder where expr)`。
- [x] 实现 `collect(value from binder where expr)`。
- [x] 实现内置函数。
- [x] 实现 `narrc query` 文本输出。
- [x] 实现 `narrc query --json` 输出。

验收：

- [x] `narrc query 'active_threads(structure.vol01.ch01)' --project examples/红楼梦` 返回结果。
- [x] `narrc query 'count(chapter 章节 in chapters_in(structure.vol01) where 章节 serves promises.通灵宝玉来历伏笔)' --project examples/红楼梦` 返回结果。

---

## M8: Build / Info

状态：`[x]`

完成项：

- [x] 实现 ChapterBuild struct。
- [x] 实现 build context 推导。
- [x] 实现 build state 字段。
- [x] 实现 build structure 字段。
- [x] 实现 build beats 字段。
- [x] 实现 JSON writer。
- [x] 实现 `--out-dir`。
- [x] 实现 `--dry-run`。
- [x] 实现 `narrc info` 文本输出。

验收：

- [x] `narrc build vol01.ch01 --project examples/红楼梦 --out-dir build` 生成 JSON。
- [x] `narrc build vol01.ch01 --project examples/红楼梦 --json` 输出 JSON。
- [x] `narrc info vol01.ch01 --project examples/红楼梦` 输出摘要。

---

## M9: Test Runner

状态：`[x]`

完成项：

- [x] 实现 test 文件选择：`--all`。
- [x] 实现 test 文件选择：显式 files。
- [x] 实现 assert。
- [x] 实现 let。
- [x] 实现 forall。
- [x] 实现 exists 无 block。
- [x] 实现 exists 有 block。
- [x] 实现 failure bindings。
- [x] 实现文本报告。
- [x] 实现 JSON 报告。

验收：

- [x] `narrc test --all --project examples/红楼梦` 通过。
- [x] 失败测试报告包含 test 名、source span、失败表达式和绑定值。

---

## M10: 稳定性与文档

状态：`[x]`

完成项：

- [x] 补全 error codes。
- [x] 补全 README 用法。
- [x] 加入 `go test ./...`。
- [x] 加入 `go vet ./...`。
- [x] 加入 `examples/红楼梦` golden JSON。
- [x] 加入 `examples/红楼梦` info golden text。
- [x] 加入 invalid testdata。
- [x] 加入 CI。

验收：

- [x] `go test ./...` 通过。
- [x] `go vet ./...` 通过。
- [x] CI 通过。
