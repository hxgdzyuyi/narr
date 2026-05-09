# Narr 语言浓缩规则 v0.3.5

## 1. 定位

长篇小说蓝图语言。**只有两层结构**：

- **全局蓝图层**：`novel / volume / thread / promise / arc / invariant / start_pattern / entity`
- **局部章节层**：`chapter / beat / effect`

**不存在**：`thread_stage / chapter_plan / outline / act / sequence / scene_plan`。全局到局部的展开由编译器派生，不是新结构。

## 2. 文件与配置

```text
.narr        蓝图文件（可声明结构、可改状态）
.test.narr   测试文件（不可声明结构、不可改状态）
narr.toml    工程配置
```

最小 `narr.toml`：
```toml
[project]
name = "长夜之城"
version = "0.3.5"
```

## 3. Namespace 与 Import

```narr
namespace 长夜之城.structure
import 长夜之城.world as world
```

不支持通配符或 named import。

## 4. Chapter Code（章节唯一身份）

```text
格式：volN.chM        例：vol01.ch01
归一化：vol1.ch1 == vol01.ch001 == vol01.ch01
默认显示：vol01.ch01
顺序：先比卷号，再比章号
```

## 5. 顶层声明清单

```text
novel / enum / class
volume / chapter
start_pattern
place / character / collective / faction / object / fact
beat / promise / thread / arc / invariant / style_note
```

## 6. Novel

```narr
novel 长夜之城 {
  title: "长夜之城"
  language: zh-CN
  summary: "..."
  length:
    volumes = 5
    chapters_per_volume = 80
    chapter = 2500 字
  prose_hint: "克制、冷峻。"
}
```

`length` 是规划提示，不是硬约束。硬性字数约束写在 `.test.narr`。

## 7. Volume（只是元信息，不含章节）

```narr
volume vol01 alias 开端卷 {
  title: "..."
  purpose: setup     // setup|escalation|reversal|descent|revelation|resolution|interlude|aftermath
  summary: "..."
  target_chapters: 80
}
```

## 8. Chapter

```narr
chapter vol01.ch01 alias 城门 {
  title: "城门"
  purpose: entry     // entry|encounter|discovery|conflict|choice|reversal|aftermath|reveal|transition|interlude|climax|quiet
  start_pattern: 流亡者入城
  pov: chars.沈夜
  location: world.王都南门
  target_length: 2500 字
  summary: "..."
  beats: [beats.沈夜抵达城门, beats.火印异常]   // 决定章内 beat 顺序
  prose_hint: "..."
}
```

## 9. Beat（章内状态变化单位，不构成第三层结构）

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
  // 还可：pays_off / advances / resolves / mentions / render_hint
}
```

**规则**：顶层 beat 必须 `@` 到某 chapter 且必须出现在该 chapter 的 `beats:` 中。叙事标注只从显式字段和 effect 推导。

## 10. Effect（唯一可改状态的位置）

```narr
effect:
  state.field = expr        // 赋值
  state.set += value        // 集合添加
  state.set -= value        // 集合移除
  state.list append value   // 列表追加
```

`precondition` 与 `test` 不改变状态。effect 不得违反 active invariant。

## 11. TimeRef 与 AnchorRef

```text
time_ref:    beginning | end_of_story | volume_code | chapter_code | alias | qualified_ref
anchor_suffix: .begin | .end | .before | .after

volume/chapter：可用 .begin / .end，省略时默认 .begin
beat：可用 .before / .after
state(...) 只支持 chapter.begin / chapter.end
```

例：`at: beats.黑塔显形.after`

## 12. Start Pattern（start-only 模型）

```narr
start_pattern 流亡者入城 {
  at: vol01.ch01                              // 也可写 beats.X.after 实现章内启动
  requires:
    chars.沈夜.位置 == world.城外
  starts:
    arc 沈夜角色弧
    promise 王血伏笔
    thread 王都线
}
```

引用同一 start_pattern 的 thread/promise/arc 起点必须等于 `start_pattern.at`。

## 13. Promise

```narr
promise 王血伏笔 {
  setup_at: vol01.ch01
  start_pattern: 流亡者入城
  setup_strength: medium     // weak|medium|strong
  payoff_by: vol03.ch12
  payoff_at: ...
  payoff_kind: answered      // answered|reversed|transformed_question|emotional_payoff|symbolic_payoff
  question: "..."
  reader_visibility: implied // hidden|implied|visible
}
```

payoff 不得早于 setup。

## 14. Thread

```narr
thread 王都线 {
  kind: main_plot            // main_plot|mystery|romance|political|emotional|thematic|subplot
  starts_at: vol01.ch01
  start_pattern: 流亡者入城
  expected_resolution: vol03.ch12
  resolved_at: ...
  priority: main             // main|major|minor|background
}
```

## 15. Arc

```narr
arc 沈夜角色弧 {
  subject: chars.沈夜          // 必须是 character
  starts_at: vol01.ch01
  start_pattern: 流亡者入城
  state_field: arc_state       // 必须是 subject 的可追踪字段
  initial: 逃避
  states: [逃避, 怀疑, 承认身份, 改变]   // effect 对该字段赋值必须落在此集合内
}
```

## 16. Invariant（编译器默认验证）

只有两类：

```narr
// hidden：fact 在指定 anchor 前不得被 beat.reveals 暴露
invariant 王血秘密 {
  hidden: chars.沈夜是王血 until vol03.ch12
}

// always：在 active_until 之前的 chapter 边界保持为真（只接受可在边界求值的简单条件）
invariant 主角存活 {
  always:
    chars.沈夜.存活 == true
  active_until: vol03.ch12
}
```

## 17. Entity 声明

```narr
enum 城市状态 { 封闭  开放  戒严 }

class MajorCharacter {
  field 位置: Place
  field 存活: Bool = true
  field 知道: Set<Fact> = {}
}

place 王都 { 状态: 城市状态 = 封闭 }
place 王都南门 in 王都
character 沈夜 : MajorCharacter { 位置 = world.城外 }
collective 守卫团 { ... }
faction 王党 { ... }
object 火印 in 王都南门 { ... }
fact 沈夜是王血 = "沈夜拥有旧王血脉"
```

## 18. Chapter Build（编译器派生，非声明）

```text
narrc build vol01.ch01
narrc build --all
narrc build vol01.ch01 --llm
```

包含：chapter 元信息、summary（novel/volume/chapter）、context（相关实体）、state（章首/章末/预期变化）、structure（active/served threads/promises/arcs）、beats（顺序/前置/效果/render_hint）、prose（hint/length）。

默认生成 Narr-like 写作上下文文件，文件头包含说明文档。它仍是编译器派生产物，不是 `.narr` 源声明。`--all` 按 chapter 顺序为项目内全部章节生成 build artifact。`--json` 输出 JSON 到 stdout。

## 19. 派生视图（确定性计算，不可被 .narr import）

```text
chapters_in(volume)             beats(chapter)
active_threads(chapter)         active_promises(chapter)         active_arcs(chapter)
served_threads(chapter)         served_promises(chapter)         served_arcs(chapter)
reveals_in(chapter)             mentions_in(chapter)
state(field, at chapter.begin|chapter.end)
build(chapter)
```

## 20. `.test.narr`（项目特异约束）

```narr
test "vol01 内王都线必须推进至少 5 次" tags {vol01, pacing} {
  assert count(chapter 章节 in chapters_in(s.vol01) where 章节 advances s.王都线) >= 5
}

test "主角第一卷不得进入黑塔" tags {world, vol01} {
  forall chapter 章节 in chapters_in(s.vol01):
    assert state(chars.沈夜.位置, at 章节.end) != world.黑塔
}

test "strong 伏笔 setup 与 payoff 至少间隔 5 章" tags {promise} {
  forall promise 伏笔 where 伏笔.setup_strength == strong and 伏笔.payoff_by exists:
    assert chapter_distance(伏笔.setup_at, 伏笔.payoff_by) >= 5
}
```

`.test.narr` 只能含 `namespace / import / test`，不可写 effect、不可声明结构。

## 21. Test 语句

```text
assert expr [message "..."]
let id = expr                                // 仅 test block 内有效
forall binder [where expr]: stmt+
exists binder [where expr] [: stmt+]

binder ::= domain_type id [in query_expr]
domain_type ::= novel|volume|chapter|beat|thread|promise|arc|character|place|object|fact
```

## 22. Query Expression（用于 test 与 `narrc query`）

```text
collection: novels|volumes|chapters|beats|threads|promises|arcs|characters|places|objects|facts
relation:   chapters_in(v) | beats(c) | active_threads(c) | active_promises(c) | active_arcs(c)
            served_threads(c) | served_promises(c) | served_arcs(c)
            reveals_in(c) | mentions_in(c) | build(c)
projection: collect(value_expr from binder [where expr])
aggregate:  count(query_expr | binder [where])
```

`narrc query` 不执行 test，不修改状态。

## 23. 表达式与谓词

```text
逻辑：    not / and / or / =>
比较：    == != < <= > >=
成员：    in / not in
存在：    expr exists | expr missing
时序：    a precedes b | a at_or_before b | a at_or_after b
          a between b and c | x in_volume v
```

## 24. State 查询（只查章节边界）

```narr
state(chars.沈夜.位置, at s.vol01.ch01.begin) == world.城外
state(chars.沈夜.位置, at s.vol01.ch01.end)   == world.王都南门
```

基于 `beginning` + 实体初值 + chapter 顺序 + `beat.effect` 推导。边界状态不唯一即报错。

## 25. 内置函数（无用户自定义函数）

```text
canonical(x)
volume_of(chapter)            chapter_of(beat_or_anchor)
previous(chapter)             next(chapter)
chapter_distance(a, b)        chapters_between(a, b)
```

## 26. 叙事谓词（只从结构与标注推导，不靠散文猜测）

```text
subject serves X                  // X: thread|promise|arc|start_pattern
subject mentions ref

subject sets_up promise           subject pays_off promise
subject starts thread             subject advances thread     subject resolves thread
subject starts arc                subject advances arc
subject reveals fact

subject changes field
subject changes field to value
subject changes field from a to b
```

**语义要点**：
- `chapter serves X`：章的 start_pattern 启动 X，或某 beat 显式服务 X，或某 beat.effect 改 arc 状态字段。
- `chapter sets_up/pays_off promise`：是 promise 的 setup_at/payoff_at 章，或某 beat 显式标注。
- `chapter advances arc`：某 beat 显式 advances，或某 beat.effect 改 arc 状态字段。

**不存在**的谓词：`obeys blueprint / build is self_contained / satisfies start_pattern / preserves invariant`（这些由编译器默认检查，不是测试谓词）。

## 27. 名称解析顺序

```text
1. 当前 block 内 let / quantifier 局部变量（最高优先级）
2. 当前 namespace 内声明
3. import alias
4. 当前 namespace 内 chapter_code
5. import namespace 内 chapter_code
6. 唯一 chapter alias
7. volume code / volume alias
8. enum / Symbol 类型上下文
9. 报错
```

## 28. 顺序判定

```text
volume：按 volume_number
chapter：先比卷号，再比章号
beat：同章按 chapter.beats 列表，跨章先比 chapter_code
```

## 29. 编译器默认检查（不需测试）

- 语法、namespace 唯一、import 可解析、引用可解析、类型合法
- chapter_code 合法且无重复、可比较
- volume 不含 chapter；顶层 beat 必须 @ 到 chapter 且在 chapter.beats 中；章内顺序唯一确定
- beat.precondition 在位置可满足；beat.effect 类型合法且不破坏 active invariant
- 显式叙事标注引用合法
- start_pattern：at 可解析、requires 可满足、starts 目标合法、引用方起点等于 at
- promise.payoff 不早于 setup；arc.state_field 存在、状态变化在 states 内
- hidden invariant 不被提前 reveal；always invariant 在 chapter 边界为真
- build(chapter) 可生成且 ref 可解析；state 边界查询可唯一推导
- `.test.narr` 不声明结构、不写 effect

## 30. 编译器**不**检查

是否好看 / 精彩 / 有魅力 / 震撼 / 深刻 / 优美 / LLM 是否写出好文本。叙事谓词只表示结构关系，不表示文学质量。

## 31. 最小完整示例

```narr
namespace 长夜之城.structure
import 长夜之城.world as world
import 长夜之城.characters as chars
import 长夜之城.beats as beats

volume vol01 alias 开端卷 {
  title: "灰色王都"  purpose: setup  target_chapters: 80
}

chapter vol01.ch01 alias 城门 {
  title: "城门"  purpose: entry
  start_pattern: 流亡者入城
  pov: chars.沈夜  location: world.王都南门
  beats: [beats.沈夜抵达城门, beats.火印异常]
}

start_pattern 流亡者入城 {
  at: vol01.ch01
  requires: chars.沈夜.位置 == world.城外
  starts:
    arc 沈夜角色弧
    promise 王血伏笔
    thread 王都线
}

promise 王血伏笔 {
  setup_at: vol01.ch01  start_pattern: 流亡者入城
  setup_strength: medium  payoff_by: vol03.ch12
  reader_visibility: implied
}

thread 王都线 {
  kind: main_plot  starts_at: vol01.ch01
  start_pattern: 流亡者入城  priority: main
}

arc 沈夜角色弧 {
  subject: chars.沈夜  starts_at: vol01.ch01
  start_pattern: 流亡者入城
  state_field: arc_state  initial: 逃避
  states: [逃避, 怀疑, 承认身份, 改变]
}
```

```narr
// beats.narr
beat 火印异常 @ structure.vol01.ch01 {
  precondition: chars.沈夜.位置 == world.王都南门
  effect: world.火印.状态 = 异常
  sets_up: structure.王血伏笔
  reveals: world.火印异常现象
}
```

---

**核心心智模型**：写 `.narr` = 声明结构骨架与状态变化点；编译器派生一切关系视图；写 `.test.narr` = 表达项目特异约束；散文写作交给 LLM，由 `build(chapter)` 提供自包含上下文。
