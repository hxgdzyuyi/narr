# narrc Go 实现计划

目标：用 Go 尽可能完整实现 `narrc`，覆盖 `docs/syntax.md` 精简语法与 `docs/cli-docs.md` 中定义的命令：

```text
narrc build <chapter_code>
narrc test [files|--all]
narrc lint [files]
narrc info <chapter_code>
narrc query <expr>
```

实现范围以当前精简语法为准，不兼容旧语法扩展，例如 `snapshot`、`fixture`、`group`、`trigger`、`effective_at`、`first/last/min/max`、自定义函数、可配置 lint 矩阵。

语法实现必须以 `docs/syntax.md` 中的 EBNF 片段为准。本文档只规划实现顺序和工程结构；当实现细节与 `docs/syntax.md` 的 EBNF 不一致时，以 `docs/syntax.md` 为规范来源，并同步修正文档或实现。

---

## 1. 交付物

```text
go.mod
cmd/narrc/main.go
internal/...
```

生成的二进制：

```bash
go build ./cmd/narrc
```

验证命令：

```bash
go test ./...
go run ./cmd/narrc lint examples/红楼梦
go run ./cmd/narrc build vol01.ch01 --project examples/红楼梦
go run ./cmd/narrc info vol01.ch01 --project examples/红楼梦
go run ./cmd/narrc test --all --project examples/红楼梦
go run ./cmd/narrc query 'active_threads(structure.vol01.ch01)' --project examples/红楼梦
```

---

## 2. 设计原则

1. 先实现确定性编译管线，再实现 CLI 包装。
2. 解析器采用手写 lexer + recursive descent parser，逐条覆盖 `docs/syntax.md` 的 EBNF，避免生成器引入额外构建步骤。
3. `.narr` 与 `.test.narr` 使用同一 lexer，不同顶层 parser mode。
4. 语义模型与 AST 分离：AST 保留源位置，Model 是解析并解析引用后的规范化结构。
5. 所有诊断都带 code、severity、file、line、column、message、hint。
6. 查询、测试、build 复用同一个表达式求值器和派生视图引擎。
7. `build` 输出 JSON 是稳定结构，字段顺序通过 struct 定义固定。
8. Go 依赖尽量少；TOML 可使用 `github.com/pelletier/go-toml/v2`，CLI 可先用标准库实现。

---

## 3. 语法契约

`docs/syntax.md` 是语法唯一来源。实现时必须维护一个 EBNF 覆盖矩阵，用来追踪每条语法产生式对应的 parser 函数、测试文件和完成状态。

覆盖矩阵建议放在：

```text
internal/parser/coverage.md
```

矩阵格式：

```text
syntax.md section | EBNF production | parser function | positive tests | negative tests | status
```

必须覆盖的 EBNF 组：

```text
source_file / narr_file / test_file
namespace_decl / import_decl
narr_declaration
基础 literal / list_expr / set_expr
chapter_code / volume_code
novel_decl / volume_decl / chapter_decl
beat_decl / effect_block / effect_stmt
time_ref / anchor_ref
start_pattern_decl
promise_decl / thread_decl / arc_decl
invariant_decl
enum_decl / class_decl / entity declarations
test_decl / test_stmt / binder
query_expr / collection_query / relation_query / projection_query
expr / predicate_expr / value_expr
existence / temporal / aggregate / state / function_call / narrative predicates
```

Parser 实现要求：

```text
每个 EBNF production 对应一个明确的 parse 函数或小函数组。
每个 parse 函数有至少一个正例测试。
有歧义或边界条件的 production 必须有负例测试。
实现不得引入 syntax.md 未定义的语法。
如果实现需要调整语法，先改 syntax.md，再更新 parser 和覆盖矩阵。
```

---

## 4. 目录结构

```text
cmd/narrc/
  main.go

internal/cli/
  cli.go
  build.go
  test.go
  lint.go
  info.go
  query.go
  flags.go

internal/project/
  config.go
  discover.go
  files.go
  load.go

internal/source/
  file.go
  span.go
  diagnostic.go
  reporter.go

internal/lexer/
  token.go
  lexer.go
  lexer_test.go

internal/parser/
  parser.go
  narr_file.go
  test_file.go
  decl.go
  block.go
  expr.go
  parser_test.go

internal/ast/
  file.go
  decl.go
  expr.go
  stmt.go
  literal.go

internal/model/
  project.go
  names.go
  entities.go
  structure.go
  chapter_code.go
  types.go

internal/resolve/
  symbols.go
  imports.go
  refs.go
  namespace.go

internal/check/
  checker.go
  file_rules.go
  type_rules.go
  structure_rules.go
  test_rules.go

internal/state/
  value.go
  store.go
  effect.go
  timeline.go
  query.go

internal/derive/
  views.go
  active.go
  served.go
  build.go

internal/eval/
  env.go
  expr.go
  predicates.go
  functions.go
  query.go

internal/tester/
  runner.go
  result.go
  bindings.go

internal/build/
  chapter_build.go
  json.go
  writer.go

internal/format/
  text.go
  json.go
```

---

## 5. CLI 行为

### 5.1 全局选项

```text
--project DIR    工程根目录。默认从当前目录向上寻找 narr.toml。
--json           JSON 输出。
--verbose        输出解析、检查、求值过程。
--no-color       禁用颜色。
--version        输出版本。
```

### 5.2 `lint [files]`

行为：

```text
1. 加载 project。
2. 始终解析并检查整个 project，保证跨文件约束可验证。
3. 如果传入 files / dirs，只过滤输出这些路径相关的 diagnostics。
4. dirs 递归展开；shell glob 由 shell 展开后作为 files 处理。
5. 执行语法、名称解析、类型、静态结构、状态相关结构检查。
6. 输出 diagnostics。
7. 有 error 时 exit 1，否则 exit 0。
```

输出：

```text
PASS examples/红楼梦/structure/chapters.narr
ERROR [E0403] examples/.../chapters.narr:12:9 chapter code vol01.chX 不合法
```

JSON 输出：

```json
{
  "ok": false,
  "diagnostics": [
    {
      "severity": "error",
      "code": "E0403",
      "file": "structure/chapters.narr",
      "line": 12,
      "column": 9,
      "message": "chapter code vol01.chX 不合法"
    }
  ]
}
```

### 5.3 `build <chapter_code>`

选项：

```text
--out-dir DIR    默认 build
--dry-run        只检查，不写文件
--json           输出 build JSON 到 stdout，不写文件
```

行为：

```text
1. 完整加载并检查 project。
2. 解析 chapter_code，归一化。
3. 计算 chapter build。
4. dry-run：只输出摘要。
5. json：输出 JSON 到 stdout。
6. 默认写入 <out-dir>/<canonical_code>.build.narr；`--json` 输出 JSON 到 stdout。
```

### 5.4 `info <chapter_code>`

行为：

```text
1. 完整加载并检查 project。
2. 计算目标 chapter 的 build。
3. 输出人工可读摘要：chapter、beats、active/served 结构、状态摘要。
```

### 5.5 `query <expr>`

选项：

```text
--namespace NS   指定 query 的当前 namespace。
```

行为：

```text
1. 完整加载并检查 project。
2. 使用表达式 parser 解析命令行 expr。
3. 建立 query eval env。
4. 在 project eval env 中求值。
5. 输出 result。
6. 如果表达式内部含 binder/count/collect，verbose 输出匹配绑定。
```

query 命名环境：

```text
如果指定 --namespace：
  CurrentNamespace = NS。
  Imports = NS 下所有文件声明的 import alias 合并结果。

如果未指定 --namespace：
  优先读取 narr.toml 的 project.main 文件。
  CurrentNamespace = main.narr 的 namespace。
  Imports = main.narr 中声明的 import alias。

如果 main.narr 缺失或没有 import：
  CurrentNamespace = project.name 对应 namespace。
  Imports = 空。

命令行表达式使用虚拟 source file "<query>"，但名称解析规则与 .test.narr 表达式一致。
```

### 5.6 `test [files|--all]`

行为：

```text
1. 完整加载并检查 project。
2. 选择测试文件：--all 加载全部 .test.narr；否则使用显式 files。
3. 执行 test 声明。
4. 输出 pass/fail。
5. 有失败时 exit 1。
```

---

## 6. 编译管线

```text
DiscoverProject
  -> LoadConfig
  -> CollectFiles
  -> Lex
  -> Parse
  -> BuildSymbolTables
  -> ResolveImports
  -> ResolveRefs
  -> TypeCheck
  -> StaticStructureCheck
  -> BuildStateTimeline
  -> StateDependentCheck
  -> DeriveViews
  -> RunCommand
```

关键约束：

```text
lint 也应跑完整管线，除非语法错误阻断后续阶段。
build/info/query/test 不应各自重复实现解析或检查逻辑。
StaticStructureCheck 只做不依赖状态时间线的结构检查。
StateDependentCheck 做 precondition、start_pattern.requires、invariant、state conflict 等检查。
每个阶段只追加 diagnostics，不直接 os.Exit。
CLI 层统一决定输出和 exit code。
```

---

## 7. Project 加载

### 7.1 工程发现

规则：

```text
如果 --project 存在，以该目录为根。
否则如果命令显式传入一个目录参数且该目录包含 narr.toml，以该目录为根。
否则从 cwd 向上寻找 narr.toml。
找不到 narr.toml 报 E0001。
```

### 7.2 文件收集

```text
递归扫描 project root。
忽略 .git、build、vendor、node_modules。
收集 .narr 与 .test.narr。
只把 .narr 纳入结构编译。
test 命令才执行 .test.narr。
```

### 7.3 narr.toml

支持字段：

```toml
[project]
name = "红楼梦"
version = "0.3.5"
language = "zh-CN"
main = "main.narr"
```

检查：

```text
[project].name 必填。
[project].version 必填且兼容 0.3.5。
mode/type/generation 禁止出现。
未知顶层表先 warning，未知 [project] 字段 warning。
```

---

## 8. Lexer

### 8.1 Token 类型

```text
Identifier
String
MultilineString
Integer
Symbol
Keyword
LBrace RBrace
LBracket RBracket
LParen RParen
Colon Comma Dot
Equal EqualEqual BangEqual
Less LessEqual Greater GreaterEqual
PlusEqual MinusEqual
Arrow
Newline
EOF
```

### 8.2 关键词

```text
namespace import as
novel enum class field
volume chapter beat start_pattern
place character collective faction object fact
promise thread arc invariant
title language summary length prose_hint
purpose target_chapters target_length
pov location time_hint beats
precondition effect on_screen observers
sets_up pays_off advances resolves reveals mentions render_hint
at requires starts tags note
setup_at start_pattern setup_strength payoff_by payoff_at payoff_kind question reader_visibility
kind starts_at expected_resolution resolved_at priority
subject state_field initial states
hidden until always active_until
test assert let forall exists where in not and or
true false beginning end_of_story append
```

### 8.3 Unicode 标识符

规则：

```text
中文、英文、数字、下划线可组成 identifier。
identifier 不能以数字开头。
chapter code 在 parser 层识别为 identifier "." identifier。
```

### 8.4 注释

建议支持：

```text
# line comment
// line comment
```

---

## 9. Parser

### 9.1 顶层 parser mode

```text
Narr mode:
  namespace import* narr_declaration*

Test mode:
  namespace import* test_decl*
```

### 9.2 Block 解析

支持：

```narr
chapter vol01.ch01 alias 城门 {
  title: "城门"
}
```

字段块：

```narr
effect:
  chars.沈夜.位置 = world.王都南门
```

实现建议：

```text
lexer 保留 newline。
parser 不使用“扫描到下一个字段名”的通用策略。
每类字段块使用专用 parser：
  parseLengthBlock
  parseConditionBlock
  parseEffectBlock
  parseStartTargetBlock
  parseStatementBlock
  parseListExpr / parseSetExpr
字段专用 parser 明确知道合法 statement 起始 token、终止条件和错误恢复点。
普通 object block 只负责分派字段名，不负责猜测字段块边界。
```

### 9.3 AST 节点

必须保留：

```text
Kind
Name
Fields
Children / Statements
Span
Raw tokens where needed
```

### 9.4 错误恢复

解析失败时：

```text
记录 diagnostic。
跳到下一个顶层 declaration keyword 或 test keyword。
尽可能继续解析同文件后续内容。
```

---

## 10. 名称与引用解析

### 10.1 Symbol Key

```text
namespace + local name
namespace + chapter_code
import alias + local name
```

### 10.2 Namespace 规则

```text
每个文件必须有且只有一个 namespace。
同一 namespace 可跨多个文件。
同一 namespace 内声明名不得重复。
chapter_code 归一化后不得重复。
```

### 10.3 Import 规则

```text
import 红楼梦.world as world
```

检查：

```text
namespace 存在。
alias 不冲突。
不支持 wildcard 和 named import。
```

### 10.4 名称解析顺序

```text
1. 当前 block 内 let / quantifier 局部变量。
2. 当前 namespace 内声明。
3. imported namespace alias。
4. 当前 namespace 中的 chapter_code。
5. imported namespace 中的 chapter_code。
6. 唯一 chapter alias。
7. volume code / volume alias。
8. enum / Symbol 类型上下文。
9. 报错。
```

---

## 11. Model

### 11.1 核心结构

```go
type Project struct {
    Config     Config
    Namespaces map[string]*Namespace
    Novel      *Novel
    Volumes    map[VolumeCode]*Volume
    Chapters   map[ChapterCode]*Chapter
    Beats      map[SymbolID]*Beat
    Threads    map[SymbolID]*Thread
    Promises   map[SymbolID]*Promise
    Arcs       map[SymbolID]*Arc
    Invariants map[SymbolID]*Invariant
    Entities   *Entities
}
```

### 11.2 ChapterCode

```go
type ChapterCode struct {
    VolumePrefix string
    VolumeNumber int
    ChapterPrefix string
    ChapterNumber int
}
```

方法：

```text
ParseChapterCode(raw string)。
Canonical() -> vol01.ch01。
Compare(other)。
VolumeCode()。
```

### 11.3 Values

需要支持：

```text
Null / Missing
Bool
Int
String
Symbol
Ref
List
Set
Object
```

Set 内部用稳定 key 去重，输出时按 key 排序。

---

## 12. 类型系统

### 12.1 内置类型

```text
Bool
Int
String
Symbol
Text
Novel
Volume
Chapter
Beat
Thread
Promise
Arc
StartPattern
Invariant
Place
Character
Object
Faction
Fact
Claim
Set<T>
List<T>
```

结构类型用于 binder、relation query、narrative predicate、build 字段和内部引用检查。即使当前 query collection 不暴露某些声明类型，resolver 和 checker 仍需要能表达它们。

### 12.2 class field

```narr
class 叙事人物 {
  field 位置: Place = world.未明
  field 知道: Set<Fact> = {}
}
```

检查：

```text
字段类型存在。
默认值类型兼容。
character 初始化字段必须存在于 class。
effect 修改字段必须存在且类型兼容。
```

### 12.3 Symbol 宽松策略

规范允许大量未声明 symbol。实现策略：

```text
Symbol 类型字段可接受任意 bare identifier。
enum 类型字段只接受 enum 成员。
没有类型上下文的 bare identifier 先解析 ref；失败则作为 Symbol。
```

---

## 13. 结构检查

实现 `docs/syntax.md` 的必须检查，拆成以下规则组。规则分为两类：

```text
StaticStructureCheck:
  不需要状态时间线即可完成，例如声明合法性、引用解析、类型、章节/beat 归属。

StateDependentCheck:
  需要 BuildStateTimeline 之后才能完成，例如 precondition、start_pattern.requires、
  effect 状态冲突、invariant、state(...) 唯一性。
```

### 13.1 文件规则

```text
.narr 不能包含 test。
.test.narr 不能包含小说结构声明。
禁止 timeline/mode/generation/function。
```

### 13.2 Chapter / Beat

```text
chapter_code 合法、归一化、无重复。
chapter.beats 中 beat 存在且不重复。
顶层 beat 必须有 chapter anchor。
beat anchor 必须指向所属 chapter。
chapter.beats 必须让章内 beat 顺序唯一确定。
```

### 13.3 Start Pattern

```text
at 可解析。
starts 目标存在且类型为 thread / promise / arc。
thread/promise/arc 起点匹配 start_pattern.at。
requires 在 at 位置可满足。此项属于 StateDependentCheck。
```

### 13.4 Promise / Thread / Arc

```text
promise.setup_at 可解析。
promise.payoff_by/payoff_at 可解析。
promise payoff 不早于 setup。
thread starts/resolution anchor 可解析。
arc.subject 是 character。
arc.state_field 存在于 subject class。
arc state effect 值落在 arc.states 内。
```

### 13.5 Effect / State

```text
effect 目标字段存在。
assignment 值类型兼容。
set += / -= 目标是 Set<T>。
append 目标是 List<T>。
precondition 在对应 beat.before 可满足。此项属于 StateDependentCheck。
明显状态冲突报错。此项属于 StateDependentCheck。
```

### 13.6 Invariant

```text
hidden fact until anchor:
  before anchor 的 beat.reveals 不得包含该 fact。

always condition active_until:
  在相关 chapter.begin / chapter.end 求值必须为 true。

两类 invariant 检查均属于 StateDependentCheck。
```

---

## 14. State Timeline

### 14.1 顺序

```text
beginning
chapter.begin
beat.before
beat.after
chapter.end
end_of_story
```

`state(...)` 对外只支持 chapter begin/end；内部检查 precondition 与 start_pattern 可以使用 beat before/after。

### 14.2 初始状态

来源：

```text
class field default。
entity block 初始化字段。
place/object/faction block 初始化字段。
```

### 14.3 应用 effect

按 chapter_code 顺序与 chapter.beats 顺序执行：

```text
assignment 覆盖字段值。
set_add 加入集合。
set_remove 移除集合。
list_append 追加列表。
```

每个边界保存 immutable checkpoint 或 copy-on-write delta。

### 14.4 状态查询

```text
StateAt(field, anchor) -> Value。
如果 anchor 不可解析，报错。
如果 field 不存在，报错。
```

---

## 15. 派生视图

实现：

```text
chapters_in(volume)
beats(chapter)
active_threads(chapter)
active_promises(chapter)
active_arcs(chapter)
served_threads(chapter)
served_promises(chapter)
served_arcs(chapter)
reveals_in(chapter)
mentions_in(chapter)
state(field, at chapter.begin|chapter.end)
build(chapter)
```

### 15.1 active

```text
thread active:
  starts_at <= chapter.end
  and resolved_at missing or resolved_at > chapter.begin

promise active:
  setup_at <= chapter.end
  and payoff_at missing or payoff_at > chapter.begin

arc active:
  starts_at <= chapter.end
  and expected_resolution missing or expected_resolution >= chapter.begin
```

### 15.2 served

按语法规范中的 narrative predicate 语义实现：

```text
chapter serves X:
  chapter.start_pattern starts X
  or any beat explicitly links X
  or arc state_field changed
```

### 15.3 build

`build(chapter)` 复用 ChapterBuild 结构，不重新计算一套逻辑。

---

## 16. 表达式求值

### 16.1 支持表达式

```text
literal
ref
path_expr
query_expr
aggregate_expr count
state_expr
function_call builtin
comparison
membership
exists/missing
temporal predicates
narrative predicates
and/or/not/implication
```

### 16.2 EvalEnv

```go
type Env struct {
    Project *model.Project
    Views   *derive.Views
    State   *state.Timeline
    Vars    map[string]Value
    CurrentNamespace string
    Imports map[string]string
}
```

`Imports` 保存 alias 到 namespace 的映射。`.test.narr` 求值时来自测试文件自身 imports；`narrc query` 求值时来自 query 命名环境。

### 16.3 内置函数

```text
canonical(x)
volume_of(chapter)
chapter_of(beat_or_anchor)
previous(chapter)
next(chapter)
chapter_distance(anchor, anchor)
chapters_between(anchor, anchor)
```

### 16.4 Query

支持：

```text
collection_query
relation_query
collect(value from binder where expr)
count(query)
count(binder where expr)
```

集合输出稳定排序：

```text
chapters 按 chapter_code。
beats 按 chapter.beats。
threads/promises/arcs 按声明顺序。
entities 按声明顺序。
```

---

## 17. Test Runner

### 17.1 Test AST

```text
test name tags block
assert expr message?
let name = expr
forall binder where? block
exists binder where? block?
```

### 17.2 执行语义

```text
assert: expr 必须 true。
let: 当前 test block 内绑定。
forall: 遍历 domain，每个绑定执行子句。
exists 无 block: 至少一个绑定满足 where。
exists 有 block: 至少一个绑定满足 where 且子句全部通过。
```

### 17.3 失败报告

输出：

```text
test 名称。
source span。
失败表达式。
绑定值。
actual / expected。
```

JSON：

```json
{
  "ok": false,
  "tests": [
    {
      "name": "第一回推进士隐悟道与英莲命运",
      "status": "fail",
      "file": "tests/continuity.test.narr",
      "line": 21,
      "message": "state(chars.甄士隐.状态, at 第一回.end) == 出家",
      "bindings": {"第一回": "structure.vol01.ch01"},
      "actual": "寄居困顿",
      "expected": "出家"
    }
  ]
}
```

---

## 18. Chapter Build JSON

Go struct：

```go
type ChapterBuild struct {
    Chapter   BuildChapter   `json:"chapter"`
    Summary   BuildSummary   `json:"summary"`
    Context   BuildContext   `json:"context"`
    State     BuildState     `json:"state"`
    Structure BuildStructure `json:"structure"`
    Beats     BuildBeats     `json:"beats"`
    Prose     BuildProse     `json:"prose"`
}
```

字段必须覆盖 `docs/syntax.md` 的 build 要求：

```text
chapter:
  code, canonical_code, alias, title, purpose, target_length, volume_code,
  previous_chapter, next_chapter

summary:
  novel_summary, volume_summary, chapter_summary

context:
  relevant_characters, relevant_places, relevant_objects, relevant_facts

state:
  state_at_chapter_begin, expected_state_changes, state_at_chapter_end

structure:
  start_patterns, active_threads, active_promises, active_arcs,
  served_threads, served_promises, served_arcs

beats:
  ordered_beats, beat_preconditions, beat_effects, beat_render_hints

prose:
  prose_hint, target_length
```

Context 推导规则：

```text
chapter pov/location。
chapter beats 中出现的 refs。
beat effects 中的 subject refs。
beat reveals/mentions 中的 facts/objects。
active/served structures 的 subject refs。
```

---

## 19. Diagnostics

### 19.1 Code 范围

```text
E0000-E0013 project/config/files
E0200-E0222 parser/source-file shape
E0301-E0313 symbols/imports/references/query env
E0401-E0414 model/type checking
E0501-E0513 state/effect/anchor timeline
E0601-E0627 structure checks and derived views
E0701-E0726 query/eval
E0801-E0807 build/test command execution
W0001-W0002 config warnings
```

完整错误码目录见 `docs/error-codes.md`。

### 19.2 Severity

当前规范只需要：

```text
error
warning
```

结构检查默认 error。

---

## 20. 测试计划

### 20.1 单元测试

```text
lexer:
  Unicode identifier。
  string / multiline string。
  comments。
  operators。

parser:
  narr file。
  test file。
  field block。
  expression precedence。

chapter_code:
  parse / canonical / compare。

resolver:
  namespace split files。
  import alias。
  local variables precedence。

eval:
  comparison / membership。
  count / collect。
  narrative predicates。
  state queries。
```

### 20.2 集成测试

使用 `examples/红楼梦`：

```text
lint examples/红楼梦 -> pass。
build vol01.ch01 -> golden JSON in testdata。
info vol01.ch01 -> stable text golden。
query active_threads(...) -> expected refs。
test --all -> pass。
```

### 20.3 负例测试

在 `testdata/invalid` 准备：

```text
duplicate chapter code。
invalid import。
beat missing from chapter.beats。
effect unknown field。
promise payoff before setup。
start_pattern starts missing target。
test declares chapter。
query unknown relation。
```

---

## 21. 实现里程碑

### M1: Go 工程骨架

```text
创建 go.mod。
创建 cmd/narrc。
实现全局 flag 和子命令分发。
实现 project discovery 与 narr.toml 读取。
```

验收：

```bash
go run ./cmd/narrc --version
go run ./cmd/narrc lint --project examples/红楼梦
```

### M2: Lexer / Parser

```text
实现 tokenization。
实现 .narr 顶层声明解析。
实现 .test.narr 顶层 test 解析。
实现表达式 parser。
保留 source span。
```

验收：

```text
能解析 examples/红楼梦 所有 .narr 与 .test.narr。
语法错误有文件行列。
```

### M3: Symbol / Resolve

```text
建立 namespace symbol table。
实现 imports。
实现 chapter_code 归一化与查重。
实现基础 ref/path 解析。
```

验收：

```text
能解析 examples/红楼梦 所有跨 namespace 引用。
重复声明、未知引用能报错。
```

### M4: Model / Type Check

```text
AST 转 Model。
实现 class/entity 字段。
实现 enum / Symbol。
实现 effect 类型检查。
```

验收：

```text
examples/红楼梦 类型检查通过。
负例 effect unknown field 失败。
```

### M5: State Timeline

```text
构建初始状态。
按 chapter/beat 顺序应用 effect。
保存 chapter 与 beat 边界 checkpoint。
实现 state(field, at chapter.begin/end)。
```

验收：

```text
能求出 examples/红楼梦 第一回、第二回结尾状态。
```

### M6: Structure Check / Derive

```text
实现 start_pattern。
实现 promise/thread/arc 检查。
实现 StateDependentCheck：precondition、start_pattern.requires、state conflict。
实现 invariant。
实现 active/served/reveals/mentions。
```

验收：

```text
examples/红楼梦 lint pass。
active_threads / served_promises 查询返回预期。
```

### M7: Eval / Query

```text
实现表达式求值。
实现 count / collect。
实现 temporal predicate。
实现 narrative predicate。
实现内置函数。
```

验收：

```bash
go run ./cmd/narrc query 'count(chapter 章节 in chapters_in(structure.vol01) where 章节 serves promises.通灵宝玉来历伏笔)' --project examples/红楼梦
```

### M8: Build / Info

```text
实现 ChapterBuild。
实现 JSON writer。
实现 info 人类可读输出。
```

验收：

```bash
go run ./cmd/narrc build vol01.ch01 --project examples/红楼梦 --out-dir build
go run ./cmd/narrc info vol01.ch01 --project examples/红楼梦
```

### M9: Test Runner

```text
实现 test statement 执行。
实现 failure bindings。
实现 --all / explicit files。
实现 JSON test report。
```

验收：

```bash
go run ./cmd/narrc test --all --project examples/红楼梦
```

### M10: 稳定性与文档

```text
补全 error codes。
补全 README 用法。
加入 CI: go test ./...。
加入 examples/红楼梦 golden tests。
```

验收：

```bash
go test ./...
go vet ./...
```

---

## 22. 关键实现细节

### 22.1 Bare identifier 的歧义

示例：

```narr
chars.贾雨村.状态 = 新任知府
```

`新任知府` 不是 ref，而是 Symbol。解析期不要急于判定；求值/类型检查阶段依据目标字段类型决定。

### 22.2 `章节.end` 这类 path anchor

`章节` 可能是 test binder。实现 anchor 解析时要先 eval `章节` 得到 Chapter，再识别 `.begin/.end`。

### 22.3 `beat.after` 起点

虽然对外 `state(...)` 只支持 chapter 边界，start_pattern/thread/promise/arc 起点仍可使用 beat.before/after anchor。内部 timeline 需要保存 beat 边界。

### 22.4 循环依赖

避免：

```text
served -> build -> served
```

规则：

```text
build 依赖 served。
served 不依赖 build。
```

### 22.5 状态 checkpoint 性能

第一版可直接深拷贝 map，优先正确性。后续如果项目变大，再改 copy-on-write delta。

---

## 23. 风险与处理

```text
风险：字段块没有明确缩进语法。
处理：parser 使用 context-aware block 读取，遇到同级字段名或 } 停止。

风险：中文 bare symbol 与 ref 歧义高。
处理：解析期保留 UnresolvedName，类型检查期按上下文决定。

风险：precondition/start_pattern requires 需要 beat 边界状态。
处理：内部 timeline 支持 beat.before/after，CLI state 查询仍限制 chapter 边界。

风险：build context 推导可能过宽或过窄。
处理：先实现保守超集，保证自包含，再通过 golden tests 收窄。

风险：query/test 输出顺序不稳定。
处理：所有集合输出定义稳定排序。
```

---

## 24. 完成定义

满足以下条件视为 `narrc` Go 实现达到当前规范的完整可用版本：

```text
1. `go test ./...` 通过。
2. `narrc lint examples/红楼梦` 通过。
3. `narrc test --all --project examples/红楼梦` 通过。
4. `narrc build vol01.ch01 --project examples/红楼梦` 生成合法 JSON。
5. `narrc info vol01.ch01 --project examples/红楼梦` 输出章节摘要、状态和结构信息。
6. `narrc query` 支持 docs/syntax.md 中定义的 collection/relation/collect/count/state/predicate。
7. 主要错误路径有稳定 diagnostics 与测试覆盖。
8. 不接受旧语法扩展，遇到旧语法给出明确错误。
```
