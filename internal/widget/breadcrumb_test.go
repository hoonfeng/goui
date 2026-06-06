package widget

import (
	"testing"

	"github.com/user/goui/internal/event"
	"github.com/user/goui/internal/layout"
)

// TestBreadcrumbClick 点链接项触发 OnClick，点末项(当前态)不触发。
func TestBreadcrumbClick(t *testing.T) {
	clicked := -1
	el := NewBreadcrumb(
		BreadcrumbItem{Label: "首页", OnClick: func() { clicked = 0 }},
		BreadcrumbItem{Label: "组件", OnClick: func() { clicked = 1 }},
		BreadcrumbItem{Label: "当前页"}, // 末项，不可点
	).CreateElement().(*BreadcrumbElement)
	el.Layout(&layout.LayoutContext{Constraints: layout.BoxConstraints{MaxWidth: 400, MaxHeight: 40}})

	// 点第一项「首页」
	r0 := el.itemRects[0]
	el.HandleEvent(event.NewMouseEvent(event.TypeMouseDown, r0.X+r0.Width/2, 10, event.ButtonLeft, event.ModNone))
	if clicked != 0 {
		t.Errorf("点「首页」后 clicked=%d, want 0", clicked)
	}

	// 点末项「当前页」(不可点)
	clicked = -1
	rl := el.itemRects[2]
	el.HandleEvent(event.NewMouseEvent(event.TypeMouseDown, rl.X+rl.Width/2, 10, event.ButtonLeft, event.ModNone))
	if clicked != -1 {
		t.Errorf("点末项(当前)不应触发 OnClick, clicked=%d", clicked)
	}
}

// TestBreadcrumbItemAtX 末项不参与命中(返回 -1)。
func TestBreadcrumbItemAtX(t *testing.T) {
	el := NewBreadcrumb(
		BreadcrumbItem{Label: "A"},
		BreadcrumbItem{Label: "B"},
	).CreateElement().(*BreadcrumbElement)
	el.Layout(&layout.LayoutContext{Constraints: layout.BoxConstraints{MaxWidth: 400, MaxHeight: 40}})
	if got := el.itemAtX(el.itemRects[0].X + 2); got != 0 {
		t.Errorf("命中第一项=%d, want 0", got)
	}
	if got := el.itemAtX(el.itemRects[1].X + 2); got != -1 {
		t.Errorf("末项不应命中=%d, want -1", got)
	}
}
