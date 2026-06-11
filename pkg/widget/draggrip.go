package widget

import (
	"github.com/hoonfeng/goui/pkg/canvas"
	"github.com/hoonfeng/goui/pkg/event"
	"github.com/hoonfeng/goui/internal/layout"
	"github.com/hoonfeng/goui/pkg/paint"
	"github.com/hoonfeng/goui/pkg/types"
)

// DragGrip 自绘拖拽手柄：画一个 lucide 图标，按下并拖动时回调起/中/止的鼠标坐标（窗口坐标）。
// 用 RequestPointerCapture 保证拖到别处也持续收到 move/up（同 ResizeHandle 模式）。
// 用于面板拖拽停靠：上层据拖动方向/落点决定目标区。微小移动可由上层当作「点击」处理。
type DragGrip struct {
	StatelessWidget
	Icon    string
	Box     float64     // 命中+绘制方框边长（默认 18）
	IconSz  float64     // 图标尺寸（默认 13）
	Color   types.Color
	Bg      types.Color // 背景（A>0 时画圆角底，使手柄更像可拖拽按钮、更醒目）
	OnStart func(x, y float64)
	OnMove  func(x, y float64)
	OnEnd   func(x, y float64)
}

func (g *DragGrip) CreateElement() Element {
	return &dragGripElement{BaseElement: BaseElement{widget: g}, g: g}
}

type dragGripElement struct {
	BaseElement
	g        *DragGrip
	dragging bool
}

func (e *dragGripElement) Build() []Element { return nil }

func (e *dragGripElement) Layout(ctx *layout.LayoutContext) layout.LayoutResult {
	b := e.g.Box
	if b <= 0 {
		b = 18
	}
	e.size = types.Size{Width: b, Height: b}
	return layout.LayoutResult{Size: e.size}
}

func (e *dragGripElement) Paint(cvs canvas.Canvas, offset types.Point) {
	pos := e.Offset()
	if e.g.Bg.A > 0 { // 圆角底：让手柄更像可拖拽按钮、更醒目
		p := paint.DefaultPaint()
		p.Color = e.g.Bg
		cvs.DrawRoundedRect(pos.X, pos.Y, e.size.Width, e.size.Height, 4, p)
	}
	sz := e.g.IconSz
	if sz <= 0 {
		sz = 13
	}
	PaintLucide(cvs, e.g.Icon, pos.X+(e.size.Width-sz)/2, pos.Y+(e.size.Height-sz)/2, sz, 2, e.g.Color)
}

func (e *dragGripElement) HandleEvent(ev event.Event) bool {
	me, ok := ev.(*event.MouseEvent)
	if !ok {
		return false
	}
	switch ev.Type() {
	case event.TypeMouseDown:
		e.dragging = true
		if RequestPointerCapture != nil {
			RequestPointerCapture(e) // 抢捕获：拖到面板上方也持续收到 move/up
		}
		if e.g.OnStart != nil {
			e.g.OnStart(me.X, me.Y)
		}
		return true
	case event.TypeMouseMove:
		if e.dragging {
			if e.g.OnMove != nil {
				e.g.OnMove(me.X, me.Y)
			}
			return true
		}
	case event.TypeMouseUp:
		if e.dragging {
			e.dragging = false
			if e.g.OnEnd != nil {
				e.g.OnEnd(me.X, me.Y)
			}
			return true
		}
	}
	return false
}

func (e *dragGripElement) Update(nw Widget) {
	if g, ok := nw.(*DragGrip); ok {
		e.g = g // 同步新闭包（OnStart/Move/End 捕获最新状态）；不重置 dragging，拖动跨重建不丢
	}
	e.widget = nw
	e.dirty = true
}

func (e *dragGripElement) Cursor() int { return CursorHand }
