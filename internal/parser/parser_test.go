package parser

import (
	"path/filepath"
	"strconv"
	"testing"

	"narr/internal/ast"
	"narr/internal/project"
	"narr/internal/source"
)

func TestParseExamplesProject(t *testing.T) {
	root := filepath.Join("..", "..", "examples", "红楼梦")
	loaded, diagnostics := project.Load(project.LoadOptions{ProjectDir: root})
	if source.HasErrors(diagnostics) {
		t.Fatalf("project.Load diagnostics: %#v", diagnostics)
	}
	files, diagnostics := ParseProject(loaded)
	if source.HasErrors(diagnostics) {
		t.Fatalf("ParseProject diagnostics:\n%s", formatDiagnosticsForTest(diagnostics))
	}
	if len(files) != 16 {
		t.Fatalf("len(files) = %d, want 16", len(files))
	}
}

func TestParseNarrDeclarations(t *testing.T) {
	file, diagnostics := ParseNarrFile("sample.narr", `namespace 长夜.structure
import 长夜.world as world

novel 长夜 {
  title: "长夜"
  language: zh-CN
  length:
    volumes = 1
    chapters_per_volume = 80
    chapter = 2500 字
}

enum 状态 {
  封闭
  开放
}

class 叙事人物 {
  field 位置: Place = world.未明
  field 知道: Set<Fact> = {}
}

place 王都 {
  状态: 状态 = 封闭
}

character 沈夜 : 叙事人物 {
  位置 = world.王都
}

volume vol01 alias 开端卷 {
  title: "开端"
  purpose: setup
}

chapter vol01.ch01 alias 城门 {
  title: "城门"
  target_length: 2500 字
  beats: [入城]
}

beat 入城 @ vol01.ch01 {
  precondition:
    world.王都.状态 == 封闭

  effect:
    chars.沈夜.位置 = world.王都
    chars.沈夜.知道 += world.王都事实

  on_screen: true
  observers: { chars.沈夜 }
}

start_pattern 入城模式 {
  at: vol01.ch01
  requires:
    world.王都.状态 == 封闭
  starts:
    thread 主线
    promise 入城伏笔
    arc 沈夜弧
}

promise 入城伏笔 {
  setup_at: vol01.ch01
  setup_strength: strong
  payoff_by: vol01.ch02
}

thread 主线 {
  kind: main_plot
  starts_at: vol01.ch01
  priority: main
}

arc 沈夜弧 {
  subject: chars.沈夜
  starts_at: vol01.ch01
  state_field: arc_state
  initial: 未启动
  states: [未启动, 入城]
}

invariant 秘密 {
  hidden: world.秘密 until vol01.ch10
}

fact 王都事实 = "王都封闭"
`)
	if source.HasErrors(diagnostics) {
		t.Fatalf("ParseNarrFile diagnostics:\n%s", formatDiagnosticsForTest(diagnostics))
	}
	if file.Namespace != "长夜.structure" {
		t.Fatalf("Namespace = %q, want 长夜.structure", file.Namespace)
	}
	if len(file.Imports) != 1 {
		t.Fatalf("len(Imports) = %d, want 1", len(file.Imports))
	}
	if len(file.Decls) != 14 {
		t.Fatalf("len(Decls) = %d, want 14", len(file.Decls))
	}
	if file.Decls[0].Kind != ast.DeclNovel {
		t.Fatalf("first decl kind = %s, want novel", file.Decls[0].Kind)
	}
}

func TestParseTestFile(t *testing.T) {
	file, diagnostics := ParseTestFile("sample.test.narr", `namespace 长夜.tests
import 长夜.structure as s

test "结构约束" tags {vol01, structure} {
  let 第一回 = s.vol01.ch01

  forall chapter 章节 in chapters_in(s.vol01) where 章节.alias exists:
    assert 章节.title exists message "章节必须有标题"
    assert count(beat 事件 in beats(章节) where 事件.on_screen == true) >= 1

  exists beat 事件 in beats(第一回) where 事件 reveals s.秘密
}
`)
	if source.HasErrors(diagnostics) {
		t.Fatalf("ParseTestFile diagnostics:\n%s", formatDiagnosticsForTest(diagnostics))
	}
	if len(file.Tests) != 1 {
		t.Fatalf("len(Tests) = %d, want 1", len(file.Tests))
	}
	if len(file.Tests[0].Statements) != 3 {
		t.Fatalf("len(Statements) = %d, want 3", len(file.Tests[0].Statements))
	}
}

func TestParseExpressionPrecedence(t *testing.T) {
	expr, diagnostics := ParseExpression("<query>", `count(chapter 章节 in chapters_in(s.vol01) where 章节 advances s.王都线 or 章节 reveals s.秘密) >= 2`)
	if source.HasErrors(diagnostics) {
		t.Fatalf("ParseExpression diagnostics:\n%s", formatDiagnosticsForTest(diagnostics))
	}
	if expr.Kind != ast.ExprBinary || expr.Op != ">=" {
		t.Fatalf("expr = %#v, want top-level >= binary", expr)
	}
}

func TestParseErrorsContainLineAndColumn(t *testing.T) {
	_, diagnostics := ParseNarrFile("bad.narr", "namespace x\nchapter {\n")
	if !source.HasErrors(diagnostics) {
		t.Fatalf("ParseNarrFile returned no errors")
	}
	first := diagnostics[0]
	if first.File != "bad.narr" || first.Line == 0 || first.Column == 0 {
		t.Fatalf("diagnostic location = %#v, want file/line/column", first)
	}
}

func TestParseRecoversToNextTopLevelDeclaration(t *testing.T) {
	file, diagnostics := ParseNarrFile("bad.narr", "namespace x\nnonsense here\nfact 后续 = \"仍应解析\"\n")
	if !source.HasErrors(diagnostics) {
		t.Fatalf("ParseNarrFile returned no errors")
	}
	if len(file.Decls) != 1 {
		t.Fatalf("len(Decls) = %d, want 1 after recovery", len(file.Decls))
	}
	if file.Decls[0].Kind != ast.DeclFact || file.Decls[0].Name != "后续" {
		t.Fatalf("recovered decl = %#v, want fact 后续", file.Decls[0])
	}
}

func TestTestFileRejectsNarrDeclarations(t *testing.T) {
	_, diagnostics := ParseTestFile("bad.test.narr", "namespace x\nchapter vol01.ch01 {}\n")
	if !source.HasErrors(diagnostics) {
		t.Fatalf("ParseTestFile returned no errors for narr declaration")
	}
}

func formatDiagnosticsForTest(diagnostics []source.Diagnostic) string {
	out := ""
	for _, diagnostic := range diagnostics {
		out += diagnostic.File + ":" + strconv.Itoa(diagnostic.Line) + ":" + strconv.Itoa(diagnostic.Column) + " " + diagnostic.Code + " " + diagnostic.Message + "\n"
	}
	return out
}
