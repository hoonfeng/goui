package widget

import (
	"testing"

	"github.com/user/goui/internal/i18n"
	"github.com/user/goui/internal/layout"
)

// TestFlexRowRTLMirror 验证 RTL 语言下 Row 子项从右到左镜像排列。
func TestFlexRowRTLMirror(t *testing.T) {
	i18n.SetLocale("ar") // 阿拉伯语：RTL
	defer i18n.SetLocale("en")

	row := NewRow([]Widget{
		&Container{Width: 30, Height: 10},
		&Container{Width: 20, Height: 10},
	})
	el := row.CreateElement().(*FlexElement)
	el.Mount(nil, 0)
	el.Build()
	cons := layout.LayoutContext{Constraints: layout.BoxConstraints{MaxWidth: 200, MaxHeight: 100}}
	el.Layout(&cons)

	// RTL：内容宽 50，第一个(30)靠右 → X=20，第二个(20) → X=0
	if c0, c1 := el.Children()[0].Position().X, el.Children()[1].Position().X; c0 != 20 || c1 != 0 {
		t.Errorf("RTL positions c0=%.0f c1=%.0f, want 20,0", c0, c1)
	}

	// 切回 LTR：恢复从左到右
	i18n.SetLocale("en")
	el.Layout(&cons)
	if c0, c1 := el.Children()[0].Position().X, el.Children()[1].Position().X; c0 != 0 || c1 != 30 {
		t.Errorf("LTR positions c0=%.0f c1=%.0f, want 0,30", c0, c1)
	}
}
