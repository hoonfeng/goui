package widget

import "testing"

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
