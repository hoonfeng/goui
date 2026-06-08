package vterm

import (
	"strings"
	"testing"
)

func rowText(t *Terminal, row int) string {
	cols, _ := t.Size()
	b := make([]rune, 0, cols)
	for c := 0; c < cols; c++ {
		b = append(b, t.Cell(row, c).Ch)
	}
	return strings.TrimRight(string(b), " ")
}

func TestPrintAndNewline(t *testing.T) {
	vt := New(20, 5)
	vt.Write([]byte("hello\r\nworld"))
	if got := rowText(vt, 0); got != "hello" {
		t.Errorf("row0=%q 期望 hello", got)
	}
	if got := rowText(vt, 1); got != "world" {
		t.Errorf("row1=%q 期望 world", got)
	}
	if cx, cy := vt.Cursor(); cx != 5 || cy != 1 {
		t.Errorf("光标=(%d,%d) 期望 (5,1)", cx, cy)
	}
}

func TestSGRColor(t *testing.T) {
	vt := New(20, 3)
	vt.Write([]byte("\x1b[31mR\x1b[0mN")) // 红 R，复位，默认 N
	if c := vt.Cell(0, 0); c.Ch != 'R' || c.FG.Default || c.FG.R != 205 {
		t.Errorf("R 格=%+v 期望红色 R", c)
	}
	if c := vt.Cell(0, 1); c.Ch != 'N' || !c.FG.Default {
		t.Errorf("N 格=%+v 期望默认色 N", c)
	}
}

func TestBoldAndBg(t *testing.T) {
	vt := New(20, 3)
	vt.Write([]byte("\x1b[1;44mB")) // 粗体 + 蓝底
	c := vt.Cell(0, 0)
	if c.Ch != 'B' || !c.Bold || c.BG.Default {
		t.Errorf("格=%+v 期望粗体+蓝底", c)
	}
}

func TestCursorPositionAndErase(t *testing.T) {
	vt := New(10, 3)
	vt.Write([]byte("ABCDE"))
	vt.Write([]byte("\x1b[1;1H")) // 光标回左上
	if cx, cy := vt.Cursor(); cx != 0 || cy != 0 {
		t.Errorf("home 光标=(%d,%d)", cx, cy)
	}
	vt.Write([]byte("\x1b[K")) // 擦到行末 → 清掉 ABCDE
	if got := rowText(vt, 0); got != "" {
		t.Errorf("擦除后 row0=%q 期望空", got)
	}
}

func TestScrollback(t *testing.T) {
	vt := New(10, 2)
	vt.Write([]byte("a\r\nb\r\nc")) // a/b 后 c 把 a 挤出顶部
	if n := len(vt.Scrollback()); n != 1 {
		t.Errorf("scrollback=%d 期望 1", n)
	}
	if got := rowText(vt, 0); got != "b" {
		t.Errorf("row0=%q 期望 b", got)
	}
	if got := rowText(vt, 1); got != "c" {
		t.Errorf("row1=%q 期望 c", got)
	}
}

func Test256Color(t *testing.T) {
	vt := New(10, 2)
	vt.Write([]byte("\x1b[38;5;196mX")) // 256 色 196 ≈ 纯红
	c := vt.Cell(0, 0)
	if c.Ch != 'X' || c.FG.Default || c.FG.R != 255 {
		t.Errorf("256 色格=%+v 期望红 X", c)
	}
}

func TestResize(t *testing.T) {
	vt := New(10, 3)
	vt.Write([]byte("hello"))
	vt.Resize(20, 5)
	if cols, rows := vt.Size(); cols != 20 || rows != 5 {
		t.Errorf("resize 后尺寸=(%d,%d)", cols, rows)
	}
	if got := rowText(vt, 0); got != "hello" { // 内容保留
		t.Errorf("resize 后 row0=%q 期望 hello", got)
	}
}
