package widget

import (
	"github.com/user/goui/internal/canvas"
	"github.com/user/goui/internal/event"
	"github.com/user/goui/internal/layout"
	"github.com/user/goui/internal/types"
)

// ContextArea 透明包裹：捕获右键（TypeContextMenu）触发自定义菜单，布局**完全透传**给子
// （不改约束、不加 padding、尺寸=子尺寸）。因此可给列表行/面板/滚动区等加右键菜单而不影响其布局
// （区别于 Clickable——后者会给子无界高约束、不适合包 ScrollView）。
// 右键事件经父链冒泡上来（见 app.dispatch/routeMouseEvent），命中即调 OnContextMenu(屏幕坐标)
// 并 StopPropagation（避免外层 ContextArea 也触发）。回调内通常调 widget.ShowContextMenu(x,y,items)。
type ContextArea struct {
	SingleChildWidget
	OnContextMenu func(x, y float64)
}

// CreateElement 创建 ContextAreaElement。
func (c *ContextArea) CreateElement() Element {
	return &ContextAreaElement{BaseElement: BaseElement{widget: c}, ca: c}
}

// ContextAreaElement ContextArea 的运行时 Element。
type ContextAreaElement struct {
	BaseElement
	ca    *ContextArea
	child Element
}

// Update 同步配置。
func (e *ContextAreaElement) Update(nw Widget) {
	e.widget = nw
	e.dirty = true
	if c, ok := nw.(*ContextArea); ok {
		e.ca = c
	}
}

// Build 构建子 Element。
func (e *ContextAreaElement) Build() []Element {
	if e.ca.Child != nil {
		e.child = CreateElementFor(e.ca.Child)
		e.child.Mount(e, 0)
		e.children = []Element{e.child}
		return e.children
	}
	e.child = nil
	e.children = nil
	return nil
}

// Layout 透传约束给子，自身尺寸=子尺寸（不改动布局）。
func (e *ContextAreaElement) Layout(ctx *layout.LayoutContext) layout.LayoutResult {
	if e.child != nil {
		res := e.child.Layout(ctx)
		e.child.SetPosition(types.Point{})
		e.size = res.Size
		return layout.LayoutResult{Size: e.size}
	}
	e.size = ctx.Constraints.Constrain(types.Size{})
	return layout.LayoutResult{Size: e.size}
}

// Paint 画子。
func (e *ContextAreaElement) Paint(cvs canvas.Canvas, offset types.Point) {
	if e.child != nil {
		e.child.Paint(cvs, e.Offset())
	}
}

// HandleEvent 捕获右键 → 回调 + 阻断冒泡（不影响左键/滚轮，交由子处理）。
func (e *ContextAreaElement) HandleEvent(ev event.Event) bool {
	if ev.Type() == event.TypeContextMenu {
		if me, ok := ev.(*event.MouseEvent); ok && e.ca.OnContextMenu != nil {
			e.ca.OnContextMenu(me.X, me.Y)
			ev.StopPropagation()
			return true
		}
	}
	return false
}
