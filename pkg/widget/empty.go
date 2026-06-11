package widget

import (
	"github.com/hoonfeng/goui/pkg/canvas"
	"github.com/hoonfeng/goui/internal/layout"
	"github.com/hoonfeng/goui/pkg/paint"
	"github.com/hoonfeng/goui/pkg/types"
)

// ─── Empty 空状态 ──────────────────────────────────────────────
// 1:1 复刻 el-empty：居中的空状态插画 + 描述文字 + 可选底部操作(children)。
//   props: description / image-size / (children 作为底部操作区)

// Empty 空状态。
type Empty struct {
	StatelessWidget
	Description string
	ImageSize   float64
	Children    []Widget
}

func NewEmpty(desc string) *Empty {
	return &Empty{Description: desc, ImageSize: 80}
}
func (e *Empty) WithImageSize(s float64) *Empty   { e.ImageSize = s; return e }
func (e *Empty) WithChildren(ws ...Widget) *Empty { e.Children = ws; return e }

func (em *Empty) CreateElement() Element {
	return &EmptyElement{BaseElement: BaseElement{widget: em}, empty: em}
}

type EmptyElement struct {
	BaseElement
	empty    *Empty
	childEls []Element
}

func (e *EmptyElement) Build() []Element {
	e.childEls = nil
	for i, w := range e.empty.Children {
		ce := CreateElementFor(w)
		ce.Mount(e, i)
		e.childEls = append(e.childEls, ce)
	}
	e.children = e.childEls
	return e.children
}

func (e *EmptyElement) imgSize() float64 {
	s := e.empty.ImageSize
	if s <= 0 {
		s = 80
	}
	return s
}

func (e *EmptyElement) Layout(ctx *layout.LayoutContext) layout.LayoutResult {
	w := ctx.Constraints.MaxWidth
	if w <= 0 || w > 1e6 {
		w = 300
	}
	img := e.imgSize()
	descH := canvas.GetFaceLineHeight(14) + 10
	cx := w / 2
	childTop := img + descH + 8
	childH := 0.0
	for _, ce := range e.childEls {
		res := ce.Layout(&layout.LayoutContext{Constraints: layout.BoxConstraints{
			MinWidth: 0, MaxWidth: w, MinHeight: 0, MaxHeight: 10000,
		}})
		ce.SetPosition(types.Point{X: cx - res.Size.Width/2, Y: childTop + childH})
		childH += res.Size.Height + 8
	}
	h := img + descH + childH + 16
	e.size = ctx.Constraints.Constrain(types.Size{Width: w, Height: h})
	return layout.LayoutResult{Size: e.size}
}

func (e *EmptyElement) Paint(cvs canvas.Canvas, offset types.Point) {
	pos := e.Offset()
	img := e.imgSize()
	cx := pos.X + e.size.Width/2

	gray := elBorder()
	grayLight := types.ColorFromRGB(238, 240, 244)

	// 底座(浅灰扁圆角矩形，模拟阴影)
	basep := paint.DefaultPaint()
	basep.Color = grayLight
	cvs.DrawRoundedRect(cx-img*0.45, pos.Y+img*0.84, img*0.9, img*0.12, img*0.06, basep)

	// 空盒子轮廓(梯形 + 盖口线)
	op := paint.DefaultStrokePaint()
	op.Color = gray
	op.StrokeWidth = 2
	topY := pos.Y + img*0.28
	botY := pos.Y + img*0.8
	leftX := cx - img*0.3
	rightX := cx + img*0.3
	path := canvas.NewPath()
	path.MoveTo(leftX, topY)
	path.LineTo(rightX, topY)
	path.LineTo(rightX+img*0.08, botY)
	path.LineTo(leftX-img*0.08, botY)
	path.Close()
	cvs.DrawPath(path, op)
	cvs.DrawLine(cx-img*0.16, topY, cx+img*0.16, topY, op)

	// description（用户没传则用 i18n 默认「暂无数据/No Data」，随当前语言变）
	if desc := orI18n(e.empty.Description, "el.empty.description"); desc != "" {
		font := canvas.DefaultFont()
		font.Size = 14
		dp := paint.DefaultPaint()
		dp.Color = elTextSecondary()
		tw := canvas.MeasureTextGlobal(desc, font).Width
		descY := pos.Y + img + canvas.GetFaceAscent(14) + 8
		cvs.DrawText(desc, cx-tw/2, descY, font, dp)
	}

	// children(底部操作)
	for _, ce := range e.childEls {
		ce.Paint(cvs, offset)
	}
}

func (e *EmptyElement) Update(newWidget Widget) {
	if ne, ok := newWidget.(*Empty); ok {
		e.empty = ne
		e.BaseElement.widget = newWidget
		e.dirty = true
	}
}
