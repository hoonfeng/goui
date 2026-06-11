package widget

import "testing"

// TestInputFirstCharNotOverwritten 复现并守护「聚焦后首字被选中、次字覆盖它」的回归：
// 点击聚焦会把 selAnchor 置成光标位（空选区），插入第一字后光标右移，若不清 selAnchor 就
// 变成覆盖首字的真选区，下一字的 deleteSelection 会把首字删掉。insertText 插入后须清选区。
func TestInputFirstCharNotOverwritten(t *testing.T) {
	in := NewInput("", nil)
	e := in.CreateElement().(*InputElement)
	e.focused = true
	e.cursorPos, e.selAnchor = 0, 0 // 模拟点击聚焦：空选区(anchor==cursor)

	e.insertText("中")
	if e.text != "中" {
		t.Fatalf("首字后 text=%q，期望「中」", e.text)
	}
	if e.hasSelection() {
		t.Errorf("插入后不应有选区，但 selAnchor=%d cursor=%d", e.selAnchor, e.cursorPos)
	}

	e.insertText("文")
	if e.text != "中文" {
		t.Errorf("次字后 text=%q，期望「中文」（首字被覆盖=bug 未修）", e.text)
	}

	// 仍要保留「选中后输入替换选区」的正常行为：选中全部再输入应替换。
	e.selAnchor, e.cursorPos = 0, 2 // 选中「中文」
	e.insertText("X")
	if e.text != "X" {
		t.Errorf("选区上输入应替换：text=%q，期望「X」", e.text)
	}
}
