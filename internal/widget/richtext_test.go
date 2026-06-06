package widget

import (
	"testing"

	"github.com/user/goui/internal/types"
)

// richAllText 把富文本所有块的纯文本拼起来（\n 分块），供测试。
func richAllText(e *RichTextElement) string {
	var s string
	for i, b := range e.blocks {
		if i > 0 {
			s += "\n"
		}
		for _, sp := range b.Spans {
			s += sp.Text
		}
	}
	return s
}

// TestRichUndoRedo 验证富文本撤销/重做（插入→撤销回退→重做恢复）。
func TestRichUndoRedo(t *testing.T) {
	rt := NewRichText("ab")
	e := rt.CreateElement().(*RichTextElement)
	e.cursor = richPos{0, blockLen(e.blocks[0])}
	e.anchor = e.cursor
	e.insertText("X")
	if txt := richAllText(e); txt != "abX" {
		t.Fatalf("插入后 %q，期望 abX", txt)
	}
	e.undo()
	if txt := richAllText(e); txt != "ab" {
		t.Errorf("撤销后 %q，期望 ab", txt)
	}
	e.redo()
	if txt := richAllText(e); txt != "abX" {
		t.Errorf("重做后 %q，期望 abX", txt)
	}
	e.undo()
	e.undo() // 撤销到底再撤销不崩
	if txt := richAllText(e); txt != "ab" {
		t.Errorf("撤销到底 %q，期望 ab", txt)
	}
}

// TestRichToggleBoldOnSelection 验证选中文字加粗后 span 数据正确。
func TestRichToggleBoldOnSelection(t *testing.T) {
	rt := NewRichText("hello world")
	el := rt.CreateElement().(*RichTextElement)
	el.anchor = richPos{0, 0}
	el.cursor = richPos{0, 5} // 选中 "hello"
	if !el.hasSelection() {
		t.Fatal("expected selection")
	}
	el.toggleStyle("bold")

	b := el.blocks[0]
	pos := 0
	helloBold, worldBold := false, false
	for _, sp := range b.Spans {
		rs := []rune(sp.Text)
		for i := range rs {
			abs := pos + i
			if abs < 5 {
				helloBold = sp.Bold
			} else {
				worldBold = worldBold || sp.Bold
			}
		}
		pos += len(rs)
	}
	if !helloBold {
		t.Errorf("hello should be bold; spans=%+v", b.Spans)
	}
	if worldBold {
		t.Errorf("' world' should NOT be bold; spans=%+v", b.Spans)
	}
}

// TestRichToggleItalic 验证斜体应用。
func TestRichToggleItalic(t *testing.T) {
	rt := NewRichText("abcdef")
	el := rt.CreateElement().(*RichTextElement)
	el.anchor = richPos{0, 2}
	el.cursor = richPos{0, 4} // 选 "cd"
	el.toggleStyle("italic")
	b := el.blocks[0]
	found := false
	for _, sp := range b.Spans {
		if sp.Text == "cd" && sp.Italic {
			found = true
		}
	}
	if !found {
		t.Errorf("'cd' should be italic; spans=%+v", b.Spans)
	}
}

// TestRichColorNoSelection 验证无选区设色后输入带色，且后续输入靠继承延续同色。
func TestRichColorNoSelection(t *testing.T) {
	rt := NewRichText("")
	el := rt.CreateElement().(*RichTextElement)
	blue := types.ColorFromRGB(0, 100, 255)
	el.applyColorPick(&blue) // 无选区 → 设待应用色（不污染为永久画笔）
	el.insertText("a")       // a 用 pending 色
	el.insertText("b")       // b 继承前一个字符 a → 仍蓝
	for _, sp := range el.blocks[0].Spans {
		if sp.Text == "" {
			continue
		}
		if sp.Color == nil || *sp.Color != blue {
			t.Errorf("span %q 应为蓝色，实际 %v", sp.Text, sp.Color)
		}
	}
}

// TestRichColorSelectionThenElsewhere 验证选区设色只染选区；之后到别处（黑字后）输入不带色。
func TestRichColorSelectionThenElsewhere(t *testing.T) {
	rt := NewRichText("hello world")
	el := rt.CreateElement().(*RichTextElement)
	red := types.ColorFromRGB(255, 0, 0)
	el.anchor = richPos{0, 0}
	el.cursor = richPos{0, 5} // 选 "hello"
	el.applyColorPick(&red)   // 有选区 → 染选区
	// 把光标移到 " world" 里的黑字后（offset 8），输入应为默认色（继承黑字）
	el.cursor = richPos{0, 8}
	el.anchor = el.cursor
	el.insertText("X")
	for _, sp := range el.blocks[0].Spans {
		for _, r := range sp.Text {
			if r == 'X' && sp.Color != nil {
				t.Errorf("黑字处输入的 X 不应带色，实际 %v", sp.Color)
			}
		}
	}
}

// TestRichFontSize 验证选区设字号写入 span.Size。
func TestRichFontSize(t *testing.T) {
	rt := NewRichText("abcdef")
	el := rt.CreateElement().(*RichTextElement)
	el.anchor = richPos{0, 2}
	el.cursor = richPos{0, 4} // 选 "cd"
	el.applyFontSize(24)
	found := false
	for _, sp := range el.blocks[0].Spans {
		if sp.Text == "cd" && sp.Size == 24 {
			found = true
		}
	}
	if !found {
		t.Errorf("'cd' 应为 24 号字；spans=%+v", el.blocks[0].Spans)
	}
}

// TestRichStrike 验证删除线应用。
func TestRichStrike(t *testing.T) {
	rt := NewRichText("abcdef")
	el := rt.CreateElement().(*RichTextElement)
	el.anchor = richPos{0, 0}
	el.cursor = richPos{0, 3} // 选 "abc"
	el.toggleStyle("strike")
	for _, sp := range el.blocks[0].Spans {
		if sp.Text == "abc" && !sp.Strike {
			t.Errorf("'abc' 应有删除线；spans=%+v", el.blocks[0].Spans)
		}
	}
}

// TestInputContextMenuDefault 验证默认右键菜单项及按选区/文本置灰。
func TestInputContextMenuDefault(t *testing.T) {
	oldR, oldW := ClipboardRead, ClipboardWrite
	ClipboardRead = func() string { return "x" }
	ClipboardWrite = func(string) {}
	defer func() { ClipboardRead, ClipboardWrite = oldR, oldW }()

	in := NewTextarea("", 3, nil)
	el := in.CreateElement().(*InputElement)
	el.text = "hello"
	items := el.defaultContextItems()
	if len(items) != 4 {
		t.Fatalf("应有 4 个菜单项，实际 %d", len(items))
	}
	if items[1].Label != "复制" || items[1].Enabled {
		t.Errorf("无选区时复制应禁用；items=%+v", items)
	}
	if items[3].Label != "全选" || !items[3].Enabled {
		t.Errorf("有文本时全选应启用")
	}
	el.selAnchor = 0
	el.cursorPos = 5
	if !el.defaultContextItems()[1].Enabled {
		t.Errorf("有选区时复制应启用")
	}
}

// TestInputContextMenuConfig 验证禁用/自定义右键菜单配置。
func TestInputContextMenuConfig(t *testing.T) {
	in := NewInput("", nil).WithoutContextMenu()
	if !in.ContextMenuDisabled {
		t.Error("WithoutContextMenu 应禁用右键菜单")
	}
	in2 := NewInput("", nil).WithContextMenu(MenuItem{Label: "自定义"})
	if len(in2.ContextMenuItems) != 1 || in2.ContextMenuItems[0].Label != "自定义" {
		t.Error("WithContextMenu 应设置自定义项")
	}
}

// TestRichCurSize 验证工具栏「字号」显示的当前字号（块默认 / 选区设定）。
func TestRichCurSize(t *testing.T) {
	rt := NewRichText("hello")
	el := rt.CreateElement().(*RichTextElement)
	if el.curSize() != 15 {
		t.Errorf("正文默认字号应为 15，实际 %g", el.curSize())
	}
	el.anchor = richPos{0, 0}
	el.cursor = richPos{0, 5}
	el.applyFontSize(24)
	if el.curSize() != 24 {
		t.Errorf("设 24 后当前字号应为 24，实际 %g", el.curSize())
	}
}

// TestInputTextDragMove 验证多行/单行文本框把选中文本拖到别处的移动逻辑。
func TestInputTextDragMove(t *testing.T) {
	in := NewInput("", nil)
	el := in.CreateElement().(*InputElement)
	el.text = "0123456789"
	el.selAnchor = 2
	el.cursorPos = 4 // 选 "23"
	el.dropPos = 8
	el.textDragging = true
	el.finishTextDrag()
	if el.text != "0145672389" {
		t.Errorf("拖放移动后应为 0145672389，实际 %q", el.text)
	}
	if lo, hi := el.selectionRange(); lo != 6 || hi != 8 {
		t.Errorf("应选中移动后的文本 [6,8)，实际 [%d,%d)", lo, hi)
	}
	// drop 落在选区内 → 取消，不动
	el2 := in.CreateElement().(*InputElement)
	el2.text = "hello"
	el2.selAnchor = 0
	el2.cursorPos = 5
	el2.dropPos = 3
	el2.textDragging = true
	el2.finishTextDrag()
	if el2.text != "hello" {
		t.Errorf("drop 落在选区内应取消，文本应不变；实际 %q", el2.text)
	}
}

// TestRichTextDragMove 验证富文本拖放移动（纯文本，扁平偏移校正目标位置）。
func TestRichTextDragMove(t *testing.T) {
	rt := NewRichText("hello world")
	el := rt.CreateElement().(*RichTextElement)
	el.anchor = richPos{0, 0}
	el.cursor = richPos{0, 5} // 选 "hello"
	el.dropPos = richPos{0, 11}
	el.textDragging = true
	el.finishTextDragRich()
	got := string(blockText(el.blocks[0]))
	if got != " worldhello" {
		t.Errorf("拖放移动后应为 ' worldhello'，实际 %q", got)
	}
}

// TestRichRangeText 验证富文本跨块纯文本提取（供复制/剪切）。
func TestRichRangeText(t *testing.T) {
	rt := NewRichText("")
	el := rt.CreateElement().(*RichTextElement)
	el.blocks = []RichBlock{
		{Type: "p", Spans: []RichSpan{{Text: "hello"}}},
		{Type: "p", Spans: []RichSpan{{Text: "world"}}},
	}
	txt := el.richRangeText(richPos{0, 2}, richPos{1, 2}) // "llo" + 换行 + "wo"
	if txt != "llo\nwo" {
		t.Errorf("跨块纯文本应为 'llo\\nwo'，实际 %q", txt)
	}
}

// TestRichImageAtomicCursor 验证图片块当原子字符：blockLen==1，光标可停在图片前/后并跨过。
func TestRichImageAtomicCursor(t *testing.T) {
	rt := NewRichText("hello")
	el := rt.CreateElement().(*RichTextElement)
	el.cursor = richPos{0, 5}
	el.anchor = el.cursor
	el.insertImage() // 测试环境 OpenFileDialog==nil → 插入占位图片块（夹在两个文本段之间）
	imgIdx := -1
	for i, b := range el.blocks {
		if b.Type == "image" {
			imgIdx = i
		}
	}
	if imgIdx < 0 {
		t.Fatalf("未插入图片块；blocks=%+v", el.blocks)
	}
	if blockLen(el.blocks[imgIdx]) != 1 {
		t.Errorf("图片块 blockLen 应为 1，实际 %d", blockLen(el.blocks[imgIdx]))
	}
	// 从图片后的段落开头向左：先停图片后(offset 1)，再停图片前(offset 0)
	el.cursor = richPos{imgIdx + 1, 0}
	el.moveCursor(-1)
	if el.cursor.block != imgIdx || el.cursor.offset != 1 {
		t.Errorf("应停在图片后(块%d,1)，实际 %+v", imgIdx, el.cursor)
	}
	el.moveCursor(-1)
	if el.cursor.block != imgIdx || el.cursor.offset != 0 {
		t.Errorf("应停在图片前(块%d,0)，实际 %+v", imgIdx, el.cursor)
	}
	// 在图片块上打字应自动移出到相邻文本块
	el.cursor = richPos{imgIdx, 1}
	el.anchor = el.cursor
	el.insertText("X")
	if el.blocks[el.cursor.block].Type == "image" {
		t.Error("在图片块打字后，光标不应仍在图片块")
	}
}

// TestRichSetAlign 验证段落对齐写入。
func TestRichSetAlign(t *testing.T) {
	rt := NewRichText("hi")
	el := rt.CreateElement().(*RichTextElement)
	el.anchor = richPos{0, 0}
	el.cursor = richPos{0, 2}
	el.setAlign("center")
	if el.blocks[0].Align != "center" {
		t.Errorf("对齐应为 center，实际 %q", el.blocks[0].Align)
	}
}
