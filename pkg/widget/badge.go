package widget

import (
	"strconv"

	"github.com/hoonfeng/goui/pkg/canvas"
	"github.com/hoonfeng/goui/internal/layout"
	"github.com/hoonfeng/goui/pkg/paint"
	"github.com/hoonfeng/goui/pkg/types"
)

// ─── Badge 徽章 ────────────────────────────────────────────────
// 1:1 复刻 el-badge：包裹子元素，在其右上角显示数字/文本角标或小红点。
//   props: value / max / is-dot / hidden / type / show-zero

type BadgeType int

const (
	BadgeDanger  BadgeType = iota // danger（默认，红）
	BadgePrimary                  // primary
	BadgeSuccess                  // success
	BadgeWarning                  // warning
	BadgeInfo                     // info
)

// Badge 徽章。
type Badge struct {
	StatelessWidget
	Child    Widget
	Value    string    // 角标内容(数字或文本)
	Max      int       // >0 且 Value 为数字超出时显示 "Max+"
	IsDot    bool      // 小红点模式
	Hidden   bool      // 隐藏角标
	Type     BadgeType // 配色
	ShowZero bool      // Value 为 "0" 时是否显示
}

func NewBadge(value string, child Widget) *Badge {
	return &Badge{Child: child, Value: value, Type: BadgeDanger}
}
func NewDotBadge(child Widget) *Badge {
	return &Badge{Child: child, IsDot: true, Type: BadgeDanger}
}
func (b *Badge) WithType(t BadgeType) *Badge { b.Type = t; return b }
func (b *Badge) WithMax(m int) *Badge        { b.Max = m; return b }
func (b *Badge) WithHidden(h bool) *Badge    { b.Hidden = h; return b }

func badgeColor(tp BadgeType) types.Color {
	tg := CurrentTheme().Tag
	switch tp {
	case BadgePrimary:
		return tg.PrimaryColor
	case BadgeSuccess:
		return tg.SuccessColor
	case BadgeWarning:
		return tg.WarningColor
	case BadgeInfo:
		return tg.InfoColor
	default:
		return tg.DangerColor
	}
}

func (b *Badge) CreateElement() Element {
	return &BadgeElement{BaseElement: BaseElement{widget: b}, badge: b}
}

type BadgeElement struct {
	BaseElement
	badge   *Badge
	childEl Element
}

func (e *BadgeElement) Build() []Element {
	if e.badge.Child != nil {
		e.childEl = CreateElementFor(e.badge.Child)
		e.childEl.Mount(e, 0)
		e.children = []Element{e.childEl}
		return e.children
	}
	e.children = nil
	return nil
}

func (e *BadgeElement) Layout(ctx *layout.LayoutContext) layout.LayoutResult {
	if e.childEl != nil {
		res := e.childEl.Layout(ctx)
		e.childEl.SetPosition(types.Point{})
		e.size = res.Size
	}
	return layout.LayoutResult{Size: e.size}
}

func (e *BadgeElement) displayText() string {
	b := e.badge
	if b.Max > 0 {
		if n, err := strconv.Atoi(b.Value); err == nil && n > b.Max {
			return strconv.Itoa(b.Max) + "+"
		}
	}
	return b.Value
}

func (e *BadgeElement) Paint(cvs canvas.Canvas, offset types.Point) {
	if e.childEl != nil {
		e.childEl.Paint(cvs, offset)
	}
	b := e.badge
	if b.Hidden {
		return
	}
	pos := e.Offset()
	color := badgeColor(b.Type)
	// 角标锚点：从外接矩形右上角朝圆心方向内移，落到圆形头像的右上圆周(约 45°)，
	// 避免红点/角标悬浮在圆外。内移量 = R(1−cos45°)，按较短边换算 ≈ side·0.146。
	side := e.size.Width
	if e.size.Height < side {
		side = e.size.Height
	}
	inset := side * 0.146
	anchorX := pos.X + e.size.Width - inset
	anchorY := pos.Y + inset

	if b.IsDot {
		dot := paint.DefaultPaint()
		dot.Color = color
		cvs.DrawCircle(anchorX, anchorY, 5, dot)
		return
	}

	text := e.displayText()
	if text == "" || (text == "0" && !b.ShowZero) {
		return
	}
	font := canvas.DefaultFont()
	font.Size = 12
	tw := canvas.MeasureTextGlobal(text, font).Width
	h := 18.0
	w := tw + 12
	if w < h {
		w = h // 单字符呈圆形
	}
	// 左端圆角对齐圆周点，pill 向右延伸：宽角标(如 99+)往外长而非向左深入遮盖头像；
	// 单字符(w==h)时左端 = 中心-半径，恰好居中于圆周点。
	bx := anchorX - h/2
	by := anchorY - h/2
	bg := paint.DefaultPaint()
	bg.Color = color
	cvs.DrawRoundedRect(bx, by, w, h, h/2, bg)
	canvas.DrawTextAligned(cvs, text, types.Rect{X: bx, Y: by, Width: w, Height: h}, font, types.ColorWhite, canvas.HAlignCenter, canvas.VAlignMiddle)
}

func (e *BadgeElement) Update(newWidget Widget) {
	if nb, ok := newWidget.(*Badge); ok {
		e.badge = nb
		e.BaseElement.widget = newWidget
		e.dirty = true
	}
}
