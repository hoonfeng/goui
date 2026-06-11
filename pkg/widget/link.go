package widget

import (
	"github.com/hoonfeng/goui/pkg/canvas"
	"github.com/hoonfeng/goui/pkg/event"
	"github.com/hoonfeng/goui/internal/layout"
	"github.com/hoonfeng/goui/pkg/paint"
	"github.com/hoonfeng/goui/pkg/types"
)

// ─── Link 链接 ─────────────────────────────────────────────────
// 1:1 复刻 el-link：type×6 配色、hover 下划线、disabled、click 事件。

type LinkType int

const (
	LinkDefault LinkType = iota // default（文字色）
	LinkPrimary
	LinkSuccess
	LinkWarning
	LinkDanger
	LinkInfo
)

// Link 文字链接。
type Link struct {
	StatelessWidget
	Text      string
	Type      LinkType
	Underline bool // hover 时是否显示下划线(默认 true)
	Disabled  bool
	OnClick   func()
}

func NewLink(text string, onClick func()) *Link {
	return &Link{Text: text, Type: LinkDefault, Underline: true, OnClick: onClick}
}
func (l *Link) WithType(t LinkType) *Link  { l.Type = t; return l }
func (l *Link) WithUnderline(b bool) *Link { l.Underline = b; return l }
func (l *Link) WithDisabled(b bool) *Link  { l.Disabled = b; return l }

func linkColor(t LinkType) types.Color {
	th := CurrentTheme()
	switch t {
	case LinkPrimary:
		return th.Tag.PrimaryColor
	case LinkSuccess:
		return th.Tag.SuccessColor
	case LinkWarning:
		return th.Tag.WarningColor
	case LinkDanger:
		return th.Tag.DangerColor
	case LinkInfo:
		return th.Tag.InfoColor
	default:
		return elTextRegular() // default 文字色
	}
}

func (l *Link) CreateElement() Element {
	return &LinkElement{BaseElement: BaseElement{widget: l}, link: l}
}

type LinkElement struct {
	BaseElement
	link    *Link
	hovered bool
}

func (e *LinkElement) Build() []Element { return nil }

func (e *LinkElement) Layout(ctx *layout.LayoutContext) layout.LayoutResult {
	font := canvas.DefaultFont()
	font.Size = 14
	m := canvas.MeasureTextGlobal(e.link.Text, font)
	e.size = ctx.Constraints.Constrain(types.Size{Width: m.Width, Height: canvas.GetFaceLineHeight(14)})
	return layout.LayoutResult{Size: e.size}
}

func (e *LinkElement) Paint(cvs canvas.Canvas, offset types.Point) {
	pos := e.Offset()
	l := e.link
	color := linkColor(l.Type)
	if l.Disabled {
		color = types.Color{R: color.R, G: color.G, B: color.B, A: 130}
	} else if e.hovered {
		color = tagMix(color, 0.7) // hover 变浅
	}
	font := canvas.DefaultFont()
	font.Size = 14
	ascent := canvas.GetFaceAscent(14)
	lineH := canvas.GetFaceLineHeight(14)
	tp := paint.DefaultPaint()
	tp.Color = color
	ty := pos.Y + e.size.Height/2 + ascent - lineH/2
	cvs.DrawText(l.Text, pos.X, ty, font, tp)

	if l.Underline && e.hovered && !l.Disabled {
		lp := paint.DefaultStrokePaint()
		lp.Color = color
		lp.StrokeWidth = 1
		uy := ty + 2
		cvs.DrawLine(pos.X, uy, pos.X+e.size.Width, uy, lp)
	}
}

func (e *LinkElement) HandleEvent(ev event.Event) bool {
	if e.link.Disabled {
		return false
	}
	switch ev.Type() {
	case event.TypeMouseEnter:
		e.hovered = true
		tagRepaint()
		return true
	case event.TypeMouseLeave:
		e.hovered = false
		tagRepaint()
		return true
	case event.TypeMouseUp:
		if e.link.OnClick != nil {
			e.link.OnClick()
		}
		return true
	}
	return false
}

func (e *LinkElement) Update(newWidget Widget) {
	if nl, ok := newWidget.(*Link); ok {
		e.link = nl
		e.BaseElement.widget = newWidget
		e.dirty = true
	}
}
