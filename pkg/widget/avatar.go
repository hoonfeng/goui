package widget

import (
	"github.com/hoonfeng/goui/pkg/canvas"
	"github.com/hoonfeng/goui/internal/layout"
	"github.com/hoonfeng/goui/pkg/paint"
	"github.com/hoonfeng/goui/pkg/types"
)

// ─── Avatar 头像 ───────────────────────────────────────────────
// 1:1 复刻 el-avatar 核心：circle/square 形状，文字或图标内容，可定制尺寸/配色。
//   props: shape / size / (文字 or 图标) / bg-color
//   注: 图片头像留待 Image 接入后补，本版支持文字/图标。

type AvatarShape int

const (
	AvatarCircle AvatarShape = iota // circle（默认）
	AvatarSquare                    // square
)

// Avatar 头像。
type Avatar struct {
	StatelessWidget
	Text      string // 文字内容(如姓名首字)
	Icon      string // 图标字符(与 Text 二选一)
	Shape     AvatarShape
	Size      float64 // 0 → 默认 40
	BgColor   types.Color
	TextColor types.Color
}

func NewAvatar(text string) *Avatar {
	return &Avatar{Text: text, Shape: AvatarCircle}
}
func (a *Avatar) WithShape(s AvatarShape) *Avatar   { a.Shape = s; return a }
func (a *Avatar) WithSize(s float64) *Avatar        { a.Size = s; return a }
func (a *Avatar) WithBgColor(c types.Color) *Avatar { a.BgColor = c; return a }
func (a *Avatar) WithIcon(s string) *Avatar         { a.Icon = s; return a }

func (a *Avatar) CreateElement() Element {
	return &AvatarElement{BaseElement: BaseElement{widget: a}, avatar: a}
}

type AvatarElement struct {
	BaseElement
	avatar *Avatar
}

func (e *AvatarElement) Build() []Element { return nil }

func (e *AvatarElement) sideLen() float64 {
	s := e.avatar.Size
	if s <= 0 {
		s = 40
	}
	return s
}

func (e *AvatarElement) Layout(ctx *layout.LayoutContext) layout.LayoutResult {
	s := e.sideLen()
	e.size = ctx.Constraints.Constrain(types.Size{Width: s, Height: s})
	return layout.LayoutResult{Size: e.size}
}

func (e *AvatarElement) Paint(cvs canvas.Canvas, offset types.Point) {
	pos := e.Offset()
	a := e.avatar
	s := e.sideLen()

	bg := a.BgColor
	if bg == (types.Color{}) {
		bg = elPlaceholder() // Element 默认灰
	}
	bgp := paint.DefaultPaint()
	bgp.Color = bg
	if a.Shape == AvatarCircle {
		cvs.DrawCircle(pos.X+s/2, pos.Y+s/2, s/2, bgp)
	} else {
		cvs.DrawRoundedRect(pos.X, pos.Y, s, s, 4, bgp)
	}

	content := a.Text
	if content == "" {
		content = a.Icon
	}
	if content == "" {
		return
	}
	fg := a.TextColor
	if fg == (types.Color{}) {
		fg = types.ColorWhite
	}
	font := canvas.DefaultFont()
	font.Size = s * 0.4
	canvas.DrawTextAligned(cvs, content, types.Rect{X: pos.X, Y: pos.Y, Width: s, Height: s}, font, fg, canvas.HAlignCenter, canvas.VAlignMiddle)
}

func (e *AvatarElement) Update(newWidget Widget) {
	if na, ok := newWidget.(*Avatar); ok {
		e.avatar = na
		e.BaseElement.widget = newWidget
		e.dirty = true
	}
}
