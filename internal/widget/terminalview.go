package widget

import (
	"github.com/user/goui/internal/canvas"
	"github.com/user/goui/internal/event"
	"github.com/user/goui/internal/layout"
	"github.com/user/goui/internal/types"
)

// TerminalView 可聚焦的终端视图：铺满父约束、OnPaint 自绘网格、点击聚焦、KeyDown/KeyChar 转 OnKey。
// 用于真终端（PTY+vterm）：上层在 OnPaint 里画 vterm 网格、在 OnKey 里把按键转 VT 写进 PTY。
// 事件：MouseDown 触发 app 的 focus-on-down 选中本元素；聚焦后键盘事件路由到这里。
type TerminalView struct {
	StatelessWidget
	OnPaint func(cvs canvas.Canvas, x, y, w, h float64)
	OnKey   func(ev *event.KeyEvent)  // KeyDown 与 KeyChar 都回调；上层据 ev.Type()/Char/Key 区分
	OnWheel func(deltaY float64)      // 滚轮：deltaY>0 上滚（看历史），<0 下滚
}

func (w *TerminalView) CreateElement() Element {
	return &terminalViewElement{BaseElement: BaseElement{widget: w}, w: w}
}

type terminalViewElement struct {
	BaseElement
	w       *TerminalView
	focused bool
}

func (e *terminalViewElement) Build() []Element { return nil }

func (e *terminalViewElement) Layout(ctx *layout.LayoutContext) layout.LayoutResult {
	e.size = ctx.Constraints.Constrain(types.Size{Width: ctx.Constraints.MaxWidth, Height: ctx.Constraints.MaxHeight})
	return layout.LayoutResult{Size: e.size}
}

func (e *terminalViewElement) Paint(cvs canvas.Canvas, offset types.Point) {
	if e.w.OnPaint != nil {
		pos := e.Offset()
		e.w.OnPaint(cvs, pos.X, pos.Y, e.size.Width, e.size.Height)
	}
}

func (e *terminalViewElement) Focus() {
	e.focused = true
	if OnNeedsRepaint != nil {
		OnNeedsRepaint()
	}
}

func (e *terminalViewElement) Blur() {
	e.focused = false
	if OnNeedsRepaint != nil {
		OnNeedsRepaint()
	}
}

func (e *terminalViewElement) IsFocused() bool { return e.focused }

func (e *terminalViewElement) HandleEvent(ev event.Event) bool {
	switch ev.Type() {
	case event.TypeMouseDown:
		return true // 让 app focus-on-down 把焦点设到本元素（键盘才会路由过来）
	case event.TypeKeyDown, event.TypeKeyChar:
		if ke, ok := ev.(*event.KeyEvent); ok && e.w.OnKey != nil {
			e.w.OnKey(ke)
			return true
		}
	case event.TypeMouseWheel:
		if me, ok := ev.(*event.MouseEvent); ok && e.w.OnWheel != nil {
			e.w.OnWheel(me.DeltaY)
			return true
		}
	}
	return false
}

func (e *terminalViewElement) Update(nw Widget) {
	if w, ok := nw.(*TerminalView); ok {
		e.w = w
	}
	e.widget = nw
	e.dirty = true
}
