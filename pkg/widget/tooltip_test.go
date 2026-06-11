package widget

import (
	"testing"

	"github.com/hoonfeng/goui/pkg/event"
	"github.com/hoonfeng/goui/internal/layout"
)

func newTooltipElement(t *Tooltip) *TooltipElement {
	e := t.CreateElement().(*TooltipElement)
	e.Build()
	e.Layout(&layout.LayoutContext{Constraints: layout.BoxConstraints{MaxWidth: 500, MaxHeight: 500}})
	return e
}

func mouse(ty event.Type) *event.MouseEvent {
	return event.NewMouseEvent(ty, 0, 0, event.ButtonLeft, event.ModNone)
}

func TestTooltipHoverShowHide(t *testing.T) {
	overlayEntries = nil
	e := newTooltipElement(NewTooltip("提示文字", NewTag("锚点")))

	e.HandleEvent(mouse(event.TypeMouseEnter))
	if len(activeOverlays()) != 1 || !e.shown {
		t.Fatalf("hover enter 应显示浮层：count=%d shown=%v", len(activeOverlays()), e.shown)
	}
	e.HandleEvent(mouse(event.TypeMouseLeave))
	if len(activeOverlays()) != 0 || e.shown {
		t.Fatalf("hover leave 应隐藏浮层：count=%d shown=%v", len(activeOverlays()), e.shown)
	}
}

func TestTooltipClickTrigger(t *testing.T) {
	overlayEntries = nil
	e := newTooltipElement(NewTooltip("点击提示", NewTag("锚点")).WithTrigger(TooltipClick))

	e.HandleEvent(mouse(event.TypeMouseDown))
	if len(activeOverlays()) != 1 {
		t.Fatalf("click 应显示浮层, got %d", len(activeOverlays()))
	}
	// click 触发下，hover leave 不应隐藏
	e.HandleEvent(mouse(event.TypeMouseLeave))
	if len(activeOverlays()) != 1 {
		t.Error("click 触发下 hover leave 不应隐藏")
	}
	// 再次 click 关闭
	e.HandleEvent(mouse(event.TypeMouseDown))
	if len(activeOverlays()) != 0 {
		t.Fatalf("再次 click 应隐藏, got %d", len(activeOverlays()))
	}
}

func TestTooltipDisabled(t *testing.T) {
	overlayEntries = nil
	e := newTooltipElement(NewTooltip("提示", NewTag("锚点")).WithDisabled(true))
	e.HandleEvent(mouse(event.TypeMouseEnter))
	if len(activeOverlays()) != 0 {
		t.Error("disabled tooltip 不应显示浮层")
	}
}

func TestTooltipAnchorRect(t *testing.T) {
	overlayEntries = nil
	e := newTooltipElement(NewTooltip("提示", NewTag("锚点")))
	e.HandleEvent(mouse(event.TypeMouseEnter))
	ovs := activeOverlays()
	if len(ovs) != 1 {
		t.Fatalf("应有 1 个浮层, got %d", len(ovs))
	}
	// 锚点矩形应与子元素尺寸一致(宽高 > 0)
	if ovs[0].AnchorRect.Width <= 0 || ovs[0].AnchorRect.Height <= 0 {
		t.Errorf("锚点矩形应有正尺寸, got %+v", ovs[0].AnchorRect)
	}
}
