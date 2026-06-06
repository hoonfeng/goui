package widget

import (
	"reflect"

	"github.com/user/goui/internal/canvas"
	"github.com/user/goui/internal/event"
	"github.com/user/goui/internal/layout"
	"github.com/user/goui/internal/types"
)

// OnWindowDrag 由 app 注入：在自绘标题栏拖动区按下鼠标时发起原生窗口拖动。
var OnWindowDrag func()

// WindowDragHandle 把子组件变成「窗口拖动区」：在其区域内按下鼠标即可拖动整个窗口
//（用于 Config.Borderless 无边框窗口的自绘标题栏）。
//
// 按钮等可点击控件请放在拖动区**之外**并列摆放（拖动区只包图标/标题/空白）——
// 否则按下按钮会被当成拖动，因为按下即进入系统窗口移动循环、吞掉后续 MouseUp。
func WindowDragHandle(child Widget) Widget {
	return &windowDragWidget{child: child}
}

type windowDragWidget struct {
	SingleChildWidget
	child Widget
}

func (w *windowDragWidget) CreateElement() Element {
	return &windowDragElement{BaseElement: BaseElement{widget: w}, w: w}
}

type windowDragElement struct {
	BaseElement
	w     *windowDragWidget
	child Element
}

func (e *windowDragElement) Update(nw Widget) {
	e.widget = nw
	e.dirty = true
	if w, ok := nw.(*windowDragWidget); ok {
		e.w = w
	}
}

func (e *windowDragElement) Build() []Element {
	if e.w.child == nil {
		if e.child != nil {
			e.child.Unmount()
			e.child = nil
		}
		e.children = nil
		return nil
	}
	if e.child != nil && reflect.TypeOf(e.child.Widget()) == reflect.TypeOf(e.w.child) {
		e.child.Update(e.w.child)
	} else {
		if e.child != nil {
			e.child.Unmount()
		}
		e.child = CreateElementFor(e.w.child)
		e.child.Mount(e, 0)
	}
	e.children = []Element{e.child}
	return e.children
}

func (e *windowDragElement) Layout(ctx *layout.LayoutContext) layout.LayoutResult {
	if e.child != nil {
		r := e.child.Layout(ctx)
		e.child.SetPosition(types.Point{})
		e.size = r.Size
	} else {
		e.size = types.Size{Width: ctx.Constraints.MinWidth, Height: ctx.Constraints.MinHeight}
	}
	return layout.LayoutResult{Size: e.size}
}

func (e *windowDragElement) Paint(cvs canvas.Canvas, offset types.Point) {
	if e.child != nil {
		e.child.Paint(cvs, e.Offset())
	}
}

// HandleEvent 拖动区按下鼠标 → 发起原生窗口拖动。
func (e *windowDragElement) HandleEvent(ev event.Event) bool {
	if ev.Type() == event.TypeMouseDown && OnWindowDrag != nil {
		OnWindowDrag()
		return true
	}
	return false
}
