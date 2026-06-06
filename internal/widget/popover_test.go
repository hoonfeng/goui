package widget

import (
	"testing"

	"github.com/user/goui/internal/event"
	"github.com/user/goui/internal/layout"
)

func newPopoverElement(p *Popover) *PopoverElement {
	e := p.CreateElement().(*PopoverElement)
	e.Build()
	e.Layout(&layout.LayoutContext{Constraints: layout.BoxConstraints{MaxWidth: 500, MaxHeight: 500}})
	return e
}

func TestPopoverClickShowHide(t *testing.T) {
	overlayEntries = nil
	e := newPopoverElement(NewPopover(NewTag("触发")).WithTitle("标题").WithContent("内容"))
	e.HandleEvent(mouse(event.TypeMouseDown))
	if len(activeOverlays()) != 1 {
		t.Fatalf("click 应显示弹出框, got %d", len(activeOverlays()))
	}
	e.HandleEvent(mouse(event.TypeMouseDown))
	if len(activeOverlays()) != 0 {
		t.Fatalf("再次 click 应关闭, got %d", len(activeOverlays()))
	}
}

func TestPopoverHoverTrigger(t *testing.T) {
	overlayEntries = nil
	e := newPopoverElement(NewPopover(NewTag("触发")).WithContent("hover 内容").WithTrigger(TooltipHover))
	e.HandleEvent(mouse(event.TypeMouseEnter))
	if len(activeOverlays()) != 1 {
		t.Fatalf("hover 应显示, got %d", len(activeOverlays()))
	}
	e.HandleEvent(mouse(event.TypeMouseLeave))
	if len(activeOverlays()) != 0 {
		t.Fatalf("hover leave 应隐藏, got %d", len(activeOverlays()))
	}
}

func TestPopoverPanelLayout(t *testing.T) {
	panel := &popoverPanel{title: "标题", content: "一段较长的弹出框内容文字用于测试换行排版", width: 200}
	pe := panel.CreateElement().(*popoverPanelElement)
	pe.Build()
	res := pe.Layout(&layout.LayoutContext{Constraints: layout.BoxConstraints{MaxWidth: 1000, MaxHeight: 1000}})
	if res.Size.Width != 200 {
		t.Errorf("panel 宽度应受 width 控制=200, got %.0f", res.Size.Width)
	}
	if res.Size.Height <= 0 {
		t.Errorf("panel 高度应 > 0, got %.0f", res.Size.Height)
	}
}

func TestPopoverBodyWidget(t *testing.T) {
	overlayEntries = nil
	e := newPopoverElement(NewPopover(NewTag("触发")).WithBody(NewTag("富内容")))
	e.HandleEvent(mouse(event.TypeMouseDown))
	if len(activeOverlays()) != 1 {
		t.Fatalf("带 Body 的 popover click 应显示, got %d", len(activeOverlays()))
	}
}
