package widget

import (
	"testing"

	"github.com/hoonfeng/goui/internal/layout"
	"github.com/hoonfeng/goui/pkg/types"
)

// TestMessageBoxIcon 类型图标固定 28×28。
func TestMessageBoxIcon(t *testing.T) {
	el := (&messageBoxIcon{color: types.ColorFromRGB(230, 162, 60), ch: "!"}).CreateElement().(*messageBoxIconElement)
	el.Layout(&layout.LayoutContext{Constraints: layout.BoxConstraints{MaxWidth: 100, MaxHeight: 100}})
	if el.size.Width != 28 || el.size.Height != 28 {
		t.Errorf("icon size=%.0fx%.0f, want 28x28", el.size.Width, el.size.Height)
	}
}

// TestMessageBoxBody 内容区构造非空(图标 + 文字横排)。
func TestMessageBoxBody(t *testing.T) {
	if MessageBoxBodyPreview("确定删除吗？", MsgWarning) == nil {
		t.Error("messageBoxBody 不应为 nil")
	}
}
