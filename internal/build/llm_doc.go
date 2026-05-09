package build

const llmBuildNarrDoc = `// LLM 使用说明
// 1. 本文件由 narrc build 生成，是单章写作上下文，不是项目源文件；不要把 build {...} 写回 .narr。
// 2. Narr 只有全局蓝图层和局部章节层；不要新增 outline、chapter_plan、scene_plan、act、sequence 等结构层。
// 3. 写作时严格遵守 chapter、ordered_beats、precondition、effect、state_at_chapter_begin、state_at_chapter_end 与 target_length。
// 4. beat 是章内状态变化单位；按 ordered_beats 顺序展开，effect 描述本章必须完成的状态变化。
// 5. state_at_chapter_begin 是开章事实，expected_state_changes 是本章预期变化，state_at_chapter_end 是收束状态。
// 6. 只补充散文表达和场面细节，不改动结构、引用、状态边界或未在 build 中出现的设定。
//
// build.narr 简易语法与属性说明
// build <chapter_code> { ... }：单章派生上下文根节点；chapter_code 是归一化章节号。
// chapter：目标章节元信息。
// chapter.code：源文件中写入的章节号。
// chapter.canonical_code：编译器归一化后的章节号，作为 build 主键。
// chapter.alias：章节别名，供人读和引用 disambiguation 使用。
// chapter.title：章回标题或章节标题。
// chapter.purpose：章节叙事功能，例如 entry、discovery、conflict。
// chapter.target_length：本章目标字数或长度提示。
// chapter.volume_code：本章所属卷。
// chapter.previous_chapter / chapter.next_chapter：相邻章节，用于承接与悬念控制。
// summary：写作前必须继承的摘要。
// summary.novel_summary：整部作品的总体叙事摘要。
// summary.volume_summary：当前卷的叙事摘要。
// summary.chapter_summary：本章必须完成的摘要。
// context：本章相关实体集合，只写这些实体已知设定，不擅自扩展世界观。
// context.relevant_characters：本章相关角色。
// context.relevant_places：本章相关地点。
// context.relevant_objects：本章相关物件。
// context.relevant_facts：本章相关事实、信息或已揭示设定。
// state：章节边界状态，约束本章开局、变化与收束。
// state.state_at_chapter_begin：开章时已经成立的状态事实。
// state.expected_state_changes：本章从 begin 到 end 必须发生的状态变化。
// state.state_at_chapter_end：章末必须成立的状态事实。
// structure：本章涉及的叙事结构关系。
// structure.start_patterns：本章触发或承接的开端模式。
// structure.active_threads：本章开始时仍活跃的主线/支线。
// structure.active_promises：本章开始时仍活跃的伏笔或悬念。
// structure.active_arcs：本章开始时仍活跃的人物或主题弧线。
// structure.served_threads：本章实际推进的线索。
// structure.served_promises：本章实际设置、推进或兑现的伏笔。
// structure.served_arcs：本章实际推进的人物或主题弧线。
// beats：章内状态变化单位，必须按 ordered_beats 顺序展开成正文。
// beats.ordered_beats：本章 beat 顺序，是正文组织的主骨架。
// beats.beat_preconditions：进入某 beat 前必须满足的条件。
// beats.beat_effects：每个 beat 必须造成的状态变化；effect 行使用 target op value。
// beats.beat_render_hints：单个 beat 的呈现方式提示。
// prose：正文生成提示。
// prose.prose_hint：语气、文风、叙述方式等散文提示。
// prose.target_length：正文目标长度，通常与 chapter.target_length 一致。
// 字面量说明：字符串使用双引号；引用形如 namespace.name；列表使用 [item, ...]；状态集合使用 {item, ...}。
// effect 操作：= 表示赋值，+= 表示集合添加，-= 表示集合移除，append 表示列表追加。

`
