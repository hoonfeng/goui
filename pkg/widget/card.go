package widget

import (
	"github.com/hoonfeng/goui/pkg/canvas"
	"github.com/hoonfeng/goui/internal/layout"
	"github.com/hoonfeng/goui/pkg/paint"
	"github.com/hoonfeng/goui/pkg/types"
)

// Card 卡片容器控件（对应 Element Plus el-card）。
// 包含圆角、阴影效果、背景色和内边距；视觉偏 Material，
// el-card 的 header/body/footer 三段可按需用 Div 拼。
type Card struct {
	SingleChildWidget
	// 注意：没有独立的 Child 字段，使用嵌入的 SingleChildWidget.Child
	Color        types.Color  // 背景色
	ShadowColor  types.Color  // 阴影颜色
	ShadowOffset types.Point  // 阴影偏移
	ShadowBlur   float64      // 阴影模糊半径
	Radius       float64      // 圆角半径
	Padding      types.Inset  // 内边距
	BorderColor  *types.Color // 边框颜色（nil=无边框）
	BorderWidth  float64      // 边框宽度
	Width        float64      // 固定宽度（0=自适应）
	Height       float64      // 固定高度（0=自适应）
	Elevation    int          // 阴影层级（0=无阴影，1-5 预定义层级）
}

// DefaultCard 返回默认配置的卡片
func DefaultCard() *Card {
	return &Card{
		Color:        types.ColorWhite,
		Radius:       8,
		Elevation:    1,
		ShadowColor:  types.ColorFromRGBA(0, 0, 0, 30),
		ShadowOffset: types.Point{X: 0, Y: 2},
		ShadowBlur:   8,
	}
}

// CreateElement 创建 CardElement
func (c *Card) CreateElement() Element {
	return &CardElement{
		BaseElement: BaseElement{widget: c},
		card:        c,
	}
}

// Update 更新 Widget 配置并同步内部 card 引用。
func (e *CardElement) Update(newWidget Widget) {
	e.widget = newWidget
	e.dirty = true
	if c, ok := newWidget.(*Card); ok {
		e.card = c
	}
}

// CardElement 卡片的运行时 Element
type CardElement struct {
	BaseElement
	card  *Card
	child Element
}

// Build 构建子 Element
func (e *CardElement) Build() []Element {
	if e.card.Child != nil {
		e.child = CreateElementFor(e.card.Child)
		e.child.Mount(e, 0)
		e.children = []Element{e.child}
		return e.children
	}
	e.children = nil
	return nil
}

// Layout 执行布局
func (e *CardElement) Layout(ctx *layout.LayoutContext) layout.LayoutResult {
	c := e.card

	// 计算卡片内部可用空间
	pad := c.Padding
	childConstraints := ctx.Constraints
	childConstraints.MaxWidth -= pad.Left + pad.Right
	childConstraints.MaxHeight -= pad.Top + pad.Bottom

	if c.Width > 0 {
		childConstraints.MaxWidth = c.Width - pad.Left - pad.Right
	}
	if c.Height > 0 {
		childConstraints.MaxHeight = c.Height - pad.Top - pad.Bottom
	}
	childConstraints.MinWidth = 0
	childConstraints.MinHeight = 0

	if e.child != nil {
		result := e.child.Layout(&layout.LayoutContext{Constraints: childConstraints})
		e.size = types.Size{
			Width:  result.Size.Width + pad.Left + pad.Right,
			Height: result.Size.Height + pad.Top + pad.Bottom,
		}
	} else {
		w := ctx.Constraints.MaxWidth
		h := ctx.Constraints.MaxHeight
		if c.Width > 0 {
			w = c.Width
		}
		if c.Height > 0 {
			h = c.Height
		}
		e.size = types.Size{Width: w, Height: h}
	}

	// 应用固定尺寸
	if c.Width > 0 {
		e.size.Width = c.Width
	}
	if c.Height > 0 {
		e.size.Height = c.Height
	}

	e.size = ctx.Constraints.Constrain(e.size)
	return layout.LayoutResult{Size: e.size}
}

// Paint 绘制卡片
func (e *CardElement) Paint(cvs canvas.Canvas, offset types.Point) {
	pos := e.Offset()
	c := e.card

	radius := c.Radius
	if radius < 0 {
		radius = 0
	}

	// 绘制阴影（通过多次叠加半透明矩形模拟模糊阴影）
	if c.Elevation > 0 {
		shadowColor := c.ShadowColor
		if shadowColor == (types.Color{}) {
			shadowColor = types.ColorFromRGBA(0, 0, 0, 30)
		}
		shadowOffset := c.ShadowOffset
		if shadowOffset == (types.Point{}) {
			shadowOffset = types.Point{X: 0, Y: 2}
		}
		shadowBlur := c.ShadowBlur
		if shadowBlur <= 0 {
			shadowBlur = 8
		}

		// 简化阴影：绘制多个半透明偏移层
		layers := int(shadowBlur / 2)
		if layers < 3 {
			layers = 3
		}
		for i := layers; i >= 0; i-- {
			alpha := uint8(8 + i*3)
			if alpha > 40 {
				alpha = 40
			}
			shadowPaint := paint.DefaultPaint()
			shadowPaint.Color = types.ColorFromRGBA(0, 0, 0, alpha)
			offsetY := shadowOffset.Y * float64(layers-i) / float64(layers)
			offsetX := shadowOffset.X * float64(layers-i) / float64(layers)
			shrink := float64(layers-i) * 0.5

			if radius > 0 {
				cvs.DrawRoundedRect(
					pos.X+offsetX+shrink, pos.Y+offsetY+shrink,
					e.size.Width-shrink*2, e.size.Height-shrink*2,
					radius, shadowPaint,
				)
			} else {
				cvs.DrawRect(
					pos.X+offsetX+shrink, pos.Y+offsetY+shrink,
					e.size.Width-shrink*2, e.size.Height-shrink*2,
					shadowPaint,
				)
			}
		}
	}

	// 绘制卡片背景
	bgPaint := paint.DefaultPaint()
	bgPaint.Color = c.Color
	if radius > 0 {
		cvs.DrawRoundedRect(pos.X, pos.Y, e.size.Width, e.size.Height, radius, bgPaint)
	} else {
		cvs.DrawRect(pos.X, pos.Y, e.size.Width, e.size.Height, bgPaint)
	}

	// 绘制边框
	if c.BorderColor != nil {
		borderPaint := paint.DefaultStrokePaint()
		borderPaint.Color = *c.BorderColor
		borderPaint.StrokeWidth = c.BorderWidth
		if borderPaint.StrokeWidth <= 0 {
			borderPaint.StrokeWidth = 1
		}
		if radius > 0 {
			cvs.DrawRoundedRect(pos.X, pos.Y, e.size.Width, e.size.Height, radius, borderPaint)
		} else {
			cvs.DrawRect(pos.X, pos.Y, e.size.Width, e.size.Height, borderPaint)
		}
	}

	// 绘制子控件（偏移内边距）
	if e.child != nil {
		childOffset := types.Point{
			X: pos.X + c.Padding.Left,
			Y: pos.Y + c.Padding.Top,
		}
		e.child.Paint(cvs, childOffset)
	}
}
