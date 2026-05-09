## `narrc` 命令设计

```bash
Usage: narrc [global options] <command> [options] <args>

Commands:
  build <chapter_code>|--all    构建指定单章或全部章节的 build 文件
  test [files|--all]            执行 .test.narr 文件中的测试
  lint [files]                  检查 .narr 文件语法与规范
  info <chapter_code>           显示单章 build / 状态信息
  query <expr>                  查询派生视图与结构指标
  install-skill [name|--all]    安装内置 Codex skill
  init-project <name>           创建空 Narr 工程
```

---

### 1️⃣ `build` — 构建章节

```bash
narrc build vol01.ch01
narrc build --all
```

* **作用**：构建指定章节或项目内全部章节的 `build` 输出，默认生成 Narr-like 的 LLM 写作上下文。
* **输出示例**：

```
build/vol01.ch01.build.narr
chapter: vol01.ch01
beats: 2
active_threads: 王都线
active_promises: 王血伏笔
state_at_begin: ...
state_at_end: ...
```

* **可选参数**：

  * `--all`：按项目章节顺序构建全部 chapter
  * `--out-dir DIR`：指定输出目录
  * `--dry-run`：只检查语法，不生成 build
  * `--llm`：显式使用默认的 Narr-like LLM 写作上下文输出，文件头包含说明文档
  * `--json`：输出 JSON 到 stdout，不写文件；与 `--all` 搭配时输出 `builds`

---

### 2️⃣ `test` — 执行测试

```bash
narrc test tests/blueprint.test.narr
narrc test --all
```

* **作用**：运行 `.test.narr` 文件中所有 `test` 声明。
* **输出示例**：

```
[PASS] 所有章节服从全局蓝图
[FAIL] 章节只能服务当前活跃 promise
```

* **选项**：

  * `--verbose`：显示详细求值过程
  * `[files]`：指定某些 `.test.narr` 文件

---

### 3️⃣ `lint` — 检查语法和规范

```bash
narrc lint structure/chapters.narr
narrc lint world/*.narr
```

* **作用**：检查 `.narr` 文件语法、chapter_code、namespace、import 等是否符合规范。
* **输出示例**：

```
ERROR [E0403] chapter code vol01.chX 不合法
WARNING [W0102] import alias 冲突: world
PASS structure/volumes.narr
```

* **可选参数**：

  * `--verbose`：显示详细检查信息

---

### 4️⃣ `info` — 单章信息查询

```bash
narrc info vol01.ch01
```

* **作用**：显示单章的 build 状态和关键上下文（beats、active_threads、active_promises 等）。
* **用途**：辅助调试和测试 `.test.narr` 断言。

---

### 5️⃣ `query` — 查询派生视图与结构指标

```bash
narrc query 'active_threads(s.vol01.ch01)'
narrc query 'count(chapter 章节 in chapters_in(s.vol01) where 章节 advances s.王都线)'
```

* **作用**：直接执行查询表达式，查看编译器从 `.narr` 推导出的派生视图、结构关系或统计指标。
* **输出示例**：

```
query: count(chapter 章节 in chapters_in(s.vol01) where 章节 advances s.王都线)
result: 3
matched:
  - s.vol01.ch01
  - s.vol01.ch07
  - s.vol01.ch18
```

* **可选参数**：

  * `--json`：以 JSON 格式输出结果
  * `--verbose`：显示查询求值过程与匹配绑定

* **规则**：

  * `query` 不执行 `.test.narr` 测试
  * `query` 不生成或更新 snapshot
  * `query` 不修改 `.narr` 状态
  * `query` 只用于调试派生视图与结构指标

---

### 6️⃣ `install-skill` — 安装内置 skill

```bash
narrc install-skill --list
narrc install-skill use-narrc-cli
narrc install-skill use-narr-lang --global
```

* **作用**：从二进制内嵌的 `skills/` 目录安装 Codex skill，不依赖源码目录存在。
* **默认目标**：当前工作目录下的 `skills/`，并在 `.agents/skills/` 与 `.claude/skills/` 下创建指向 `./skills/<name>` 的软链接。
* **全局目标**：`${CODEX_HOME:-$HOME/.codex}/skills`。
* **选项**：

  * `--list`：列出二进制内置 skill
  * `--all`：安装全部内置 skill
  * `--local`：安装到当前目录 `./skills`，这是默认行为
  * `--global`：安装到全局 Codex skills 目录
  * `--force`：覆盖已存在的目标 skill
  * `--json`：输出 JSON 结果

---

### 7️⃣ `init-project` — 创建空项目

```bash
narrc init-project 长夜之城
narrc init-project "My Novel" --dir my-novel
```

* **作用**：用小说名创建一个空 Narr 工程，并从二进制内嵌的 `skills/` 自动安装本地 skill。
* **默认目录**：从小说名派生目录名。
* **生成内容**：

  * `narr.toml`：项目名、版本、语言和 main 文件
  * `main.narr`：合法空 namespace 文件
  * `AGENTS.md`：当前项目的 Codex 协作说明
  * `skills/`：内置 Codex skills，例如 `use-narrc-cli` 和 `use-narr-lang`
  * `.agents/skills/`：指向 `skills/` 内同名 skill 的软链接
  * `.claude/skills/`：指向 `skills/` 内同名 skill 的软链接

* **选项**：

  * `--dir DIR`：指定项目目录
  * `--json`：输出 JSON 结果

---

## 示例完整用法

```bash
# 构建单章 LLM 写作上下文
narrc build vol01.ch01 --out-dir build

# 构建全部章节 LLM 写作上下文
narrc build --all --out-dir build

# 显式使用默认 LLM 输出模式
narrc build vol01.ch01 --llm --out-dir build

# 运行某个测试文件
narrc test tests/blueprint.test.narr

# 运行全部测试
narrc test --all

# 检查语法和规范
narrc lint structure/*.narr world/*.narr

# 查询单章信息
narrc info vol01.ch01

# 查询派生视图与结构指标
narrc query 'active_threads(s.vol01.ch01)'
narrc query 'count(chapter 章节 in chapters_in(s.vol01) where 章节 advances s.王都线)'

# 安装内置 skill 到当前目录
narrc install-skill use-narrc-cli

# 安装内置 skill 到全局 Codex skills 目录
narrc install-skill use-narr-lang --global

# 创建空 Narr 工程，并自动安装内置 skill
narrc init-project 长夜之城
```
