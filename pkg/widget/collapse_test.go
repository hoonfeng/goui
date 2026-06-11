package widget

import (
	"testing"

	"github.com/hoonfeng/goui/pkg/event"
	"github.com/hoonfeng/goui/internal/layout"
	"github.com/hoonfeng/goui/pkg/types"
)

func layoutCollapse(c *Collapse) *CollapseElement {
	el := c.CreateElement().(*CollapseElement)
	el.Build()
	el.Layout(&layout.LayoutContext{Constraints: layout.BoxConstraints{MaxWidth: 400, MaxHeight: 500}})
	return el
}

// clickHeader 点击第 i 个标题(按初始全收起时的位置；测试中 OnNeedsLayout 被 mock 不重排)。
func clickHeader(el *CollapseElement, i int) {
	y := float64(i)*collapseHeaderH + collapseHeaderH/2
	el.HandleEvent(event.NewMouseEvent(event.TypeMouseDown, 20, y, event.ButtonLeft, event.ModNone))
}

func twoItems() *Collapse {
	black := types.ColorFromRGB(0, 0, 0)
	return NewCollapse(
		CollapseItem{Title: "A", Content: NewText("a", black)},
		CollapseItem{Title: "B", Content: NewText("b", black)},
	)
}

// TestCollapseToggle 点标题展开，再点收起。
func TestCollapseToggle(t *testing.T) {
	prev := OnNeedsLayout
	OnNeedsLayout = func() {}
	defer func() { OnNeedsLayout = prev }()

	el := layoutCollapse(twoItems())
	if el.expanded[0] {
		t.Fatal("初始应全收起")
	}
	clickHeader(el, 0)
	if !el.expanded[0] {
		t.Error("点 header0 应展开")
	}
	clickHeader(el, 0)
	if el.expanded[0] {
		t.Error("再点 header0 应收起")
	}
}

// TestCollapseAccordion 手风琴模式：展开一个会关闭其他。
func TestCollapseAccordion(t *testing.T) {
	prev := OnNeedsLayout
	OnNeedsLayout = func() {}
	defer func() { OnNeedsLayout = prev }()

	el := layoutCollapse(twoItems().WithAccordion(true))
	clickHeader(el, 0)
	clickHeader(el, 1)
	if el.expanded[0] {
		t.Error("手风琴：展开 1 应关闭 0")
	}
	if !el.expanded[1] {
		t.Error("1 应展开")
	}
}

// TestCollapseDefaultExpanded 默认展开指定项。
func TestCollapseDefaultExpanded(t *testing.T) {
	el := layoutCollapse(twoItems().WithExpanded(1))
	if !el.expanded[1] || el.expanded[0] {
		t.Errorf("默认展开应为 [false, true], got %v", el.expanded)
	}
}
