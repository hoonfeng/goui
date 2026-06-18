//go:build !cgo

// Package canvas 提供了 2D 绘制接口。
// 此文件为非 CGO 构建提供 SkiaCanvas 的桩实现。
package canvas

import (
	"image"

	"github.com/hoonfeng/goui/pkg/paint"
	"github.com/hoonfeng/goui/pkg/types"
)

// SkiaCanvas 的桩——CGO 不可用时调用会 panic。
type SkiaCanvas struct {
	w, h int
	img  *image.RGBA
}

func NewSkiaCanvas(width, height int) *SkiaCanvas {
	// 非 CGO 下创建一个最小功能的实例（SoftCanvas 代替）
	c := &SkiaCanvas{w: width, h: height}
	c.img = image.NewRGBA(image.Rect(0, 0, width, height))
	return c
}

func NewSkiaCanvasGPU(width, height int, fboID uint32) (*SkiaCanvas, error) {
	return NewSkiaCanvas(width, height), nil
}

func (c *SkiaCanvas) Release()           {}
func (c *SkiaCanvas) Image() *image.RGBA { return c.img }
func (c *SkiaCanvas) Clear(col types.Color) {
	c.img = image.NewRGBA(image.Rect(0, 0, c.w, c.h))
}
func (c *SkiaCanvas) Save()                                                        {}
func (c *SkiaCanvas) SaveLayerAlpha(alpha float64)                                 {}
func (c *SkiaCanvas) Restore()                                                     {}
func (c *SkiaCanvas) Translate(x, y float64)                                       {}
func (c *SkiaCanvas) Rotate(angle float64)                                         {}
func (c *SkiaCanvas) Scale(x, y float64)                                           {}
func (c *SkiaCanvas) Clip(path *Path)                                              {}
func (c *SkiaCanvas) ClipRect(x, y, w, h float64)                                  {}
func (c *SkiaCanvas) DrawPath(path *Path, p paint.Paint)                           {}
func (c *SkiaCanvas) DrawRect(x, y, w, h float64, p paint.Paint)                   {}
func (c *SkiaCanvas) DrawRoundedRect(x, y, w, h, r float64, p paint.Paint)         {}
func (c *SkiaCanvas) DrawCircle(cx, cy, radius float64, p paint.Paint)             {}
func (c *SkiaCanvas) DrawLine(x1, y1, x2, y2 float64, p paint.Paint)               {}
func (c *SkiaCanvas) DrawText(text string, x, y float64, font Font, p paint.Paint) {}
func (c *SkiaCanvas) DrawImage(img image.Image, dst types.Rect)                    {}
func (c *SkiaCanvas) MeasureText(text string, font Font) TextMetrics               { return TextMetrics{} }
func (c *SkiaCanvas) Width() float64                                               { return float64(c.w) }
func (c *SkiaCanvas) Height() float64                                              { return float64(c.h) }
func (c *SkiaCanvas) Flush() error                                                 { return nil }
func (c *SkiaCanvas) Resize(width, height int) {
	c.w = width
	c.h = height
	c.img = image.NewRGBA(image.Rect(0, 0, width, height))
}

// fontVMetrics 返回字体垂直度量（桩实现：估算值）。
func (c *SkiaCanvas) fontVMetrics(font Font) (ascent, descent, lineHeight float64) {
	s := font.Size
	if s <= 0 {
		s = 16
	}
	return s * 0.8, s * 0.2, s * 1.2
}

// getGlyphChecker 返回字形检测器（桩实现：返回 nil，假设全覆盖）。
func (c *SkiaCanvas) getGlyphChecker() GlyphChecker { return nil }
