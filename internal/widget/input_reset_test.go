package widget

import "testing"

// 验证受控清空：ResetToken 变化时 InputElement.Update 把运行时文本重置为 Text。
func TestInputResetToken(t *testing.T) {
	in := NewTextarea("ph", 2, nil)
	el := &InputElement{input: in} // 直接构造，避免 Mount 依赖
	el.text = "user typed"
	el.cursorPos = 5

	// ResetToken 未变（都为 0）→ 不清空
	el.Update(in)
	if el.text != "user typed" {
		t.Errorf("ResetToken 未变不应清空，得 %q", el.text)
	}

	// ResetToken 递增 + Text 为空 → 清空运行时文本
	in.ResetToken = 1
	in.Text = ""
	el.Update(in)
	if el.text != "" {
		t.Errorf("ResetToken 变化应把 text 重置为 Text(\"\")，得 %q", el.text)
	}
	if el.cursorPos != 0 {
		t.Errorf("清空后 cursorPos 应为 0，得 %d", el.cursorPos)
	}
}
