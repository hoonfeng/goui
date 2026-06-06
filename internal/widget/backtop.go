package widget

import (
	"reflect"

	"github.com/user/goui/internal/canvas"
	"github.com/user/goui/internal/event"
	"github.com/user/goui/internal/layout"
	"github.com/user/goui/internal/paint"
	"github.com/user/goui/internal/types"
)

// ─── Backtop 回到顶部（滚动地基应用）─────────────────────────────
// 1:1 复刻 el-backtop：监听 ScrollView 滚动，滚动超过 VisibilityHeight 时
// 在右下角浮现圆形 ↑ 按钮，点击回到顶部。
// 设计：包裹 ScrollView —— 借事件冒泡接收点击(ScrollView 不消费 MouseDown→冒泡到这)、
// 借每帧重绘读取 scrollEl.ScrollOffset() 决定按钮显隐。

type Backtop struct {
	StatelessWidget
	scroll           *ScrollView
	VisibilityHeight float64 // 滚动超过此值(像素)才显示(默认 200)
	Right            float64 // 距右边距(默认 40)
	Bottom           float64 // 距底边距(默认 40)
}

func NewBacktop(scroll *ScrollView) *Backtop {
	return &Backtop{scroll: scroll, VisibilityHeight: 200, Right: 40, Bottom: 40}
}

func (b *Backtop) WithVisibilityHeight(h float64) *Backtop { b.VisibilityHeight = h; return b }
func (b *Backtop) WithOffset(right, bottom float64) *Backtop {
	b.Right, b.Bottom = right, bottom
	return b
}

func (b *Backtop) CreateElement() Element {
	return &BacktopElement{BaseElement: BaseElement{widget: b}, backtop: b}
}

type BacktopElement struct {
	BaseElement
	backtop  *Backtop
	scrollEl *ScrollViewElement
}

const backtopBtnSize = 40.0

func (e *BacktopElement) Build() []Element {
	c := e.backtop.scroll
	// reconcile：复用 ScrollViewElement，保留 scrollOffset（relayout 时页面不回顶）
	if e.scrollEl != nil && reflect.TypeOf(e.scrollEl.Widget()) == reflect.TypeOf(Widget(c)) {
		e.scrollEl.Update(c)
	} else {
		if e.scrollEl != nil {
			e.scrollEl.Unmount()
		}
		if sv, ok := CreateElementFor(c).(*ScrollViewElement); ok {
			e.scrollEl = sv
			e.scrollEl.Mount(e, 0)
		}
	}
	if e.scrollEl != nil {
		e.children = []Element{e.scrollEl}
	}
	return e.children
}

func (e *BacktopElement) Layout(ctx *layout.LayoutContext) layout.LayoutResult {
	if e.scrollEl != nil {
		res := e.scrollEl.Layout(ctx)
		e.scrollEl.SetPosition(types.Point{})
		e.size = res.Size
	} else {
		e.size = ctx.Constraints.Constrain(types.Size{Width: ctx.Constraints.MaxWidth, Height: ctx.Constraints.MaxHeight})
	}
	return layout.LayoutResult{Size: e.size}
}

// visible 当前是否应显示回顶按钮。
func (e *BacktopElement) visible() bool {
	return e.scrollEl != nil && e.scrollEl.ScrollOffset().Y > e.backtop.VisibilityHeight
}

// btnRect 回顶按钮的全局矩形。
func (e *BacktopElement) btnRect() types.Rect {
	pos := e.Offset()
	return types.Rect{
		X:      pos.X + e.size.Width - e.backtop.Right - backtopBtnSize,
		Y:      pos.Y + e.size.Height - e.backtop.Bottom - backtopBtnSize,
		Width:  backtopBtnSize,
		Height: backtopBtnSize,
	}
}

func (e *BacktopElement) Paint(cvs canvas.Canvas, offset types.Point) {
	if e.scrollEl != nil {
		e.scrollEl.Paint(cvs, offset)
	}
	if !e.visible() {
		return
	}
	r := e.btnRect()
	cx := r.X + r.Width/2
	cy := r.Y + r.Height/2

	// 圆形按钮底(浅阴影 + 白底)
	shadow := paint.DefaultPaint()
	shadow.Color = types.ColorFromRGBA(0, 0, 0, 30)
	cvs.DrawCircle(cx, cy+2, backtopBtnSize/2, shadow)
	bg := paint.DefaultPaint()
	bg.Color = elSurface()
	cvs.DrawCircle(cx, cy, backtopBtnSize/2, bg)

	// ↑ 箭头(蓝，3 条线段)
	ap := paint.DefaultStrokePaint()
	ap.Color = elPrimary()
	ap.StrokeWidth = 2
	cvs.DrawLine(cx, cy-7, cx, cy+7, ap)   // 竖干
	cvs.DrawLine(cx, cy-7, cx-6, cy-1, ap) // 左翼
	cvs.DrawLine(cx, cy-7, cx+6, cy-1, ap) // 右翼
}

// HandleEvent 点击按钮区(滚动已超阈值时)回到顶部。借冒泡链接收 MouseDown。
func (e *BacktopElement) HandleEvent(ev event.Event) bool {
	if ev.Type() == event.TypeMouseDown && e.visible() {
		if me, ok := ev.(*event.MouseEvent); ok {
			r := e.btnRect()
			if me.X >= r.X && me.X <= r.X+r.Width && me.Y >= r.Y && me.Y <= r.Y+r.Height {
				e.scrollEl.ScrollTo(types.Point{}) // 回到顶部
				ev.StopPropagation()
				return true
			}
		}
	}
	return false
}

func (e *BacktopElement) Update(newWidget Widget) {
	if nb, ok := newWidget.(*Backtop); ok {
		e.backtop = nb
		e.BaseElement.widget = newWidget
		e.dirty = true
	}
}
