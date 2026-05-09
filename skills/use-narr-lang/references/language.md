# Narr 语言参考

以 `docs/syntax.md` 作为事实源。本文件是处理 `.narr` 与 `narr.toml` 时的紧凑工作指南。

## 心智模型

Narr 是长篇小说蓝图语言，只有两层结构：

- 全局蓝图层：`novel`、`volume`、`thread`、`promise`、`arc`、`invariant`、`start_pattern`、实体。
- 局部章节层：`chapter`、`beat`、`effect`。

不要引入 `thread_stage`、`chapter_plan`、`outline`、`act`、`sequence`、`scene_plan`、`timeline`、`function`、`mode` 或散文生成声明，除非 `docs/syntax.md` 后续明确加入。

## 文件

```text
.narr        蓝图文件；可以声明结构和产生状态变化的 beats
.test.narr   测试文件；不能声明结构，不能修改状态
narr.toml    工程配置
```

最小配置：

```toml
[project]
name = "长夜之城"
version = "0.3.5"
```

常用配置：

```toml
[project]
name = "长夜之城"
version = "0.3.5"
language = "zh-CN"
main = "main.narr"
```

## Namespace 与 Import

每个源文件以 namespace 开始：

```narr
namespace 长夜之城.structure

import 长夜之城.world as world
import 长夜之城.characters as chars
```

文档只支持带可选 alias 的整 namespace import。不要使用通配符 import 或 named import。

## Chapter Code

`chapter_code` 是章节唯一身份：

```text
vol01.ch01
vol1.ch1
vol01.ch001
```

章节码归一化为显示形式 `volNN.chNN`。排序先比较卷号，再比较章号。前缀不同的 chapter code 不可比较。

## 核心声明

`.narr` 顶层声明包括：

```text
novel / enum / class
volume / chapter
start_pattern
place / character / collective / faction / object / fact
beat / promise / thread / arc / invariant / style_note
```

Volume 只保存元信息，不包含章节：

```narr
volume vol01 alias 开端卷 {
  title: "灰色王都"
  purpose: setup
  target_chapters: 80
}
```

Volume `purpose` 表示大叙事阶段，优先从这些值中选择：

```text
exposition
inciting_incident
conflict
rising_action
midpoint
climax
falling_action
denouement
resolution
aftermath
in_media_res
nonlinear_overview
setup
escalation
reversal
descent
revelation
interlude
```

Chapter 定义元信息和有序 beats：

```narr
chapter vol01.ch01 alias 城门 {
  title: "城门"
  purpose: entry
  start_pattern: 流亡者入城
  pov: chars.沈夜
  location: world.王都南门
  target_length: 2500 字
  summary: "..."
  beats: [beats.沈夜抵达城门, beats.火印异常]
}
```

Chapter `purpose` 表示更细的章节叙事功能，优先从这些值中选择：

```text
intro_characters
setup_conflict
escalate_conflict
reversal
foreshadow
flashback
flashforward
reveal_secret
turning_point
resolution_minor
transition
epilogue_segment
entry
encounter
discovery
conflict
choice
aftermath
reveal
interlude
climax
quiet
```

顶层 beat 必须锚定到 chapter，并出现在该 chapter 的 `beats` 列表中：

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
}
```

只有 `beat.effect` 可以改变叙事状态：

```narr
effect:
  state.field = expr
  state.set += value
  state.set -= value
  state.list append value
```

`precondition` 和 `.test.narr` assertion 不改变状态。

## 实体

用 class 声明可追踪字段，用实体声明具体世界对象：

```narr
class MajorCharacter {
  field 位置: Place
  field 存活: Bool = true
  field 知道: Set<Fact> = {}
}

place 王都 { 状态: Symbol = 封闭 }
place 王都南门 in 王都
character 沈夜 : MajorCharacter { 位置 = world.城外 }
fact 沈夜是王血 = "沈夜拥有旧王血脉"
```

## 结构关系

Start pattern 使用 start-only 模型：

```narr
start_pattern 流亡者入城 {
  at: vol01.ch01
  requires:
    chars.沈夜.位置 == world.城外
  starts:
    arc 沈夜角色弧
    promise 王血伏笔
    thread 王都线
}
```

Promise 的 payoff 不得早于 setup：

```narr
promise 王血伏笔 {
  setup_at: vol01.ch01
  start_pattern: 流亡者入城
  setup_strength: medium
  payoff_by: vol03.ch12
  question: "..."
  reader_visibility: implied
}
```

Thread 和 arc 描述持续结构：

```narr
thread 王都线 {
  kind: main_plot
  starts_at: vol01.ch01
  start_pattern: 流亡者入城
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

Invariant 由编译器检查：

```narr
invariant 王血秘密 {
  hidden: chars.沈夜是王血 until vol03.ch12
}

invariant 主角存活 {
  always:
    chars.沈夜.存活 == true
  active_until: vol03.ch12
}
```

## 编译器默认检查

除非用户明确要冗余项目测试，否则不要把以下内容重复写成 `.test.narr`：

- 语法、namespace、import、引用解析和类型合法性。
- chapter code 合法且唯一。
- 顶层 beat 的 anchor，以及是否出现在 `chapter.beats`。
- beat precondition、effect 类型合法性和 invariant 保护。
- promise setup/payoff 顺序。
- arc state field 和状态转移合法性。
- hidden/always invariant 语义。
- build 对象生成和引用可解析。
