// Package render 定义了渲染管线的核心类型。
package render

import (
	"github.com/hoonfeng/goui/pkg/canvas"
	"github.com/hoonfeng/goui/pkg/types"
)

// Layer 是渲染图层，管理需要重绘的区域
type Layer struct {
	Bounds      types.Rect
	Children    []*Layer
	Canvas      canvas.Canvas
	DirtyRegion types.Rect // 脏区域
	Offset      types.Point
}

// NewLayer 创建新图层
func NewLayer(bounds types.Rect) *Layer {
	return &Layer{
		Bounds:   bounds,
		Children: make([]*Layer, 0),
	}
}

// AddChild 添加子图层
func (l *Layer) AddChild(child *Layer) {
	l.Children = append(l.Children, child)
}

// MarkDirty 标记区域为脏
func (l *Layer) MarkDirty(region types.Rect) {
	if l.DirtyRegion.Width == 0 && l.DirtyRegion.Height == 0 {
		l.DirtyRegion = region
	} else {
		// 合并脏区域
		minX := min(l.DirtyRegion.X, region.X)
		minY := min(l.DirtyRegion.Y, region.Y)
		maxX := max(l.DirtyRegion.X+l.DirtyRegion.Width, region.X+region.Width)
		maxY := max(l.DirtyRegion.Y+l.DirtyRegion.Height, region.Y+region.Height)
		l.DirtyRegion = types.Rect{
			X: minX, Y: minY,
			Width: maxX - minX, Height: maxY - minY,
		}
	}
}

// ClearDirty 清除脏区域标记
func (l *Layer) ClearDirty() {
	l.DirtyRegion = types.Rect{}
}

// PaintContext 绘制上下文
type PaintContext struct {
	Canvas   canvas.Canvas
	Offset   types.Point
	ClipRect *types.Rect
	Opacity  float64
}

// RenderObject 是所有可渲染对象的接口
type RenderObject interface {
	// Paint 绘制到画布
	Paint(ctx *PaintContext)

	// Layout 布局
	Layout(constraints interface{}) types.Size

	// HitTest 命中测试
	HitTest(point types.Point) bool

	// Bounds 返回边界
	Bounds() types.Rect
}

func min(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}

func max(a, b float64) float64 {
	if a > b {
		return a
	}
	return b
}
