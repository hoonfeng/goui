// Package paint 定义了绘制样式类型，包括颜色、渐变、阴影等。
package paint

import "github.com/hoonfeng/goui/pkg/types"

// Style 表示填充或描边的样式类型
type Style int

const (
	StyleFill   Style = iota // 填充
	StyleStroke              // 描边
	StyleFillAndStroke       // 填充 + 描边
)

// Paint 定义绘制样式（画笔/画刷）
type Paint struct {
	Style        Style
	Color        types.Color
	StrokeWidth  float64
	AntiAlias    bool
	Opacity      float64 // 透明度 0.0-1.0
	LinearGradient *Gradient
	RadialGradient *Gradient
}

// DefaultPaint 返回默认的填充画笔
func DefaultPaint() Paint {
	return Paint{
		Style:       StyleFill,
		Color:       types.ColorBlack,
		StrokeWidth: 1.0,
		AntiAlias:   true,
		Opacity:     1.0,
	}
}

// DefaultStrokePaint 返回默认的描边画笔
func DefaultStrokePaint() Paint {
	return Paint{
		Style:       StyleStroke,
		Color:       types.ColorBlack,
		StrokeWidth: 1.0,
		AntiAlias:   true,
		Opacity:     1.0,
	}
}

// StrokePaint 便捷构造指定颜色与线宽的描边画笔。
func StrokePaint(color types.Color, width float64) Paint {
	p := DefaultStrokePaint()
	p.Color = color
	p.StrokeWidth = width
	return p
}

// GradientType 渐变类型
type GradientType int

const (
	GradientLinear GradientType = iota
	GradientRadial
)

// Gradient 定义渐变
type Gradient struct {
	Type     GradientType
	Start    types.Point
	End      types.Point
	Stops    []ColorStop
	Center   types.Point // 径向渐变的中心
	Radius   float64     // 径向渐变的半径
}

// ColorStop 定义渐变中的颜色停止点
type ColorStop struct {
	Offset float64    // 0.0-1.0
	Color  types.Color
}

// Shadow 定义阴影样式
type Shadow struct {
	Offset types.Point
	Blur   float64
	Color  types.Color
}

// BorderSide 定义边框的一条边
type BorderSide struct {
	Color types.Color
	Width float64
}

// Border 定义四边边框
type Border struct {
	Top    BorderSide
	Right  BorderSide
	Bottom BorderSide
	Left   BorderSide
}

// EdgeInsets 返回边框占据的内边距
func (b Border) EdgeInsets() types.Inset {
	return types.Inset{
		Top:    b.Top.Width,
		Right:  b.Right.Width,
		Bottom: b.Bottom.Width,
		Left:   b.Left.Width,
	}
}

// AllBorder 创建统一的四边边框
func AllBorder(color types.Color, width float64) Border {
	side := BorderSide{Color: color, Width: width}
	return Border{Top: side, Right: side, Bottom: side, Left: side}
}
