package widget

import (
	"time"

	"github.com/user/goui/internal/animation"
	"github.com/user/goui/internal/canvas"
	"github.com/user/goui/internal/event"
	"github.com/user/goui/internal/layout"
	"github.com/user/goui/internal/paint"
	"github.com/user/goui/internal/types"
)

// DragRow 可拖拽重排的列表行（自绘叶子，整行可拖）：lead 图标 + 主图标 + 文本 + 可选尾图标。
// 区分点击与拖拽：快速点击=OnTap；长按(默认 350ms)或明显移动(>8px)→进入拖拽 OnDragStart/Move/End。
// 右键→OnContext。整行是命中叶子（无子元素）故可被捕获、支持整行任意位置拖（同 DragGrip 捕获机制）。
type DragRow struct {
	StatelessWidget
	LeadIcon    string // 左侧图标（如 chevron），空=无
	LeadColor   types.Color
	Icon        string // 主图标（如 folder），空=无
	Text        string
	TextColor   types.Color
	TextSize    float64
	TrailIcon   string // 右侧图标（如 star），空=无
	TrailColor  types.Color
	Height      float64     // 行高（默认 26）
	Indent      float64     // 左右内边距（默认 6）
	Bg          types.Color // 常态底
	HoverBg     types.Color // 悬停/拖拽底
	Active      bool        // true → 用 HoverBg（如正被拖动）
	LongPressMs int         // 长按进入拖拽毫秒（默认 350）
	OnTap       func()
	OnContext   func(x, y float64)
	OnDragStart func(y float64)
	OnDragMove  func(y float64)
	OnDragEnd   func()
}

func (w *DragRow) CreateElement() Element {
	return &dragRowElement{BaseElement: BaseElement{widget: w}, w: w}
}

type dragRowElement struct {
	BaseElement
	w       *DragRow
	hovered bool
	pressed bool
	armed   bool // 已进入拖拽
	startY  float64
	timer   *animation.Controller
}

func (e *dragRowElement) Build() []Element { return nil }

func (e *dragRowElement) Layout(ctx *layout.LayoutContext) layout.LayoutResult {
	h := e.w.Height
	if h <= 0 {
		h = 26
	}
	e.size = ctx.Constraints.Constrain(types.Size{Width: ctx.Constraints.MaxWidth, Height: h})
	return layout.LayoutResult{Size: e.size}
}

func (e *dragRowElement) Paint(cvs canvas.Canvas, offset types.Point) {
	pos := e.Offset()
	w := e.w
	pad := w.Indent
	if pad <= 0 {
		pad = 6
	}
	bg := w.Bg
	if w.Active || e.hovered {
		bg = w.HoverBg
	}
	if bg.A > 0 {
		p := paint.DefaultPaint()
		p.Color = bg
		cvs.DrawRect(pos.X, pos.Y, e.size.Width, e.size.Height, p)
	}
	if w.Active { // 拖拽中：内容半透明（"拎起来"的影子感）
		cvs.SaveLayerAlpha(0.5)
		defer cvs.Restore()
	}
	cy := pos.Y + e.size.Height/2
	x := pos.X + pad
	if w.LeadIcon != "" {
		PaintLucide(cvs, w.LeadIcon, x, cy-6.5, 13, 2, w.LeadColor)
		x += 17
	}
	if w.Icon != "" {
		PaintLucide(cvs, w.Icon, x, cy-7, 14, 2, w.TextColor)
		x += 20
	}
	rightX := pos.X + e.size.Width - pad
	if w.TrailIcon != "" {
		PaintLucide(cvs, w.TrailIcon, rightX-12, cy-6, 12, 2, w.TrailColor)
		rightX -= 18
	}
	fs := w.TextSize
	if fs <= 0 {
		fs = 12
	}
	font := canvas.DefaultFont()
	font.Size = fs
	tr := types.Rect{X: x, Y: pos.Y, Width: rightX - x, Height: e.size.Height}
	canvas.DrawTextAligned(cvs, w.Text, tr, font, w.TextColor, canvas.HAlignLeft, canvas.VAlignMiddle)
}

func (e *dragRowElement) HandleEvent(ev event.Event) bool {
	me, _ := ev.(*event.MouseEvent)
	switch ev.Type() {
	case event.TypeMouseEnter:
		e.hovered = true
		e.MarkNeedsPaint()
		return true
	case event.TypeMouseLeave:
		e.hovered = false
		e.MarkNeedsPaint()
		return true
	case event.TypeContextMenu:
		if me != nil && e.w.OnContext != nil {
			e.w.OnContext(me.X, me.Y)
		}
		return true
	case event.TypeMouseDown:
		if me != nil && me.Button == event.ButtonRight {
			if e.w.OnContext != nil {
				e.w.OnContext(me.X, me.Y)
			}
			return true
		}
		e.pressed, e.armed = true, false
		if me != nil {
			e.startY = me.Y
		}
		if RequestPointerCapture != nil {
			RequestPointerCapture(e) // 抢捕获：拖到别处也持续收到 move/up
		}
		e.startTimer() // 长按计时
		return true
	case event.TypeMouseMove:
		if me == nil {
			return e.pressed
		}
		if e.armed {
			if e.w.OnDragMove != nil {
				e.w.OnDragMove(me.Y)
			}
			return true
		}
		if e.pressed { // 明显移动 → 立即进入拖拽（无需等长按）
			dy := me.Y - e.startY
			if dy < 0 {
				dy = -dy
			}
			if dy > 8 {
				e.arm(me.Y)
			}
			return true
		}
	case event.TypeMouseUp:
		e.stopTimer()
		if e.armed {
			e.armed, e.pressed = false, false
			if e.w.OnDragEnd != nil {
				e.w.OnDragEnd()
			}
			return true
		}
		if e.pressed {
			e.pressed = false
			if e.w.OnTap != nil {
				e.w.OnTap() // 快速点击（未拖）= 点击
			}
			return true
		}
	}
	return false
}

func (e *dragRowElement) arm(y float64) {
	if e.armed {
		return
	}
	e.armed = true
	e.stopTimer()
	if e.w.OnDragStart != nil {
		e.w.OnDragStart(y)
	}
}

func (e *dragRowElement) startTimer() {
	ms := e.w.LongPressMs
	if ms <= 0 {
		ms = 350
	}
	e.stopTimer()
	e.timer = animation.NewController(time.Duration(ms)*time.Millisecond, animation.Linear)
	e.timer.OnDone = func() { e.arm(e.startY) } // 长按到时（原地按住）→ 进入拖拽
	e.timer.Start()
}

func (e *dragRowElement) stopTimer() {
	if e.timer != nil {
		e.timer.Stop()
		e.timer = nil
	}
}

func (e *dragRowElement) Update(nw Widget) {
	if w, ok := nw.(*DragRow); ok {
		e.w = w // 同步新闭包；不重置 pressed/armed（拖动跨重建不丢）
	}
	e.widget = nw
	e.dirty = true
}

func (e *dragRowElement) Cursor() int { return CursorHand }
