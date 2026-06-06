package widget

import (
	"fmt"

	"github.com/user/goui/internal/canvas"
	"github.com/user/goui/internal/event"
	"github.com/user/goui/internal/layout"
	"github.com/user/goui/internal/paint"
	"github.com/user/goui/internal/types"
)

// ─── TimePicker 时间选择（el-time-picker spinner）──────────────
// 触发器(框+时间+时钟图标) + 点击弹出时/分两列滚动浮层；点时/分回填 "HH:MM"。

type TimePicker struct {
	StatelessWidget
	pseudoColors // CSS 伪类：交互态边框色覆盖（:hover/:focus）
	Value       string
	Placeholder string
	OnChange    func(string)
	Width       float64
}

func NewTimePicker() *TimePicker { return &TimePicker{Width: 160} }
func (t *TimePicker) WithValue(v string) *TimePicker          { t.Value = v; return t }
func (t *TimePicker) WithOnChange(fn func(string)) *TimePicker { t.OnChange = fn; return t }

func (t *TimePicker) CreateElement() Element {
	return &TimePickerElement{BaseElement: BaseElement{widget: t}, tp: t}
}

type TimePickerElement struct {
	BaseElement
	tp        *TimePicker
	overlayID int
	open      bool
	hovered   bool
}

func (e *TimePickerElement) Build() []Element { return nil }

func (e *TimePickerElement) Layout(ctx *layout.LayoutContext) layout.LayoutResult {
	w := e.tp.Width
	if w <= 0 {
		w = 160
	}
	e.size = ctx.Constraints.Constrain(types.Size{Width: w, Height: 32})
	return layout.LayoutResult{Size: e.size}
}

func (e *TimePickerElement) Paint(cvs canvas.Canvas, offset types.Point) {
	pos := e.Offset()
	W, H := e.size.Width, e.size.Height
	border := e.tp.borderOr(elBorder())
	if e.hovered || e.open {
		if e.open {
			border = e.tp.focusBorderOr(elPrimary())
		} else {
			border = e.tp.hoverBorderOr(elPrimary())
		}
	}
	bg := paint.DefaultPaint()
	bg.Color = elSurface()
	cvs.DrawRoundedRect(pos.X, pos.Y, W, H, 4, bg)
	bp := paint.DefaultStrokePaint()
	bp.Color = border
	bp.StrokeWidth = 1
	cvs.DrawRoundedRect(pos.X+0.5, pos.Y+0.5, W-1, H-1, 4, bp)

	// 时钟图标
	ic := paint.DefaultStrokePaint()
	ic.Color = elPlaceholder()
	ic.StrokeWidth = 1.2
	icx, icy := pos.X+16, pos.Y+H/2
	cvs.DrawCircle(icx, icy, 6, ic)
	cvs.DrawLine(icx, icy, icx, icy-3, ic)
	cvs.DrawLine(icx, icy, icx+3, icy, ic)

	label := e.tp.Value
	tc := elTextRegular()
	if label == "" {
		label, tc = orI18n(e.tp.Placeholder, "el.datepicker.selectTime"), elPlaceholder()
	}
	font := canvas.DefaultFont()
	font.Size = 14
	canvas.DrawTextAligned(cvs, label, types.Rect{X: pos.X + 30, Y: pos.Y, Width: W - 42, Height: H},
		font, tc, canvas.HAlignLeft, canvas.VAlignMiddle)
}

func (e *TimePickerElement) HandleEvent(ev event.Event) bool {
	switch ev.Type() {
	case event.TypeMouseEnter:
		e.hovered = true
		e.MarkNeedsPaint()
		return true
	case event.TypeMouseLeave:
		e.hovered = false
		e.MarkNeedsPaint()
		return true
	case event.TypeMouseDown:
		if e.open {
			e.hide()
		} else {
			e.show()
		}
		e.MarkNeedsPaint()
		return true
	}
	return false
}

func (e *TimePickerElement) show() {
	if e.open {
		return
	}
	off := e.Offset()
	panel := &timePanel{value: e.tp.Value, onSelect: func(v string) {
		e.tp.Value = v
		if e.tp.OnChange != nil {
			e.tp.OnChange(v)
		}
		e.MarkNeedsPaint()
	}}
	e.overlayID = ShowOverlay(&OverlayEntry{
		AnchorRect:     types.Rect{X: off.X, Y: off.Y, Width: e.size.Width, Height: e.size.Height},
		Content:        panel,
		Placement:      PlacementBottomStart,
		Gap:            4,
		Transition:     "fade",
		OnClickOutside: func() { e.hide(); e.MarkNeedsPaint() },
	})
	e.open = true
}

func (e *TimePickerElement) hide() {
	if !e.open {
		return
	}
	HideOverlay(e.overlayID)
	e.open = false
}

func (e *TimePickerElement) Update(newWidget Widget) {
	if nt, ok := newWidget.(*TimePicker); ok {
		e.tp = nt
		e.BaseElement.widget = newWidget
		e.dirty = true
	}
}

// ─── timePanel 时/分两列滚动浮层 ────────────────────────────────

type timePanel struct {
	StatelessWidget
	value    string
	onSelect func(string)
}

func (p *timePanel) CreateElement() Element {
	return &timePanelElement{BaseElement: BaseElement{widget: p}, panel: p, selH: -1, selM: -1}
}

const (
	tpColW   = 72.0
	tpItemH  = 30.0
	tpPanelH = 200.0
)

type timePanelElement struct {
	BaseElement
	panel            *timePanel
	selH, selM       int
	hScroll, mScroll float64
	inited           bool
}

func (e *timePanelElement) Build() []Element { return nil }

func (e *timePanelElement) ensureInit() {
	if e.inited {
		return
	}
	e.inited = true
	var h, m int
	if n, _ := fmt.Sscanf(e.panel.value, "%d:%d", &h, &m); n == 2 {
		e.selH, e.selM = h, m
		e.hScroll = clamp(float64(h)*tpItemH-tpPanelH/2, 0, 24*tpItemH-tpPanelH)
		e.mScroll = clamp(float64(m)*tpItemH-tpPanelH/2, 0, 60*tpItemH-tpPanelH)
	}
}

func (e *timePanelElement) Layout(ctx *layout.LayoutContext) layout.LayoutResult {
	e.ensureInit()
	e.size = ctx.Constraints.Constrain(types.Size{Width: tpColW * 2, Height: tpPanelH})
	return layout.LayoutResult{Size: e.size}
}

func (e *timePanelElement) Paint(cvs canvas.Canvas, offset types.Point) {
	e.ensureInit()
	pos := e.Offset()
	W, H := e.size.Width, e.size.Height

	shadow := paint.DefaultPaint()
	shadow.Color = types.ColorFromRGBA(0, 0, 0, 22)
	cvs.DrawRoundedRect(pos.X, pos.Y+2, W, H, 4, shadow)
	bg := paint.DefaultPaint()
	bg.Color = elSurface()
	cvs.DrawRoundedRect(pos.X, pos.Y, W, H, 4, bg)
	bp := paint.DefaultStrokePaint()
	bp.Color = elBorderLight()
	bp.StrokeWidth = 1
	cvs.DrawRoundedRect(pos.X+0.5, pos.Y+0.5, W-1, H-1, 4, bp)

	e.paintCol(cvs, pos.X, pos.Y, 24, e.selH, e.hScroll)
	e.paintCol(cvs, pos.X+tpColW, pos.Y, 60, e.selM, e.mScroll)
	sep := paint.DefaultStrokePaint()
	sep.Color = elBorderLighter()
	sep.StrokeWidth = 1
	cvs.DrawLine(pos.X+tpColW, pos.Y+4, pos.X+tpColW, pos.Y+H-4, sep)
}

func (e *timePanelElement) paintCol(cvs canvas.Canvas, x, y float64, count, sel int, scroll float64) {
	cvs.Save()
	cvs.ClipRect(x, y, tpColW, tpPanelH)
	f := canvas.DefaultFont()
	f.Size = 14
	for i := 0; i < count; i++ {
		iy := y + float64(i)*tpItemH - scroll
		if iy+tpItemH < y || iy > y+tpPanelH {
			continue
		}
		if i == sel {
			hp := paint.DefaultPaint()
			hp.Color = types.ColorFromRGB(236, 245, 255)
			cvs.DrawRect(x+2, iy, tpColW-4, tpItemH, hp)
		}
		col := elTextRegular()
		if i == sel {
			col = elPrimary()
		}
		canvas.DrawTextAligned(cvs, fmt.Sprintf("%02d", i), types.Rect{X: x, Y: iy, Width: tpColW, Height: tpItemH},
			f, col, canvas.HAlignCenter, canvas.VAlignMiddle)
	}
	cvs.Restore()
}

func (e *timePanelElement) HandleEvent(ev event.Event) bool {
	me, ok := ev.(*event.MouseEvent)
	if !ok {
		return false
	}
	pos := e.Offset()
	lx, ly := me.X-pos.X, me.Y-pos.Y
	inH := lx < tpColW
	switch ev.Type() {
	case event.TypeMouseWheel:
		if inH {
			e.hScroll = clamp(e.hScroll-me.DeltaY*scrollWheelStep, 0, 24*tpItemH-tpPanelH)
		} else {
			e.mScroll = clamp(e.mScroll-me.DeltaY*scrollWheelStep, 0, 60*tpItemH-tpPanelH)
		}
		ev.StopPropagation()
		e.MarkNeedsPaint()
		return true
	case event.TypeMouseDown:
		if inH {
			e.selH = int((ly + e.hScroll) / tpItemH)
		} else {
			e.selM = int((ly + e.mScroll) / tpItemH)
		}
		if e.selH >= 0 && e.selM >= 0 && e.panel.onSelect != nil {
			e.panel.onSelect(fmt.Sprintf("%02d:%02d", e.selH, e.selM))
		}
		e.MarkNeedsPaint()
		return true
	}
	return false
}

func (e *timePanelElement) Update(newWidget Widget) {
	if np, ok := newWidget.(*timePanel); ok {
		e.panel = np
		e.BaseElement.widget = newWidget
		e.dirty = true
	}
}
