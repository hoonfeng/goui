package widget

import (
	"github.com/hoonfeng/goui/pkg/canvas"
	"github.com/hoonfeng/goui/internal/layout"
	"github.com/hoonfeng/goui/pkg/paint"
	"github.com/hoonfeng/goui/pkg/types"
)

// ─── Divider 分割线（对应 Element Plus el-divider）─────────────
// 水平/垂直分割线；el-divider 的"带文字标题/位置"特性为后续增强。

// DividerDirection 分割线方向
type DividerDirection int

const (
	DividerHorizontal DividerDirection = iota // 水平分割线
	DividerVertical                           // 垂直分割线
)

// Divider 分割线控件，用于在布局中分隔内容区域。
type Divider struct {
	StatelessWidget
	Direction DividerDirection // 方向
	Color     types.Color      // 线条颜色
	Thickness float64          // 线条粗细
	Indent    float64          // 起始缩进
	EndIndent float64          // 末端缩进
	Margin    types.Inset      // 外边距
}

// DefaultDivider 返回默认配置的水平分割线
func DefaultDivider() *Divider {
	return &Divider{
		Direction: DividerHorizontal,
		Color:     types.ColorFromRGB(220, 220, 220),
		Thickness: 1,
	}
}

// CreateElement 创建 DividerElement
func (d *Divider) CreateElement() Element {
	return &DividerElement{
		BaseElement: BaseElement{widget: d},
		divider:     d,
	}
}

// DividerElement 分割线的运行时 Element
type DividerElement struct {
	BaseElement
	divider *Divider
}

// Build 返回空（无子控件）
func (e *DividerElement) Build() []Element { return nil }

// Layout 测量分割线尺寸
func (e *DividerElement) Layout(ctx *layout.LayoutContext) layout.LayoutResult {
	d := e.divider
	m := d.Margin
	thick := d.Thickness
	if thick <= 0 {
		thick = 1
	}

	if d.Direction == DividerHorizontal {
		// 水平分割线：高度 = 厚度 + 上下边距
		height := thick + m.Top + m.Bottom
		width := ctx.Constraints.MaxWidth - m.Left - m.Right
		if width < 0 {
			width = 0
		}
		e.size = ctx.Constraints.Constrain(types.Size{
			Width:  width,
			Height: height,
		})
	} else {
		// 垂直分割线：宽度 = 厚度 + 左右边距
		width := thick + m.Left + m.Right
		height := ctx.Constraints.MaxHeight - m.Top - m.Bottom
		if height < 0 {
			height = 0
		}
		e.size = ctx.Constraints.Constrain(types.Size{
			Width:  width,
			Height: height,
		})
	}
	return layout.LayoutResult{Size: e.size}
}

// Paint 绘制分割线
func (e *DividerElement) Paint(cvs canvas.Canvas, offset types.Point) {
	pos := e.Offset()
	d := e.divider
	m := d.Margin

	linePaint := paint.DefaultStrokePaint()
	linePaint.Color = d.Color
	linePaint.StrokeWidth = d.Thickness

	indent := d.Indent
	endIndent := d.EndIndent

	if d.Direction == DividerHorizontal {
		x1 := pos.X + m.Left + indent
		x2 := pos.X + e.size.Width - m.Right - endIndent
		y := pos.Y + m.Top + d.Thickness/2
		if x2 > x1 {
			cvs.DrawLine(x1, y, x2, y, linePaint)
		}
	} else {
		y1 := pos.Y + m.Top + indent
		y2 := pos.Y + e.size.Height - m.Bottom - endIndent
		x := pos.X + m.Left + d.Thickness/2
		if y2 > y1 {
			cvs.DrawLine(x, y1, x, y2, linePaint)
		}
	}
}
