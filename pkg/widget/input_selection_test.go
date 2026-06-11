package widget

import (
	"testing"

	"github.com/hoonfeng/goui/pkg/event"
)

// TestInputTypeReplacesSelection 选中文字后输入字符应替换选区。
func TestInputTypeReplacesSelection(t *testing.T) {
	el := newLaidOutInput(t)
	el.Focus()
	el.SetText("hello world")
	// 选中前 5 个字符 "hello"
	el.selAnchor = 0
	el.cursorPos = 5
	if !el.hasSelection() || el.selectedText() != "hello" {
		t.Fatalf("选区文本 = %q，期望 hello", el.selectedText())
	}

	ev := event.NewKeyEvent(event.TypeKeyChar, uint32('X'), "X", event.ModNone)
	ev.Char = 'X'
	el.HandleEvent(ev)

	if el.Text() != "X world" {
		t.Fatalf("替换选区后 = %q，期望 'X world'", el.Text())
	}
	if el.hasSelection() {
		t.Error("输入后选区应被清除")
	}
}

// TestInputBackspaceDeletesSelection 有选区时退格应删除整个选区。
func TestInputBackspaceDeletesSelection(t *testing.T) {
	el := newLaidOutInput(t)
	el.Focus()
	el.SetText("hello")
	el.selAnchor = 1
	el.cursorPos = 4 // 选中 "ell"

	back := event.NewKeyEvent(event.TypeKeyDown, 0, "Backspace", event.ModNone)
	el.HandleEvent(back)

	if el.Text() != "ho" {
		t.Fatalf("删除选区后 = %q，期望 'ho'", el.Text())
	}
	if el.cursorPos != 1 {
		t.Errorf("删除选区后光标应在 1，实际 %d", el.cursorPos)
	}
}

// TestInputClipboardShortcuts 验证 Ctrl+A/C/V/X 全选、复制、粘贴、剪切。
func TestInputClipboardShortcuts(t *testing.T) {
	// mock 剪贴板（widget 全局钩子）
	var clip string
	oldR, oldW := ClipboardRead, ClipboardWrite
	ClipboardRead = func() string { return clip }
	ClipboardWrite = func(s string) { clip = s }
	defer func() { ClipboardRead, ClipboardWrite = oldR, oldW }()

	el := newLaidOutInput(t)
	el.Focus()
	el.SetText("hello")

	ctrlA := event.NewKeyEvent(event.TypeKeyDown, 0, "A", event.ModCtrl)
	ctrlC := event.NewKeyEvent(event.TypeKeyDown, 0, "C", event.ModCtrl)
	ctrlV := event.NewKeyEvent(event.TypeKeyDown, 0, "V", event.ModCtrl)
	ctrlX := event.NewKeyEvent(event.TypeKeyDown, 0, "X", event.ModCtrl)

	// 全选 + 复制
	el.HandleEvent(ctrlA)
	if el.selectedText() != "hello" {
		t.Fatalf("Ctrl+A 应全选，selectedText = %q", el.selectedText())
	}
	el.HandleEvent(ctrlC)
	if clip != "hello" {
		t.Fatalf("Ctrl+C 后剪贴板 = %q，期望 hello", clip)
	}

	// 光标移到末尾，粘贴（追加）
	el.clearSelection()
	el.cursorPos = len([]rune(el.Text()))
	el.HandleEvent(ctrlV)
	if el.Text() != "hellohello" {
		t.Fatalf("Ctrl+V 后 = %q，期望 hellohello", el.Text())
	}

	// 全选后剪切
	el.HandleEvent(ctrlA)
	el.HandleEvent(ctrlX)
	if el.Text() != "" {
		t.Fatalf("Ctrl+X 后文本应为空，实际 = %q", el.Text())
	}
	if clip != "hellohello" {
		t.Fatalf("Ctrl+X 后剪贴板 = %q，期望 hellohello", clip)
	}
}

// TestInputMouseDragSelects 验证鼠标按下并拖动能形成选区，松开后退出选择态。
func TestInputMouseDragSelects(t *testing.T) {
	el := newLaidOutInput(t)
	el.Focus()
	el.SetText("hello world")
	pos := el.Offset() // 无父、未 SetPosition → (0,0)

	down := event.NewMouseEvent(event.TypeMouseDown, pos.X+8, pos.Y+10, event.ButtonLeft, event.ModNone)
	el.HandleEvent(down)
	if !el.selecting {
		t.Fatal("MouseDown 后应进入选择状态")
	}

	// 拖到远右，光标应落到文本末尾，与锚点 0 形成选区
	move := event.NewMouseEvent(event.TypeMouseMove, pos.X+800, pos.Y+10, event.ButtonLeft, event.ModNone)
	el.HandleEvent(move)
	if !el.hasSelection() {
		t.Error("拖动后应产生选区")
	}

	up := event.NewMouseEvent(event.TypeMouseUp, pos.X+800, pos.Y+10, event.ButtonLeft, event.ModNone)
	el.HandleEvent(up)
	if el.selecting {
		t.Error("MouseUp 后应退出选择状态")
	}
}

// TestInputCopyWithoutSelectionNoop 无选区时复制不应写剪贴板。
func TestInputCopyWithoutSelectionNoop(t *testing.T) {
	called := false
	oldW := ClipboardWrite
	ClipboardWrite = func(s string) { called = true }
	defer func() { ClipboardWrite = oldW }()

	el := newLaidOutInput(t)
	el.Focus()
	el.SetText("hello") // SetText 清除选区

	el.HandleEvent(event.NewKeyEvent(event.TypeKeyDown, 0, "C", event.ModCtrl))
	if called {
		t.Error("无选区时 Ctrl+C 不应写入剪贴板")
	}
}
