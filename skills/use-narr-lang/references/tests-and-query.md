# .test.narr 与 Query 参考

使用 `.test.narr` 表达项目特异设计约束和探索性检查。不要用测试重复编译器默认合法性检查。

## 文件形态

```narr
namespace 红楼梦.tests

import 红楼梦.structure as structure
import 红楼梦.promises as p
import 红楼梦.world as world

test "关键章都有 alias" tags {metadata} {
  forall chapter 章节 where 章节.purpose in { entry, discovery, reveal, climax }:
    assert 章节.alias exists
}
```

`.test.narr` 文件只能包含 `namespace`、`import` 和 `test` 声明。不能声明小说结构，不能写 `effect`。

## Test 语句

```text
assert expr [message "..."]
let id = expr
forall binder [where expr]: stmt+
exists binder [where expr] [: stmt+]
```

Binder 类型：

```text
novel | volume | chapter | beat | thread | promise | arc | character | place | object | fact
```

示例：

```narr
test "strong 伏笔 setup 与 payoff 至少间隔 5 章" tags {promise, pacing} {
  forall promise 伏笔 where 伏笔.setup_strength == strong and 伏笔.payoff_by exists:
    assert chapter_distance(伏笔.setup_at, 伏笔.payoff_by) >= 5
}

test "主角第一卷不得进入黑塔" tags {world, vol01, 黑塔} {
  forall chapter 章节 in chapters_in(s.vol01):
    assert state(chars.沈夜.位置, at 章节.end) != world.黑塔
}
```

## Query 表达式

Query 可用于 `.test.narr` 和 `narrc query`。

集合：

```text
novels | volumes | chapters | beats | threads | promises | arcs | characters | places | objects | facts
```

关系和派生视图：

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
build(chapter)
```

投影和聚合：

```narr
collect(章节.code from chapter 章节 in chapters where 章节 advances s.王都线)
count(chapter 章节 in chapters_in(s.vol01) where 章节 advances s.王都线)
```

CLI 示例：

```bash
go run ./cmd/narrc query --project examples/红楼梦 'active_threads(s.vol01.ch01)'
go run ./cmd/narrc query --project examples/红楼梦 'count(chapter 章节 in chapters_in(s.vol01) where 章节 advances s.王都线)'
```

`narrc query` 不运行测试，不更新 snapshot，不修改 `.narr` 状态。

## 表达式与谓词

使用标准逻辑和比较：

```text
not / and / or / =>
== != < <= > >=
in / not in
expr exists | expr missing
```

时序谓词：

```text
a precedes b
a at_or_before b
a at_or_after b
a between b and c
x in_volume v
```

State 查询只检查章节边界：

```narr
state(chars.沈夜.位置, at s.vol01.ch01.begin)
state(chars.沈夜.位置, at s.vol01.ch01.end)
```

State 从 `beginning`、实体初值、chapter 顺序和 `beat.effect` 推导。

内置函数：

```text
canonical(x)
volume_of(chapter)
chapter_of(beat_or_anchor)
previous(chapter)
next(chapter)
chapter_distance(anchor, anchor)
chapters_between(anchor, anchor)
```

叙事谓词只从结构、beat 标注和 effect 推导：

```text
subject serves X
subject mentions ref
subject sets_up promise
subject pays_off promise
subject starts thread
subject advances thread
subject resolves thread
subject starts arc
subject advances arc
subject reveals fact
subject changes field
subject changes field to value
subject changes field from a to b
```

避免使用不支持的质量谓词，例如 `obeys blueprint`、`build is self_contained`、`satisfies start_pattern` 或 `preserves invariant`；这些属于编译器职责，不是测试谓词。
