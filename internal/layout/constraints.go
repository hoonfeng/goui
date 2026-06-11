// Package layout 定义了布局引擎的核心类型。
package layout

import "github.com/hoonfeng/goui/pkg/types"

// BoxConstraints 是父控件对子控件的尺寸约束。
// 类似于 Flutter 的 BoxConstraints。
type BoxConstraints struct {
	MinWidth  float64
	MaxWidth  float64
	MinHeight float64
	MaxHeight float64
}

// NewBoxConstraints 创建新的约束
func NewBoxConstraints(minW, maxW, minH, maxH float64) BoxConstraints {
	return BoxConstraints{
		MinWidth:  minW,
		MaxWidth:  maxW,
		MinHeight: minH,
		MaxHeight: maxH,
	}

}

// Tight 创建紧约束（固定尺寸）
func Tight(width, height float64) BoxConstraints {
	return BoxConstraints{
		MinWidth:  width,
		MaxWidth:  width,
		MinHeight: height,
		MaxHeight: height,
	}
}

// Loose 创建松约束（0 到指定尺寸）
func Loose(width, height float64) BoxConstraints {
	return BoxConstraints{
		MinWidth:  0,
		MaxWidth:  width,
		MinHeight: 0,
		MaxHeight: height,
	}
}

// Expand 创建扩展约束（尽可能大）
func Expand() BoxConstraints {
	return BoxConstraints{
		MinWidth:  float64(1<<31 - 1), // 近似无限
		MaxWidth:  float64(1<<31 - 1),
		MinHeight: float64(1<<31 - 1),
		MaxHeight: float64(1<<31 - 1),
	}
}

// TightFor 根据尺寸创建紧约束
func TightFor(size types.Size) BoxConstraints {
	return Tight(size.Width, size.Height)
}

// ClampWidth 将宽度约束在范围内
func (c BoxConstraints) ClampWidth(width float64) float64 {
	if width < c.MinWidth {
		return c.MinWidth
	}
	if width > c.MaxWidth {
		return c.MaxWidth
	}
	return width
}

// ClampHeight 将高度约束在范围内
func (c BoxConstraints) ClampHeight(height float64) float64 {
	if height < c.MinHeight {
		return c.MinHeight
	}
	if height > c.MaxHeight {
		return c.MaxHeight
	}
	return height
}

// Constrain 将 Size 约束在范围内
func (c BoxConstraints) Constrain(size types.Size) types.Size {
	return types.Size{
		Width:  c.ClampWidth(size.Width),
		Height: c.ClampHeight(size.Height),
	}
}

// IsTight 检查是否为紧约束
func (c BoxConstraints) IsTight() bool {
	return c.MinWidth == c.MaxWidth && c.MinHeight == c.MaxHeight
}

// IsBounded 检查是否为有界约束
func (c BoxConstraints) IsBounded() bool {
	return c.MaxWidth < float64(1<<30) && c.MaxHeight < float64(1<<30)
}

// Deflate 减去内边距后的约束
func (c BoxConstraints) Deflate(inset types.Inset) BoxConstraints {
	return BoxConstraints{
		MinWidth:  max(0, c.MinWidth-inset.Left-inset.Right),
		MaxWidth:  max(0, c.MaxWidth-inset.Left-inset.Right),
		MinHeight: max(0, c.MinHeight-inset.Top-inset.Bottom),
		MaxHeight: max(0, c.MaxHeight-inset.Top-inset.Bottom),
	}
}

// Unbounded 创建无约束约束
func Unbounded() BoxConstraints {
	return BoxConstraints{
		MinWidth:  0,
		MaxWidth:  float64(1<<31 - 1),
		MinHeight: 0,
		MaxHeight: float64(1<<31 - 1),
	}
}

func max(a, b float64) float64 {
	if a > b {
		return a
	}
	return b
}

func min(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}
