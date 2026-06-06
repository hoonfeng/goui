package widget

import (
	"github.com/user/goui/internal/canvas"
	"github.com/user/goui/internal/event"
	"github.com/user/goui/internal/layout"
	"github.com/user/goui/internal/paint"
	"github.com/user/goui/internal/types"
)

// ─── Alert 警告 ────────────────────────────────────────────────
// 1:1 复刻 el-alert：type×4、effect(light/dark)、title/description、
// closable、show-icon、close 事件。

type AlertType int

const (
	AlertSuccess AlertType = iota
	AlertInfo
	AlertWarning
	AlertError
)

// Alert 警告提示。
type Alert struct {
	StatelessWidget
	Title       string
	Description string
	Type        AlertType
	Dark        bool // effect=dark（实色底白字）
	Closable    bool
	Center      bool
	ShowIcon    bool
	OnClose     func()
	closed      bool // 内部关闭状态(存于 Widget，使 relayout 重建 Element 后仍保留)
}

func NewAlert(title string, t AlertType) *Alert {
	return &Alert{Title: title, Type: t, ShowIcon: true}
}
func (a *Alert) WithDescription(s string) *Alert { a.Description = s; return a }
func (a *Alert) WithDark(b bool) *Alert          { a.Dark = b; return a }
func (a *Alert) WithClosable(b bool) *Alert      { a.Closable = b; return a }
func (a *Alert) WithShowIcon(b bool) *Alert      { a.ShowIcon = b; return a }

func alertBase(t AlertType) types.Color {
	tg := CurrentTheme().Tag
	switch t {
	case AlertSuccess:
		return tg.SuccessColor
	case AlertWarning:
		return tg.WarningColor
	case AlertError:
		return tg.DangerColor
	default:
		return tg.InfoColor
	}
}

func alertIcon(t AlertType) string {
	switch t {
	case AlertSuccess:
		return "✔"
	case AlertWarning:
		return "⚠"
	case AlertError:
		return "✖"
	default:
		return "ℹ"
	}
}

func (a *Alert) CreateElement() Element {
	return &AlertElement{BaseElement: BaseElement{widget: a}, alert: a}
}

type AlertElement struct {
	BaseElement
	alert *Alert
}

func (e *AlertElement) Build() []Element { return nil }

func (e *AlertElement) Layout(ctx *layout.LayoutContext) layout.LayoutResult {
	if e.alert.closed {
		e.size = types.Size{}
		return layout.LayoutResult{Size: e.size}
	}
	w := ctx.Constraints.MaxWidth
	if w <= 0 || w > 1e6 {
		w = 400
	}
	padV := 10.0
	titleH := canvas.GetFaceLineHeight(14)
	descH := 0.0
	if e.alert.Description != "" {
		descH = canvas.GetFaceLineHeight(13) + 3
	}
	h := padV*2 + titleH + descH
	e.size = ctx.Constraints.Constrain(types.Size{Width: w, Height: h})
	return layout.LayoutResult{Size: e.size}
}

func (e *AlertElement) Paint(cvs canvas.Canvas, offset types.Point) {
	if e.alert.closed {
		return
	}
	pos := e.Offset()
	a := e.alert
	base := alertBase(a.Type)
	var bg, titleColor, descColor types.Color
	if a.Dark {
		bg, titleColor, descColor = base, types.ColorWhite, types.ColorWhite
	} else {
		bg, titleColor, descColor = tagMix(base, 0.14), base, base
	}

	bgp := paint.DefaultPaint()
	bgp.Color = bg
	cvs.DrawRoundedRect(pos.X, pos.Y, e.size.Width, e.size.Height, 4, bgp)

	padH := 16.0
	centerY := pos.Y + e.size.Height/2

	// 内容块(title[+description])整体垂直居中
	titleLineH := canvas.GetFaceLineHeight(14)
	descLineH := canvas.GetFaceLineHeight(13)
	descGap := 3.0
	contentH := titleLineH
	if a.Description != "" {
		contentH += descGap + descLineH
	}
	topY := centerY - contentH/2
	titleBaseline := canvas.BaselineFor(topY, titleLineH, 14, canvas.VAlignMiddle)

	x := pos.X + padH
	if a.ShowIcon {
		ifont := canvas.DefaultFont()
		ifont.Size = 15
		ip := paint.DefaultPaint()
		ip.Color = titleColor
		cvs.DrawText(alertIcon(a.Type), x, titleBaseline, ifont, ip)
		x += 22
	}

	font := canvas.DefaultFont()
	font.Size = 14
	font.Weight = canvas.FontWeightBold
	tp := paint.DefaultPaint()
	tp.Color = titleColor
	cvs.DrawText(a.Title, x, titleBaseline, font, tp)

	if a.Description != "" {
		dfont := canvas.DefaultFont()
		dfont.Size = 13
		dp := paint.DefaultPaint()
		dp.Color = descColor
		descBaseline := canvas.BaselineFor(topY+titleLineH+descGap, descLineH, 13, canvas.VAlignMiddle)
		cvs.DrawText(a.Description, x, descBaseline, dfont, dp)
	}

	if a.Closable {
		xp := paint.DefaultStrokePaint()
		xp.Color = titleColor
		xp.StrokeWidth = 1.2
		cx := pos.X + e.size.Width - padH
		d := 4.0
		cvs.DrawLine(cx-d, centerY-d, cx+d, centerY+d, xp)
		cvs.DrawLine(cx-d, centerY+d, cx+d, centerY-d, xp)
	}
}

func (e *AlertElement) HandleEvent(ev event.Event) bool {
	if e.alert.closed || !e.alert.Closable {
		return false
	}
	if ev.Type() == event.TypeMouseUp {
		if me, ok := ev.(*event.MouseEvent); ok {
			pos := e.Offset()
			cx := pos.X + e.size.Width - 16
			cy := pos.Y + e.size.Height/2
			dx, dy := me.X-cx, me.Y-cy
			if dx*dx+dy*dy <= 120 {
				e.alert.closed = true
				if e.alert.OnClose != nil {
					e.alert.OnClose()
				}
				// closed 存于 Widget：relayout 重建 Element 后仍为 true → Layout 返回 0 尺寸，
				// 真正移除并让下方上移，且不会被其他组件触发的 relayout 复活。
				if OnNeedsLayout != nil {
					OnNeedsLayout()
				}
				return true
			}
		}
	}
	return false
}

func (e *AlertElement) Update(newWidget Widget) {
	if na, ok := newWidget.(*Alert); ok {
		e.alert = na
		e.BaseElement.widget = newWidget
		e.dirty = true
	}
}
