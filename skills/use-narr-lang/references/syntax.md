# Narr 语言规范 v0.3.5 精简草案

## 0. 定位

Narr 是一门长篇小说蓝图语言。本规范保留编译器必须实现的能力：

```text
全局蓝图层：
  novel / volume / thread / promise / arc / invariant / start_pattern / entity

局部落地层：
  chapter / beat / effect / chapter build

非结构能力：
  派生视图
  .test.narr 设计测试
  narrc query 查询
```

Narr 不引入以下结构层：

```text
thread_stage
chapter_plan
outline
act
sequence
scene_plan
plot_point_decl
```

核心原则：

```text
结构只分两层：全局蓝图层与局部章节层。
全局到局部的展开关系由编译器派生，不作为新的结构声明。
结构必然约束由编译器默认检查。
项目特异的节奏、密度和平衡约束由 .test.narr 表达。
```

---

# 1. 文件类型

```text
.narr        小说蓝图文件
.test.narr   Narr 测试文件
narr.toml    工程配置文件
.md          可选散文输出，不属于 Narr 编译对象
```

```ebnf
source_file ::= narr_file | test_file

narr_file ::= namespace_decl import_decl* narr_declaration*

test_file ::= namespace_decl import_decl* test_decl*
```

`.test.narr` 不能声明小说结构，也不能修改状态。

---

# 2. narr.toml

最小合法配置：

```toml
[project]
name = "长夜之城"
version = "0.3.5"
```

推荐配置：

```toml
[project]
name = "长夜之城"
version = "0.3.5"
language = "zh-CN"
main = "main.narr"
```

结构检查默认是 error；测试失败是 test failure。

禁止字段：

```toml
mode = "literary"
type = "novel"
generation = "中文长篇"
```

---

# 3. 顶层声明

```ebnf
narr_declaration ::=
    novel_decl
  | enum_decl
  | class_decl
  | volume_decl
  | chapter_decl
  | start_pattern_decl
  | place_decl
  | character_decl
  | collective_decl
  | faction_decl
  | object_decl
  | fact_decl
  | beat_decl
  | promise_decl
  | thread_decl
  | arc_decl
  | invariant_decl
  | style_note_decl
```

`.narr` 不包含：

```text
test
timeline
mode
generation
function
thread_stage
chapter_plan
outline
```

---

# 4. Namespace 与 Import

```ebnf
namespace_decl ::= "namespace" namespace_path

namespace_path ::= identifier ("." identifier)*

import_decl ::= "import" namespace_path alias_clause?

alias_clause ::= "as" identifier
```

示例：

```narr
namespace 长夜之城.structure

import 长夜之城.world as world
import 长夜之城.characters as chars
```

不支持通配符或 named import：

```narr
import 长夜之城.world.*
import { 王都 } from 长夜之城.world
```

---

# 5. 基础词法

```ebnf
identifier ::= unicode_identifier

string ::= quoted_string

text ::= string | multiline_string

integer ::= digit+

bool ::= "true" | "false"

language_tag ::= identifier ("-" identifier)*

qualified_ref ::= identifier ("." identifier)*

literal ::=
    string
  | integer
  | bool
  | symbol_literal
  | list_expr
  | set_expr

list_expr ::= "[" expr_list? "]"

set_expr ::= "{" expr_list? "}"

expr_list ::= expr ("," expr)*
```

---

# 6. Chapter Code

章节唯一结构身份是 `chapter_code`。

```ebnf
chapter_code ::= volume_code "." chapter_part_code

volume_code ::= code_segment

chapter_part_code ::= code_segment

code_segment ::= identifier unsigned_integer

unsigned_integer ::= digit+
```

合法：

```narr
vol01.ch01
vol1.ch1
vol01.ch001
vol05.ch80
```

非法：

```narr
vol.ch01
vol01.ch
vol01.ch01.x
vol-01.ch01
```

归一化：

```text
vol01.ch001 == vol01.ch01
vol1.ch1    == vol01.ch01
vol001.ch01 == vol01.ch01
```

默认显示形式：

```text
vol01.ch01
```

顺序按卷号、章号比较。前缀不同的 chapter code 不可比较。

---

# 7. Novel 与 Length

```ebnf
novel_decl ::= "novel" identifier? block

novel_field ::=
    "title" ":" string
  | "language" ":" language_tag
  | "summary" ":" text
  | "length" ":" length_block
  | "prose_hint" ":" text

length_block ::= length_stmt*

length_stmt ::=
    "volumes" "=" integer
  | "chapters_per_volume" "=" integer
  | "chapter" "=" length_value

length_value ::= integer length_unit

length_unit ::= "字" | "chars" | "words"
```

示例：

```narr
novel 长夜之城 {
  title: "长夜之城"
  language: zh-CN
  summary: "沈夜进入王都后，被卷入旧王血、黑塔与王国记忆的长篇故事。"

  length:
    volumes = 5
    chapters_per_volume = 80
    chapter = 2500 字

  prose_hint: "克制、冷峻，有史诗感，但不要解释设定。"
}
```

长度是规划提示，不是硬约束。硬性字数限制应写入 `.test.narr`。

---

# 8. Volume

`volume` 是卷级元信息声明，不包含章节。

```ebnf
volume_decl ::= "volume" volume_code alias_clause? block?

volume_field ::=
    "title" ":" string
  | "purpose" ":" volume_purpose
  | "summary" ":" text
  | "target_chapters" ":" integer
  | "target_length" ":" length_value
```

合法 `purpose`：

```text
setup
escalation
reversal
descent
revelation
resolution
interlude
aftermath
```

---

# 9. Chapter

```ebnf
chapter_decl ::= "chapter" chapter_code alias_clause? block?

chapter_field ::=
    "title" ":" string
  | "purpose" ":" chapter_purpose
  | "start_pattern" ":" start_pattern_ref
  | "summary" ":" text
  | "target_length" ":" length_value
  | "pov" ":" character_ref
  | "location" ":" place_ref
  | "time_hint" ":" text
  | "beats" ":" list_expr
  | "prose_hint" ":" text
```

合法 `purpose`：

```text
entry
encounter
discovery
conflict
choice
reversal
aftermath
reveal
transition
interlude
climax
quiet
```

示例：

```narr
chapter vol01.ch01 alias 城门 {
  title: "城门"
  purpose: entry
  target_length: 2500 字
  start_pattern: 流亡者入城
  pov: chars.沈夜
  location: world.王都南门
  summary: "沈夜抵达王都南门，火印异常，王血伏笔启动。"
  beats: [beats.沈夜抵达城门, beats.火印异常]
  prose_hint: "写出封闭王都的压迫感。"
}
```

---

# 10. Beat

Beat 是章内状态变化单位。Beat 不构成第三层结构。

```ebnf
beat_decl ::= "beat" identifier beat_anchor? block?

beat_anchor ::= "@" anchor_ref

beat_field ::=
    "summary" ":" text
  | "precondition" ":" condition_block
  | "effect" ":" effect_block
  | "pov" ":" character_ref
  | "location" ":" place_ref
  | "on_screen" ":" bool
  | "observers" ":" set_expr
  | "sets_up" ":" narr_link_expr
  | "pays_off" ":" narr_link_expr
  | "advances" ":" narr_link_expr
  | "resolves" ":" narr_link_expr
  | "reveals" ":" narr_link_expr
  | "mentions" ":" narr_link_expr
  | "render_hint" ":" text

narr_link_expr ::= ref | list_expr | set_expr
```

示例：

```narr
beat 火印异常 @ structure.vol01.ch01 {
  precondition:
    chars.沈夜.位置 == world.王都南门

  effect:
    world.火印.状态 = 异常
    chars.沈夜.相信 += "自己只是运气好"

  pov: chars.沈夜
  on_screen: true
  observers: { chars.沈夜, world.城门守卫 }
  sets_up: structure.王血伏笔
  reveals: world.火印异常现象
  render_hint: "火印异常要可见，但不要解释王血真相。"
}
```

规则：

```text
顶层 beat 必须通过 @ 指向所属 chapter。
顶层 beat 必须出现在所属 chapter 的 beats: [...] 中。
chapter.beats 决定章内 beat 顺序。
叙事标注只从显式字段和 effect 推导，不依赖散文语义猜测。
```

---

# 11. Effect

只有 `beat.effect` 能改变叙事状态。

```ebnf
effect_block ::= effect_stmt*

effect_stmt ::=
    assignment
  | set_add
  | set_remove
  | list_append

assignment ::= state_ref "=" expr

set_add ::= state_ref "+=" expr

set_remove ::= state_ref "-=" expr

list_append ::= state_ref "append" expr
```

示例：

```narr
effect:
  chars.沈夜.位置 = world.王都南门
  world.火印.状态 = 异常
  chars.沈夜.相信 += "自己只是运气好"
```

规则：

```text
precondition 不改变状态。
test 不改变状态。
effect 必须符合字段类型。
effect 不得违反 active invariant。
```

---

# 12. TimeRef 与 AnchorRef

```ebnf
time_ref ::=
    "beginning"
  | "end_of_story"
  | volume_code
  | chapter_code
  | alias_ref
  | qualified_ref

anchor_ref ::= time_ref anchor_suffix?

anchor_suffix ::=
    ".begin"
  | ".end"
  | ".before"
  | ".after"
```

规则：

```text
volume / chapter 可使用 .begin 或 .end。
volume / chapter 不写 suffix 时，默认 .begin。
beat 可使用 .before 或 .after。
state(...) 查询只支持 chapter.begin / chapter.end。
start_pattern.at 可以使用 beat.after 或 beat.before 表达章内启动点。
```

示例：

```narr
at: vol01.ch01
at: vol01.ch01.end
at: beats.黑塔显形.after
```

---

# 13. Start Pattern

Start Pattern 采用 start-only 模型。需要章内启动时，直接把 `at` 写成 beat anchor。

```ebnf
start_pattern_decl ::= "start_pattern" identifier block

start_pattern_field ::=
    "at" ":" anchor_ref
  | "requires" ":" condition_block
  | "starts" ":" start_target_block
  | "tags" ":" set_expr
  | "note" ":" text

start_target_block ::= start_target*

start_target ::=
    "thread" ref
  | "promise" ref
  | "arc" ref
```

示例：

```narr
start_pattern 流亡者入城 {
  at: vol01.ch01

  requires:
    chars.沈夜.位置 == world.城外
    chars.沈夜.身份 == 隐藏
    world.王都.状态 == 封闭

  starts:
    arc 沈夜角色弧
    promise 王血伏笔
    thread 王都线
}
```

章内启动：

```narr
start_pattern 黑塔初现模式 {
  at: beats.黑塔显形.after

  requires:
    world.黑塔.可见 == true

  starts:
    thread 黑塔线
}
```

编译器检查：

```text
start_pattern.at 可解析。
start_pattern.requires 在 at 对应位置可满足。
start_pattern.starts 中的目标存在且类型为 thread / promise / arc。
引用同一 start_pattern 的 thread / promise / arc 起点必须等于 start_pattern.at。
```

---

# 14. Promise

```ebnf
promise_decl ::= "promise" identifier block

promise_field ::=
    "setup_at" ":" anchor_ref
  | "start_pattern" ":" start_pattern_ref
  | "setup_strength" ":" promise_strength
  | "payoff_by" ":" anchor_ref
  | "payoff_at" ":" anchor_ref
  | "payoff_kind" ":" payoff_kind
  | "question" ":" text
  | "reader_visibility" ":" reader_visibility
  | "tags" ":" set_expr
  | "note" ":" text
```

```text
promise_strength:
  weak
  medium
  strong

payoff_kind:
  answered
  reversed
  transformed_question
  emotional_payoff
  symbolic_payoff

reader_visibility:
  hidden
  implied
  visible
```

规则：

```text
setup_at 必须可解析。
payoff_by / payoff_at 如果存在，必须可解析。
payoff 不得早于 setup。
如果 promise 引用 start_pattern，则 setup_at 必须等于 start_pattern.at。
```

---

# 15. Thread

```ebnf
thread_decl ::= "thread" identifier block

thread_field ::=
    "kind" ":" thread_kind
  | "starts_at" ":" anchor_ref
  | "start_pattern" ":" start_pattern_ref
  | "expected_resolution" ":" anchor_ref
  | "resolved_at" ":" anchor_ref
  | "priority" ":" thread_priority
  | "tags" ":" set_expr
  | "note" ":" text
```

```text
thread_kind:
  main_plot
  mystery
  romance
  political
  emotional
  thematic
  subplot

thread_priority:
  main
  major
  minor
  background
```

规则：

```text
starts_at 必须可解析。
expected_resolution / resolved_at 如果存在，必须可解析。
如果 thread 引用 start_pattern，则 starts_at 必须等于 start_pattern.at。
```

---

# 16. Arc

```ebnf
arc_decl ::= "arc" identifier block

arc_field ::=
    "subject" ":" character_ref
  | "starts_at" ":" anchor_ref
  | "start_pattern" ":" start_pattern_ref
  | "state_field" ":" identifier
  | "initial" ":" expr
  | "states" ":" list_expr
  | "expected_resolution" ":" anchor_ref
  | "tags" ":" set_expr
  | "note" ":" text
```

规则：

```text
subject 必须是 character。
state_field 必须是 subject 上可追踪的状态字段。
如果 arc 引用 start_pattern，则 starts_at 必须等于 start_pattern.at。
states 如果存在，effect 对 state_field 的赋值必须落在 states 集合内。
```

---

# 17. Invariant

Invariant 表达必须由编译器默认验证的世界规则。本规范只支持两类 invariant：

```text
hidden: 某个 fact 在指定 anchor 前不能被读者可见。
always: 简单状态条件在 active_until 前的 chapter 边界保持为真。
```

```ebnf
invariant_decl ::= "invariant" identifier block

invariant_field ::=
    hidden_rule
  | always_rule
  | "active_until" ":" anchor_ref
  | "tags" ":" set_expr
  | "note" ":" text

hidden_rule ::= "hidden" ":" fact_ref "until" anchor_ref

always_rule ::= "always" ":" condition_block
```

示例：

```narr
invariant 王血秘密 {
  hidden: chars.沈夜是王血 until vol03.ch12
}
```

```narr
invariant 主角存活 {
  always:
    chars.沈夜.存活 == true

  active_until: vol03.ch12
}
```

规则：

```text
hidden 只检查 beat.reveals 是否提前暴露该 fact。
always 在相关 chapter.begin / chapter.end 状态上求值。
always 只接受可在 chapter 边界求值的简单条件。
```

---

# 18. Entity 声明

本规范保留 `enum`、`class` 和基础实体声明。

```ebnf
enum_decl ::= "enum" identifier block

class_decl ::= "class" identifier block

class_field ::= "field" identifier ":" type_expr default_clause?

default_clause ::= "=" expr

type_expr ::=
    identifier
  | qualified_ref
  | "Set" "<" type_expr ">"
  | "List" "<" type_expr ">"
```

```ebnf
place_decl ::= "place" identifier in_clause? block?

character_decl ::= "character" identifier class_clause? block?

collective_decl ::= "collective" identifier block?

faction_decl ::= "faction" identifier block?

object_decl ::= "object" identifier in_clause? block?

fact_decl ::= "fact" identifier "=" text

in_clause ::= "in" ref

class_clause ::= ":" ref
```

示例：

```narr
enum 城市状态 {
  封闭
  开放
  戒严
}

class MajorCharacter {
  field 位置: Place
  field 存活: Bool = true
  field 知道: Set<Fact> = {}
  field 相信: Set<Claim> = {}
}

place 王都 {
  状态: 城市状态 = 封闭
}

place 王都南门 in 王都

character 沈夜 : MajorCharacter {
  位置 = world.城外
}

fact 沈夜是王血 = "沈夜拥有旧王血脉"
```

---

# 19. Chapter Build

`build(章节)` 是编译器派生对象，不是 Narr 结构声明。

```text
narrc build vol01.ch01
narrc build --all
narrc build vol01.ch01 --llm
```

build 至少包含：

```text
chapter:
  code
  canonical_code
  alias
  title
  purpose
  target_length
  volume_code
  previous_chapter
  next_chapter

summary:
  novel_summary
  volume_summary
  chapter_summary

context:
  relevant_characters
  relevant_places
  relevant_objects
  relevant_facts

state:
  state_at_chapter_begin
  expected_state_changes
  state_at_chapter_end

structure:
  start_patterns
  active_threads
  active_promises
  active_arcs
  served_threads
  served_promises
  served_arcs

beats:
  ordered_beats
  beat_preconditions
  beat_effects
  beat_render_hints

prose:
  prose_hint
  target_length
```

规则：

```text
build 必须能生成单章自包含上下文。
build 中所有 ref 必须可解析。
.test.narr 可以查询 build 字段，但不负责验证 build 机制本身。
默认生成 Narr-like 写作上下文文件，文件头必须包含面向 LLM 的说明文档；该文件仍是 build artifact，不是 `.narr` 源声明。
`--all` 按 chapter 顺序为项目内全部章节生成 build artifact。
`--json` 输出 JSON 到 stdout，不写文件。
```

---

# 20. 派生视图

派生视图从 `.narr` 声明、chapter 顺序、beat 标注、effect、start_pattern 和 invariant 中确定性计算。

派生视图：

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

规则：

```text
派生视图不是新的结构层。
派生视图没有持久结构身份。
派生视图不能被 .narr 文件 import。
派生视图可以被 .test.narr 和 narrc query 查询。
派生视图的正确性由编译器语义保证。
```

---

# 21. `.test.narr`

`.test.narr` 只承担项目特异设计约束与探索性检查。

```ebnf
test_file ::= namespace_decl import_decl* test_decl*

test_decl ::= "test" string test_attr* block

test_attr ::= tags_clause

tags_clause ::= "tags" set_expr
```

示例：

```narr
test "vol01 内王都线必须推进至少 5 次" tags {vol01, 王都线, pacing} {
  assert count(chapter 章节 in chapters_in(s.vol01) where 章节 advances s.王都线) >= 5
}
```

`.test.narr` 不负责：

```text
基础语法合法性。
引用可解析性。
字段类型合法性。
effect 是否破坏 invariant。
start_pattern 是否满足。
build 是否自包含。
```

---

# 22. Test Statement

```ebnf
test_stmt ::=
    assert_stmt
  | let_stmt
  | forall_stmt
  | exists_stmt

assert_stmt ::= "assert" expr message_clause?

message_clause ::= "message" text

let_stmt ::= "let" identifier "=" expr

forall_stmt ::= "forall" binder where_clause? ":" test_stmt+

exists_stmt ::=
    "exists" binder where_clause?
  | "exists" binder where_clause? ":" test_stmt+

binder ::= domain_type identifier in_query_clause?

in_query_clause ::= "in" query_expr

where_clause ::= "where" expr
```

示例：

```narr
test "strong 伏笔 setup 与 payoff 至少间隔 5 章" tags {promise, pacing} {
  forall promise 伏笔 where 伏笔.setup_strength == strong and 伏笔.payoff_by exists:
    assert chapter_distance(伏笔.setup_at, 伏笔.payoff_by) >= 5
}
```

```narr
test "主角第一卷不得进入黑塔" tags {world, vol01, 黑塔} {
  forall chapter 章节 in chapters_in(s.vol01):
    assert state(chars.沈夜.位置, at 章节.end) != world.黑塔
}
```

规则：

```text
let 只在当前 test block 内有效。
let 不能创建小说结构。
assert 必须求值为 Bool。
exists 无子句时只检查存在性。
```

---

# 23. Domain Type

```ebnf
domain_type ::=
    "novel"
  | "volume"
  | "chapter"
  | "beat"
  | "thread"
  | "promise"
  | "arc"
  | "character"
  | "place"
  | "object"
  | "fact"
```

默认绑定范围：

```narr
forall chapter 章节:
  assert 章节.summary exists
```

等价于：

```narr
forall chapter 章节 in chapters:
  assert 章节.summary exists
```

---

# 24. Query Expression

查询表达式可用于：

```text
.test.narr
narrc query
```

```ebnf
query_expr ::=
    collection_query
  | relation_query
  | projection_query
  | path_expr

collection_query ::=
    "novels"
  | "volumes"
  | "chapters"
  | "beats"
  | "threads"
  | "promises"
  | "arcs"
  | "characters"
  | "places"
  | "objects"
  | "facts"

relation_query ::=
    "chapters_in" "(" expr ")"
  | "beats" "(" expr ")"
  | "active_threads" "(" expr ")"
  | "active_promises" "(" expr ")"
  | "active_arcs" "(" expr ")"
  | "served_threads" "(" expr ")"
  | "served_promises" "(" expr ")"
  | "served_arcs" "(" expr ")"
  | "reveals_in" "(" expr ")"
  | "mentions_in" "(" expr ")"
  | "build" "(" expr ")"

projection_query ::= "collect" "(" value_expr "from" binder where_clause? ")"
```

示例：

```narr
collect(章节.code from chapter 章节 in chapters where 章节 advances s.王都线)
```

```text
narrc query 'active_threads(s.vol01.ch01)'
narrc query 'count(chapter 章节 in chapters_in(s.vol01) where 章节 advances s.王都线)'
```

规则：

```text
narrc query 不执行 test。
narrc query 不修改 .narr 状态。
query 只支持上述 collection / relation / collect / count 形式。
```

---

# 25. Expression

```ebnf
expr ::= implication_expr

implication_expr ::=
    or_expr
  | or_expr "=>" implication_expr

or_expr ::= and_expr ("or" and_expr)*

and_expr ::= not_expr ("and" not_expr)*

not_expr ::=
    "not" not_expr
  | predicate_expr

predicate_expr ::=
    comparison_expr
  | existence_expr
  | temporal_predicate
  | narrative_predicate

comparison_expr ::=
    value_expr comparison_op value_expr
  | value_expr membership_op value_expr
  | value_expr

comparison_op ::= "==" | "!=" | "<" | "<=" | ">" | ">="

membership_op ::= "in" | "not" "in"

value_expr ::=
    literal
  | ref
  | path_expr
  | query_expr
  | aggregate_expr
  | state_expr
  | function_call
  | "(" expr ")"
```

---

# 26. 基础谓词与聚合

```ebnf
existence_expr ::=
    value_expr "exists"
  | value_expr "missing"

temporal_predicate ::=
    value_expr "precedes" value_expr
  | value_expr "at_or_before" value_expr
  | value_expr "at_or_after" value_expr
  | value_expr "between" value_expr "and" value_expr
  | value_expr "in_volume" value_expr

aggregate_expr ::= "count" "(" aggregate_target ")"

aggregate_target ::=
    query_expr
  | binder where_clause?
```

示例：

```narr
assert 章节.summary exists
assert s.王血伏笔.setup_at at_or_before s.王血伏笔.payoff_by
assert count(chapter 章节 in chapters_in(s.vol01) where 章节 advances s.王都线) >= 5
```

---

# 27. State Expression

`state(...)` 只查询 chapter 边界状态。

```ebnf
state_expr ::= "state" "(" state_ref "," "at" chapter_boundary_anchor ")"

state_ref ::= ref

chapter_boundary_anchor ::= chapter_ref ".begin" | chapter_ref ".end" | path_expr
```

示例：

```narr
assert state(chars.沈夜.位置, at s.vol01.ch01.begin) == world.城外
assert state(chars.沈夜.位置, at s.vol01.ch01.end) == world.王都南门
```

规则：

```text
state 查询不改变状态。
state 查询基于 beginning、初始实体字段、chapter 顺序和 beat.effect 推导。
如果状态在 chapter 边界不可唯一确定，查询求值报错。
```

---

# 28. Function Call

没有用户自定义函数，只提供以下内置函数：

```ebnf
function_call ::= builtin_function "(" argument_list? ")"

argument_list ::= expr ("," expr)*
```

```text
canonical(x)
volume_of(chapter)
chapter_of(beat_or_anchor)
previous(chapter)
next(chapter)
chapter_distance(anchor, anchor)
chapters_between(anchor, anchor)
```

示例：

```narr
assert canonical(vol01.ch001) == vol01.ch01
assert chapter_distance(s.王血伏笔.setup_at, s.王血伏笔.payoff_by) >= 5
```

---

# 29. Narrative Predicate

叙事谓词只从结构声明、beat 标注和 effect 推导，不依赖散文语义猜测。

```ebnf
narrative_predicate ::=
    service_predicate
  | promise_predicate
  | thread_predicate
  | arc_predicate
  | reveal_predicate
  | change_predicate

service_predicate ::=
    narrative_subject "serves" narr_structure_ref
  | narrative_subject "mentions" ref

promise_predicate ::=
    narrative_subject "sets_up" promise_ref
  | narrative_subject "pays_off" promise_ref

thread_predicate ::=
    narrative_subject "starts" thread_ref
  | narrative_subject "advances" thread_ref
  | narrative_subject "resolves" thread_ref

arc_predicate ::=
    narrative_subject "starts" arc_ref
  | narrative_subject "advances" arc_ref

reveal_predicate ::= narrative_subject "reveals" fact_ref

change_predicate ::=
    narrative_subject "changes" state_ref
  | narrative_subject "changes" state_ref "to" expr
  | narrative_subject "changes" state_ref "from" expr "to" expr

narrative_subject ::= ref | path_expr

narr_structure_ref ::= thread_ref | promise_ref | arc_ref | start_pattern_ref
```

语义：

```text
chapter serves X:
  章节的 start_pattern starts X，
  或章节内某个 beat 显式 sets_up / pays_off / advances / resolves X，
  或章节内某个 beat.effect 改变 arc.subject.arc.state_field。

chapter sets_up promise:
  章节是 promise.setup_at 所在章，
  或章节的 start_pattern starts 该 promise，
  或章节内某个 beat 显式 sets_up 该 promise。

chapter pays_off promise:
  章节是 promise.payoff_at 所在章，
  或章节内某个 beat 显式 pays_off 该 promise。

chapter advances thread:
  章节内某个 beat 显式 advances 该 thread。

chapter resolves thread:
  章节是 thread.resolved_at 所在章，
  或章节内某个 beat 显式 resolves 该 thread。

chapter advances arc:
  章节内某个 beat 显式 advances 该 arc，
  或章节内某个 beat.effect 改变 arc.subject 的 arc.state_field。

chapter reveals fact / mentions ref:
  章节内某个 beat 显式 reveals / mentions 该对象。

subject changes field:
  beat.effect 或 chapter 内任一 beat.effect 改变该 field。
```

不定义以下谓词：

```text
obeys blueprint
build(章节) is self_contained
章节 satisfies start_pattern
章节 preserves invariant
```

---

# 30. 名称解析

普通名称解析顺序：

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

测试局部变量具有最高优先级。

---

# 31. 顺序判定

Volume 顺序：

```text
volume 左卷 precedes volume 右卷
iff 左卷.volume_number < 右卷.volume_number
```

Chapter 顺序：

```text
chapter 前章 precedes chapter 后章
iff:
  前章.volume_number < 后章.volume_number
  or
  前章.volume_number == 后章.volume_number
  and 前章.chapter_number < 后章.chapter_number
```

Beat 顺序：

```text
同一章内使用 chapter.beats 列表顺序。
跨章先比较 chapter_code，再比较章内 beat 顺序。
```

---

# 32. 编译器必须检查

基础检查：

```text
语法是否合法。
narr.toml 是否存在且版本兼容。
.narr / .test.narr 文件类型是否正确。
namespace 是否有且只有一个。
import namespace 是否存在。
import alias 是否冲突。
当前 namespace 内是否重复声明。
chapter_code / volume_code 是否合法、可归一化、无重复、可比较。
禁止结构是否被非法使用。
引用是否可解析。
字段类型是否合法。
```

结构检查：

```text
volume 不得嵌套 chapter。
chapter.beats 中的 beat 必须存在且不重复。
顶层 beat 必须有 chapter anchor。
beat anchor 必须指向所属 chapter。
chapter.beats 必须让章内 beat 顺序唯一确定。
beat.precondition 必须在对应位置可满足。
beat.effect 必须符合字段类型。
beat.effect 不得造成明显状态冲突。
beat.effect 不得破坏 active invariant。
显式 sets_up / pays_off / advances / resolves / reveals / mentions 引用必须可解析且目标类型合法。
start_pattern.at 必须可解析。
start_pattern.requires 必须在 at 可满足。
start_pattern.starts 目标必须存在且类型合法。
thread / promise / arc 起点必须匹配其 start_pattern.at。
promise payoff 不得早于 setup。
arc.state_field 必须存在于 arc.subject。
arc 状态变化必须落在 arc.states 中。
hidden invariant 不得被提前 reveals。
always invariant 必须在 chapter 边界保持为真。
build(chapter) 必须生成核心字段且其中 ref 可解析。
state(field, at chapter boundary) 必须可唯一推导。
派生视图必须可确定性计算。
```

`.test.narr` 检查：

```text
.test.narr 只能包含 namespace / import / test。
test 中 let 变量必须在作用域内唯一。
tags 必须是 set。
forall / exists 的 domain_type 必须合法。
in_query_clause 的集合元素类型必须匹配 binder 类型。
count 目标必须是集合。
collect 的 binder 与 where 必须类型合法。
state 查询只能使用 chapter 边界 anchor。
narrative predicate 的 subject 与 target 类型必须合法。
test 中所有 assert 必须求值为 Bool。
test 不得声明小说结构。
test 不得写 effect。
test 不得使用结构合规宏谓词。
```

---

# 33. 编译器不检查

Narr 不检查：

```text
小说是否好看。
章节是否精彩。
人物是否有魅力。
伏笔是否震撼。
主题是否深刻。
散文是否优美。
LLM 是否一定写出好文本。
```

叙事谓词只表示结构关系，不表示文学质量。

---

# 34. 核心示例

```narr
namespace 长夜之城.structure

import 长夜之城.world as world
import 长夜之城.characters as chars
import 长夜之城.beats as beats

volume vol01 alias 开端卷 {
  title: "灰色王都"
  purpose: setup
  summary: "沈夜进入王都，主线与王血伏笔启动。"
  target_chapters: 80
}

chapter vol01.ch01 alias 城门 {
  title: "城门"
  purpose: entry
  start_pattern: 流亡者入城
  pov: chars.沈夜
  location: world.王都南门
  summary: "沈夜抵达王都南门，火印异常，王血伏笔启动。"
  beats: [beats.沈夜抵达城门, beats.火印异常]
}

start_pattern 流亡者入城 {
  at: vol01.ch01

  requires:
    chars.沈夜.位置 == world.城外

  starts:
    arc 沈夜角色弧
    promise 王血伏笔
    thread 王都线
}

promise 王血伏笔 {
  setup_at: vol01.ch01
  start_pattern: 流亡者入城
  setup_strength: medium
  payoff_by: vol03.ch12
  question: "沈夜为什么能通过火印？"
  reader_visibility: implied
}

thread 王都线 {
  kind: main_plot
  starts_at: vol01.ch01
  start_pattern: 流亡者入城
  expected_resolution: vol03.ch12
  priority: main
}

arc 沈夜角色弧 {
  subject: chars.沈夜
  starts_at: vol01.ch01
  start_pattern: 流亡者入城
  state_field: arc_state
  initial: 逃避
  states: [逃避, 怀疑, 承认身份, 改变]
}
```

```narr
namespace 长夜之城.beats

import 长夜之城.structure as structure
import 长夜之城.world as world
import 长夜之城.characters as chars

beat 沈夜抵达城门 @ structure.vol01.ch01 {
  effect:
    chars.沈夜.位置 = world.王都南门

  pov: chars.沈夜
  on_screen: true
}

beat 火印异常 @ structure.vol01.ch01 {
  precondition:
    chars.沈夜.位置 == world.王都南门

  effect:
    world.火印.状态 = 异常

  sets_up: structure.王血伏笔
  reveals: world.火印异常现象
}
```

```narr
namespace 长夜之城.tests

import 长夜之城.structure as s
import 长夜之城.characters as chars
import 长夜之城.world as world

test "vol01 内王都线必须推进至少 5 次" tags {vol01, 王都线, pacing} {
  assert count(chapter 章节 in chapters_in(s.vol01) where 章节 advances s.王都线) >= 5
}

test "主角第一卷不得进入黑塔" tags {world, vol01, 黑塔} {
  forall chapter 章节 in chapters_in(s.vol01):
    assert state(chars.沈夜.位置, at 章节.end) != world.黑塔
}
```

---

# 35. 最终边界

```text
Narr 保留两层小说蓝图、chapter_code 顺序系统、chapter-first build、
状态变化推导、start_pattern 启动关系、thread/promise/arc 活跃与服务关系、
简单 invariant、项目设计测试，以及 narrc query。
```
