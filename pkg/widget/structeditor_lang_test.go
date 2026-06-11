package widget

import (
	"strings"
	"testing"

	"github.com/hoonfeng/goui/pkg/event"
)

// TestEditorFormatGo 代码编辑器「格式化文档」命令用 gofmt 规范化 Go 源码。
func TestEditorFormatGo(t *testing.T) {
	ce := NewCodeEditor("go", "package main\nfunc  foo( ){\nx:=1\n_=x\n}\n").WithSize(400, 300)
	e := ce.CreateElement().(*CodeEditorElement)
	e.runCommand("format")
	got := e.text()
	if !strings.Contains(got, "func foo() {") {
		t.Errorf("gofmt 后应规范化为 'func foo() {'，得:\n%s", got)
	}
}

// TestGoParseDiag 诊断：内置 Go 解析能否把 Go 源码拆成表格数据（imports/globals/consts/types/subs）。
func TestGoParseDiag(t *testing.T) {
	src := `package main

import "fmt"

type Point struct {
	X, Y int
}

const Pi = 3.14

var counter int

func add(a, b int) int {
	sum := a + b
	return sum
}

func main() {
	fmt.Println(add(1, 2))
}`
	p, err := goLangProvider{}.Parse(src)
	if err != nil {
		t.Fatalf("parse err: %v", err)
	}
	t.Logf("Imports=%v Globals=%d Consts=%d Types=%d Subs=%d",
		p.Imports, len(p.Globals), len(p.Consts), len(p.Types), len(p.Subs))
	for _, s := range p.Subs {
		t.Logf("  sub %q recv=%q params=%d locals=%d returns=%d", s.Name, s.Recv, len(s.Params), len(s.Locals), len(s.Returns))
	}
	for _, ty := range p.Types {
		t.Logf("  type %q kind=%s fields=%d", ty.Name, ty.Kind, len(ty.Fields))
	}
	if len(p.Subs) == 0 {
		t.Error("解析后无任何子程序——Go 解析器坏了")
	}
}

// TestStructEditorFuncNameEdit 函数名可编辑：点函数声明行(func:i 区段 col0)进入编辑、键入改名。
func TestStructEditorFuncNameEdit(t *testing.T) {
	p := &SEProgram{Subs: []SESub{{Name: "foo"}}}
	e := NewStructEditor(p).CreateElement().(*StructEditorElement)
	e.beginEdit("func:0", 0, 0)
	if !e.editing || e.selSection != "func:0" {
		t.Fatalf("点函数名应进入编辑，实际 editing=%v sec=%q", e.editing, e.selSection)
	}
	e.editInsert('2')
	if p.Subs[0].Name != "foo2" {
		t.Errorf("编辑函数名应变 foo2，实际 %q", p.Subs[0].Name)
	}
}

// TestGoMultiNameValues 一行多个常量/变量：每个名字配自己的值（不再统一取第一个）。
func TestGoMultiNameValues(t *testing.T) {
	src := `package main

const a, b = 1, 2

var x, y = 10, 20`
	p, err := goLangProvider{}.Parse(src)
	if err != nil {
		t.Fatal(err)
	}
	if len(p.Consts) != 2 || p.Consts[0].Array != "1" || p.Consts[1].Array != "2" {
		t.Errorf("常量 a=1,b=2，实际 %+v", p.Consts)
	}
	if len(p.Globals) != 2 || p.Globals[0].Ref != "10" || p.Globals[1].Ref != "20" {
		t.Errorf("变量 x=10,y=20，实际 %+v", p.Globals)
	}
}

// TestGoLocalConst 函数内的局部常量（含 const w,h=1200,760 快捷声明）被解析进局部变量表。
func TestGoLocalConst(t *testing.T) {
	src := `package main

func draw() {
	const w, h = 1200, 760
	const dpi = 96
	x := 5
	_ = x
}`
	p, err := goLangProvider{}.Parse(src)
	if err != nil {
		t.Fatal(err)
	}
	if len(p.Subs) != 1 {
		t.Fatalf("应 1 个函数，实际 %d", len(p.Subs))
	}
	got := map[string]string{}
	for _, l := range p.Subs[0].Locals {
		got[l.Name] = l.Type
	}
	for _, n := range []string{"w", "h", "dpi", "x"} {
		if _, ok := got[n]; !ok {
			t.Errorf("局部声明 %q 未解析，locals=%v", n, got)
		}
	}
	if got["w"] != "int" {
		t.Errorf("w 类型应推导为 int，实际 %q", got["w"])
	}
}

// TestStructEditorTypePopup 点类型「成员/底层」列开字段表浮窗、Esc 关闭。
func TestStructEditorTypePopup(t *testing.T) {
	p := &SEProgram{Types: []SEType{{Name: "Point", Fields: []SEVar{{Name: "X", Type: "int"}}}}}
	e := NewStructEditor(p).CreateElement().(*StructEditorElement)
	e.beginEdit("type:0", 0, 2) // 点成员/底层列 → 开浮窗（不进文本编辑）
	if e.popupType != 0 {
		t.Fatalf("点类型成员列应开字段表浮窗(popupType=0)，实际 %d", e.popupType)
	}
	if e.editing {
		t.Error("开浮窗时不应进入单元格编辑")
	}
	e.handleKey(event.KeyEvent{Key: event.KeyEscape}) // Esc 关
	if e.popupType != -1 {
		t.Errorf("Esc 应关浮窗(popupType=-1)，实际 %d", e.popupType)
	}
}

// TestRustImplMethods 验证 Rust impl 块内方法被提取（Recv=类型），且 impl 块闭合后 inImpl 复位，
// 使其后的顶级 fn 仍被识别为顶级函数（补全 impl 提取前的 bug：inImpl 不复位会吞掉后续顶级 fn）。
func TestRustImplMethods(t *testing.T) {
	src := `struct Point { x: i32, y: i32 }

impl Point {
	fn new(x: i32, y: i32) -> Point {
		Point { x, y }
	}
	fn dist(&self) -> f64 {
		0.0
	}
}

fn main() {
	let p = Point::new(1, 2);
}`
	p, err := rustLangProvider{}.Parse(src)
	if err != nil {
		t.Fatal(err)
	}
	var methods, topFns []string
	for _, s := range p.Subs {
		if s.Recv != "" {
			methods = append(methods, s.Name)
		} else {
			topFns = append(topFns, s.Name)
		}
	}
	if len(methods) != 2 || methods[0] != "new" || methods[1] != "dist" {
		t.Errorf("impl 方法应为 [new dist]（Recv=Point），得 %v", methods)
	}
	if len(topFns) != 1 || topFns[0] != "main" {
		t.Errorf("顶级函数应为 [main]（impl 闭合后 inImpl 须复位），得 %v", topFns)
	}
}
