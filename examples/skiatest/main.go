//go:build ignore
// +build ignore

package main

import (
	"fmt"
	"os"

	"github.com/user/goui/internal/canvas"
	"github.com/user/goui/internal/paint"
	"github.com/user/goui/internal/types"
)

func main() {
	// 创建 SkiaCanvas
	width, height := 400, 400
	sk := canvas.NewSkiaCanvas(width, height)
	defer sk.Release()

	// 清除为浅灰色背景
	sk.Save()
	sk.DrawRect(0, 0, float64(width), float64(height), paint.Paint{
		Style: paint.StyleFill,
		Color: types.Color{R: 240, G: 240, B: 248, A: 255},
	})
	sk.Restore()

	// ---- 1. 矩形填充 ----
	fillPaint := paint.DefaultPaint()
	fillPaint.Color = types.Color{R: 66, G: 133, B: 244, A: 255} // 蓝色
	sk.DrawRect(20, 20, 80, 80, fillPaint)

	// ---- 2. 圆角矩形 ----
	roundPaint := paint.DefaultPaint()
	roundPaint.Color = types.Color{R: 52, G: 168, B: 83, A: 255} // 绿色
	sk.DrawRoundedRect(120, 20, 80, 80, 16, roundPaint)

	// ---- 3. 圆形 ----
	circlePaint := paint.DefaultPaint()
	circlePaint.Color = types.Color{R: 234, G: 67, B: 53, A: 255} // 红色
	sk.DrawCircle(270, 60, 40, circlePaint)

	// ---- 4. 描边矩形 ----
	strokePaint := paint.DefaultStrokePaint()
	strokePaint.Color = types.Color{R: 251, G: 188, B: 4, A: 255} // 黄色
	strokePaint.StrokeWidth = 4
	sk.DrawRect(20, 140, 80, 80, strokePaint)

	// ---- 5. 描边圆角 ----
	rStrokePaint := paint.DefaultStrokePaint()
	rStrokePaint.Color = types.Color{R: 142, G: 68, B: 173, A: 255} // 紫色
	rStrokePaint.StrokeWidth = 4
	sk.DrawRoundedRect(120, 140, 80, 80, 16, rStrokePaint)

	// ---- 6. 描边圆形 ----
	cStrokePaint := paint.DefaultStrokePaint()
	cStrokePaint.Color = types.Color{R: 0, G: 150, B: 136, A: 255} // 青色
	cStrokePaint.StrokeWidth = 4
	sk.DrawCircle(270, 180, 40, cStrokePaint)

	// ---- 7. 线条 ----
	linePaint := paint.DefaultStrokePaint()
	linePaint.Color = types.Color{R: 66, G: 133, B: 244, A: 255}
	linePaint.StrokeWidth = 3
	sk.DrawLine(20, 280, 180, 280, linePaint)
	sk.DrawLine(20, 280, 100, 240, linePaint)
	sk.DrawLine(100, 240, 180, 280, linePaint)

	// ---- 8. 文本 ----
	textPaint := paint.DefaultPaint()
	textPaint.Color = types.Color{R: 33, G: 33, B: 33, A: 255}
	textFont := canvas.Font{
		Family: "sans-serif",
		Size:   20,
		Weight: canvas.FontWeightNormal,
	}
	sk.DrawText("Hello Skia!", 20, 340, textFont, textPaint)

	// ---- 9. 路径 ----
	path := &canvas.Path{}
	path.MoveTo(220, 280)
	path.LineTo(280, 320)
	path.LineTo(340, 270)
	path.LineTo(310, 340)
	path.LineTo(250, 350)
	path.Close()
	pathPaint := paint.DefaultPaint()
	pathPaint.Color = types.Color{R: 255, G: 87, B: 34, A: 221}
	sk.DrawPath(path, pathPaint)

	// ---- 10. 多行文本 ----
	smallFont := canvas.Font{
		Family: "sans-serif",
		Size:   14,
		Weight: canvas.FontWeightNormal,
	}
	smallFont.Size = 14
	sk.DrawText("SkiaCanvas rendering test", 20, 380, smallFont, textPaint)

	// 保存为 PNG
	if err := sk.SaveToPNG("skia_render_test.png"); err != nil {
		fmt.Fprintf(os.Stderr, "SaveToPNG failed: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("OK: skia_render_test.png saved")
}
