package canvas

import (
	"image"

	"github.com/hoonfeng/goui/pkg/paint"
	"github.com/hoonfeng/goui/pkg/types"
)

// Canvas 提供 2D 绘制接口，类似 HTML5 Canvas API。
// 它是所有绘制操作的核心入口。
type Canvas interface {
	// Save/Restore 保存和恢复画布状态栈
	Save()
	Restore()

	// SaveLayerAlpha 开启一个带整体透明度的离屏图层，后续绘制在该图层进行；
	// 配对的 Restore() 会以 alpha(0~1) 把整组绘制合成回画布。用于组透明度/淡入淡出。
	SaveLayerAlpha(alpha float64)

	// Translate/Rotate/Scale 画布变换
	Translate(x, y float64)
	Rotate(angle float64)
	Scale(x, y float64)

	// Clip 裁剪路径
	Clip(path *Path)

	// ClipRect 裁剪到指定矩形区域（快捷方法）
	ClipRect(x, y, w, h float64)

	// DrawPath 绘制路径
	DrawPath(path *Path, p paint.Paint)

	// DrawRect 绘制矩形
	DrawRect(x, y, w, h float64, p paint.Paint)

	// DrawRoundedRect 绘制圆角矩形
	DrawRoundedRect(x, y, w, h, r float64, p paint.Paint)

	// DrawCircle 绘制圆形
	DrawCircle(cx, cy, radius float64, p paint.Paint)

	// DrawLine 绘制直线
	DrawLine(x1, y1, x2, y2 float64, p paint.Paint)

	// DrawText 绘制文本
	DrawText(text string, x, y float64, font Font, p paint.Paint)

	// DrawImage 绘制图像
	DrawImage(img image.Image, dst types.Rect)

	// MeasureText 测量文本尺寸
	MeasureText(text string, font Font) TextMetrics

	// Width/Height 返回画布尺寸
	Width() float64
	Height() float64

	// Flush 将绘制内容输出到目标
	Flush() error
}

// CanvasState 保存画布状态（变换矩阵、裁剪区域等）
type CanvasState struct {
	Matrix types.Matrix4
}
