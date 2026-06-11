package widget

import (
	"testing"

	"github.com/hoonfeng/goui/pkg/event"
	"github.com/hoonfeng/goui/internal/layout"
	"github.com/hoonfeng/goui/pkg/types"
)

func layoutTabs(t *Tabs) *TabsElement {
	el := t.CreateElement().(*TabsElement)
	el.Build()
	el.Layout(&layout.LayoutContext{Constraints: layout.BoxConstraints{MaxWidth: 400, MaxHeight: 300}})
	return el
}

// TestTabsSwitch 点标签切换 activeIndex 并触发 OnChange。
func TestTabsSwitch(t *testing.T) {
	prev := OnNeedsLayout
	OnNeedsLayout = func() {}
	defer func() { OnNeedsLayout = prev }()

	changed := -1
	black := types.ColorFromRGB(0, 0, 0)
	el := layoutTabs(NewTabs(
		TabPane{Label: "用户", Content: NewText("a", black)},
		TabPane{Label: "配置", Content: NewText("b", black)},
		TabPane{Label: "权限", Content: NewText("c", black)},
	).WithOnChange(func(i int) { changed = i }))

	if el.activeIndex != 0 {
		t.Fatalf("初始 activeIndex=%d, want 0", el.activeIndex)
	}
	r := el.labelRects[1] // 第二个标签
	el.HandleEvent(event.NewMouseEvent(event.TypeMouseDown, r.X+r.Width/2, 20, event.ButtonLeft, event.ModNone))
	if el.activeIndex != 1 {
		t.Errorf("点第二标签后 activeIndex=%d, want 1", el.activeIndex)
	}
	if changed != 1 {
		t.Errorf("OnChange=%d, want 1", changed)
	}
}

// TestTabsContentAreaNoSwitch 点内容区(标签栏之下)不切换。
func TestTabsContentAreaNoSwitch(t *testing.T) {
	black := types.ColorFromRGB(0, 0, 0)
	el := layoutTabs(NewTabs(
		TabPane{Label: "A", Content: NewText("a", black)},
		TabPane{Label: "B", Content: NewText("b", black)},
	))
	handled := el.HandleEvent(event.NewMouseEvent(event.TypeMouseDown, 50, tabsBarH+20, event.ButtonLeft, event.ModNone))
	if handled || el.activeIndex != 0 {
		t.Errorf("点内容区不应切换：handled=%v activeIndex=%d", handled, el.activeIndex)
	}
}
