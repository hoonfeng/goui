// skiapaint — 全面的 SkiaCanvas 功能测试 Demo
//
// 本程序全面测试 SkiaCanvas 的所有绘制功能，包括：
//   - 基本形状绘制（矩形、圆角矩形、圆形、椭圆、直线）
//   - 路径绘制（贝塞尔曲线、复合路径）
//   - 文本渲染（多种字体、字号、颜色）
//   - 渐变填充（线性渐变、径向渐变）
//   - 画布变换（平移、旋转、缩放）
//   - 裁剪效果（矩形裁剪、路径裁剪）
//   - 透明度与混叠
//   - 图像绘制
//   - 综合场景
//
// 运行方式（需要设置 CGO_ENABLED=1）:
//   $env:CGO_ENABLED='1'; go run ./examples/skiapaint/
//
// 输出: skiapaint_output.png

//go:build !ignore
// +build !ignore

package main

import (
	"fmt"
	"image"
	"image/color"
	"math"
	"os"

	"github.com/hoonfeng/goui/pkg/canvas"
	"github.com/hoonfeng/goui/pkg/paint"
	"github.com/hoonfeng/goui/pkg/types"
)

func main() {
	// ============================================================
	// 创建 SkiaCanvas
	// ============================================================
	width, height := 800, 1200
	sk := canvas.NewSkiaCanvas(width, height)
	defer sk.Release()

	// 填充白色背景
	bgPaint := paint.DefaultPaint()
	bgPaint.Color = types.ColorWhite
	sk.DrawRect(0, 0, float64(width), float64(height), bgPaint)

	// ============================================================
	// 测试组 1: 基本形状绘制
	// ============================================================
	drawSectionTitle(sk, "1. 基本形状绘制", 20)

	// 填充矩形
	fillPaint := paint.DefaultPaint()
	fillPaint.Color = types.Color{R: 66, G: 133, B: 244, A: 255} // 蓝色
	sk.DrawRect(30, 50, 100, 80, fillPaint)

	// 填充圆角矩形
	roundPaint := paint.DefaultPaint()
	roundPaint.Color = types.Color{R: 52, G: 168, B: 83, A: 255} // 绿色
	sk.DrawRoundedRect(160, 50, 100, 80, 16, roundPaint)

	// 填充圆形
	circlePaint := paint.DefaultPaint()
	circlePaint.Color = types.Color{R: 234, G: 67, B: 53, A: 255} // 红色
	sk.DrawCircle(330, 90, 45, circlePaint)

	// 填充椭圆（通过圆角矩形实现）
	ellipsePaint := paint.DefaultPaint()
	ellipsePaint.Color = types.Color{R: 142, G: 68, B: 173, A: 255} // 紫色
	sk.DrawRoundedRect(400, 50, 120, 80, 50, ellipsePaint)

	// 描边矩形
	strokePaint := paint.DefaultStrokePaint()
	strokePaint.Color = types.Color{R: 251, G: 188, B: 4, A: 255} // 黄色
	strokePaint.StrokeWidth = 4
	sk.DrawRect(550, 50, 100, 80, strokePaint)

	// 描边圆角矩形
	rStrokePaint := paint.DefaultStrokePaint()
	rStrokePaint.Color = types.Color{R: 0, G: 150, B: 136, A: 255} // 青色
	rStrokePaint.StrokeWidth = 4
	sk.DrawRoundedRect(680, 50, 100, 80, 16, rStrokePaint)

	// 描边圆形
	cStrokePaint := paint.DefaultStrokePaint()
	cStrokePaint.Color = types.Color{R: 233, G: 30, B: 99, A: 255} // 粉红
	cStrokePaint.StrokeWidth = 4
	sk.DrawCircle(730, 170, 35, cStrokePaint)

	// 直线
	linePaint := paint.DefaultStrokePaint()
	linePaint.Color = types.Color{R: 66, G: 133, B: 244, A: 255}
	linePaint.StrokeWidth = 3
	sk.DrawLine(30, 170, 200, 170, linePaint)
	sk.DrawLine(30, 170, 115, 140, linePaint)
	sk.DrawLine(115, 140, 200, 170, linePaint)

	// ============================================================
	// 测试组 2: 路径绘制
	// ============================================================
	drawSectionTitle(sk, "2. 路径绘制（贝塞尔曲线、多边形、星形）", 190)

	// 星形路径
	starPath := &canvas.Path{}
	cx, cy := 80.0, 260.0
	outerR, innerR := 40.0, 18.0
	for i := 0; i < 10; i++ {
		angle := float64(i)*math.Pi/5 - math.Pi/2
		r := outerR
		if i%2 == 1 {
			r = innerR
		}
		x := cx + r*math.Cos(angle)
		y := cy + r*math.Sin(angle)
		if i == 0 {
			starPath.MoveTo(x, y)
		} else {
			starPath.LineTo(x, y)
		}
	}
	starPath.Close()
	starFill := paint.DefaultPaint()
	starFill.Color = types.Color{R: 255, G: 193, B: 7, A: 255}
	sk.DrawPath(starPath, starFill)

	// 贝塞尔曲线路径
	bezierPath := &canvas.Path{}
	bezierPath.MoveTo(180, 230)
	bezierPath.CubicTo(220, 200, 260, 300, 300, 230)
	bezierPaint := paint.DefaultStrokePaint()
	bezierPaint.Color = types.Color{R: 233, G: 30, B: 99, A: 255}
	bezierPaint.StrokeWidth = 4
	sk.DrawPath(bezierPath, bezierPaint)

	// 二次贝塞尔曲线
	quadPath := &canvas.Path{}
	quadPath.MoveTo(330, 280)
	quadPath.QuadTo(380, 200, 430, 280)
	quadPaint := paint.DefaultStrokePaint()
	quadPaint.Color = types.Color{R: 0, G: 150, B: 136, A: 255}
	quadPaint.StrokeWidth = 3
	sk.DrawPath(quadPath, quadPaint)

	// 复合路径：心形
	heartPath := &canvas.Path{}
	heartPath.MoveTo(500, 270)
	heartPath.CubicTo(500, 240, 540, 220, 560, 250)
	heartPath.CubicTo(580, 220, 620, 240, 620, 270)
	heartPath.LineTo(560, 320)
	heartPath.Close()
	heartPaint := paint.DefaultPaint()
	heartPaint.Color = types.Color{R: 244, G: 67, B: 54, A: 240}
	sk.DrawPath(heartPath, heartPaint)

	// 带描边的填充路径
	fillStrokePath := &canvas.Path{}
	fillStrokePath.MoveTo(680, 260)
	fillStrokePath.LineTo(740, 220)
	fillStrokePath.LineTo(770, 300)
	fillStrokePath.Close()
	fillStrokePaint := paint.DefaultPaint()
	fillStrokePaint.Color = types.Color{R: 103, G: 58, B: 183, A: 180}
	fillStrokePaint.Style = paint.StyleFillAndStroke
	fillStrokePaint.StrokeWidth = 3
	sk.DrawPath(fillStrokePath, fillStrokePaint)

	// ============================================================
	// 测试组 3: 文本渲染
	// ============================================================
	drawSectionTitle(sk, "3. 文本渲染（多种字体、字号、颜色）", 340)

	// 标题文本
	titleFont := canvas.Font{
		Family: "sans-serif",
		Size:   28,
		Weight: canvas.FontWeightBold,
	}
	titlePaint := paint.DefaultPaint()
	titlePaint.Color = types.Color{R: 33, G: 33, B: 33, A: 255}
	sk.DrawText("SkiaCanvas 渲染引擎测试", 30, 380, titleFont, titlePaint)

	// 中等文本
	mediumFont := canvas.Font{
		Family: "sans-serif",
		Size:   20,
		Weight: canvas.FontWeightMedium,
	}
	mediumPaint := paint.DefaultPaint()
	mediumPaint.Color = types.Color{R: 66, G: 133, B: 244, A: 255}
	sk.DrawText("Skia 硬件加速 2D 渲染引擎", 30, 415, mediumFont, mediumPaint)

	// 普通文本
	normalFont := canvas.Font{
		Family: "sans-serif",
		Size:   16,
		Weight: canvas.FontWeightNormal,
	}
	normalPaint := paint.DefaultPaint()
	normalPaint.Color = types.Color{R: 80, G: 80, B: 80, A: 255}

	texts := []string{
		"Hello, Skia! 你好，Skia！こんにちは",
		"Antialiased 抗锯齿渲染 — 边缘平滑无锯齿",
		"Subpixel 亚像素定位 — 精确到 1/3 像素",
		"Unicode 支持: ⊂◉‿◉⊃ ヽ(•‿•)ノ ✨ 🎨",
	}
	for i, t := range texts {
		sk.DrawText(t, 30, 445+float64(i)*24, normalFont, normalPaint)
	}

	// 不同颜色文本
	colorfulFont := canvas.Font{
		Family: "sans-serif",
		Size:   15,
		Weight: canvas.FontWeightNormal,
	}
	colors := []types.Color{
		{R: 244, G: 67, B: 54, A: 255},   // 红
		{R: 255, G: 152, B: 0, A: 255},   // 橙
		{R: 255, G: 193, B: 7, A: 255},   // 黄
		{R: 52, G: 168, B: 83, A: 255},   // 绿
		{R: 66, G: 133, B: 244, A: 255},  // 蓝
		{R: 142, G: 68, B: 173, A: 255},  // 紫
		{R: 233, G: 30, B: 99, A: 255},   // 粉
	}
	for i, c := range colors {
		cp := paint.DefaultPaint()
		cp.Color = c
		sk.DrawText(fmt.Sprintf("彩色文本 Color #%d", i+1), 30+float64(i)*110, 560, colorfulFont, cp)
	}

	// ============================================================
	// 测试组 4: 渐变填充
	// ============================================================
	drawSectionTitle(sk, "4. 渐变填充（线性渐变 + 径向渐变）", 590)

	// 线性渐变矩形
	linearGradPaint := paint.DefaultPaint()
	linearGradPaint.LinearGradient = &paint.Gradient{
		Start: types.Point{X: 30, Y: 630},
		End:   types.Point{X: 230, Y: 730},
		Stops: []paint.ColorStop{
			{Offset: 0.0, Color: types.Color{R: 255, G: 0, B: 0, A: 255}},
			{Offset: 0.5, Color: types.Color{R: 0, G: 255, B: 0, A: 255}},
			{Offset: 1.0, Color: types.Color{R: 0, G: 0, B: 255, A: 255}},
		},
	}
	sk.DrawRect(30, 630, 200, 100, linearGradPaint)

	// 线性渐变圆角矩形
	linearGradPaint2 := paint.DefaultPaint()
	linearGradPaint2.LinearGradient = &paint.Gradient{
		Start: types.Point{X: 260, Y: 630},
		End:   types.Point{X: 460, Y: 630},
		Stops: []paint.ColorStop{
			{Offset: 0.0, Color: types.Color{R: 142, G: 68, B: 173, A: 255}},
			{Offset: 1.0, Color: types.Color{R: 255, G: 193, B: 7, A: 255}},
		},
	}
	sk.DrawRoundedRect(260, 630, 200, 100, 20, linearGradPaint2)

	// 径向渐变圆形
	radialGradPaint := paint.DefaultPaint()
	radialGradPaint.RadialGradient = &paint.Gradient{
		Center: types.Point{X: 570, Y: 680},
		Radius: 60,
		Stops: []paint.ColorStop{
			{Offset: 0.0, Color: types.Color{R: 255, G: 255, B: 255, A: 255}},
			{Offset: 0.3, Color: types.Color{R: 66, G: 133, B: 244, A: 255}},
			{Offset: 1.0, Color: types.Color{R: 33, G: 66, B: 150, A: 255}},
		},
	}
	sk.DrawCircle(570, 680, 60, radialGradPaint)

	// 多色径向渐变
	multiRadialPaint := paint.DefaultPaint()
	multiRadialPaint.RadialGradient = &paint.Gradient{
		Center: types.Point{X: 710, Y: 680},
		Radius: 60,
		Stops: []paint.ColorStop{
			{Offset: 0.0, Color: types.Color{R: 255, G: 255, B: 255, A: 255}},
			{Offset: 0.2, Color: types.Color{R: 255, G: 0, B: 0, A: 255}},
			{Offset: 0.5, Color: types.Color{R: 0, G: 255, B: 0, A: 255}},
			{Offset: 0.8, Color: types.Color{R: 0, G: 0, B: 255, A: 255}},
			{Offset: 1.0, Color: types.Color{R: 142, G: 68, B: 173, A: 255}},
		},
	}
	sk.DrawCircle(710, 680, 60, multiRadialPaint)

	// ============================================================
	// 测试组 5: 画布变换
	// ============================================================
	drawSectionTitle(sk, "5. 画布变换（平移、旋转、缩放）", 780)

	// 平移 + 旋转
	sk.Save()
	sk.Translate(100, 870)
	sk.Rotate(25 * math.Pi / 180)
	rotPaint := paint.DefaultPaint()
	rotPaint.Color = types.Color{R: 66, G: 133, B: 244, A: 200}
	sk.DrawRect(-40, -30, 80, 60, rotPaint)
	sk.DrawText("旋转文本", -30, 10, normalFont, normalPaint)
	sk.Restore()

	// 缩放
	sk.Save()
	sk.Translate(300, 830)
	sk.Scale(1.5, 1.5)
	scalePaint := paint.DefaultPaint()
	scalePaint.Color = types.Color{R: 52, G: 168, B: 83, A: 180}
	sk.DrawRoundedRect(-30, -20, 60, 40, 8, scalePaint)
	sk.DrawText("缩放", -20, 10, normalFont, normalPaint)
	sk.Restore()

	// 平移 + 缩放 + 旋转（复合变换）
	sk.Save()
	sk.Translate(530, 860)
	sk.Rotate(45 * math.Pi / 180)
	sk.Scale(0.8, 0.8)
	comboPaint := paint.DefaultPaint()
	comboPaint.Color = types.Color{R: 233, G: 30, B: 99, A: 180}
	sk.DrawRect(-50, -25, 100, 50, comboPaint)
	sk.DrawText("复合变换", -40, 8, normalFont, normalPaint)
	sk.Restore()

	// 平移 + 裁剪
	sk.Save()
	sk.Translate(680, 830)
	sk.ClipRect(0, 0, 100, 80)
	clipPaint := paint.DefaultPaint()
	clipPaint.Color = types.Color{R: 142, G: 68, B: 173, A: 200}
	sk.DrawCircle(50, 40, 50, clipPaint)
	sk.DrawText("裁剪区域", 10, 70, normalFont, normalPaint)
	sk.Restore()

	// ============================================================
	// 测试组 6: 透明度与抗锯齿
	// ============================================================
	drawSectionTitle(sk, "6. 透明度与混叠效果", 980)

	// 重叠半透明圆
	alphaColors := []types.Color{
		{R: 255, G: 0, B: 0, A: 100},
		{R: 0, G: 255, B: 0, A: 100},
		{R: 0, G: 0, B: 255, A: 100},
		{R: 255, G: 255, B: 0, A: 80},
	}
	positions := []types.Point{
		{X: 60, Y: 1030},
		{X: 110, Y: 1030},
		{X: 85, Y: 1060},
		{X: 85, Y: 1000},
	}
	for i, pos := range positions {
		ap := paint.DefaultPaint()
		ap.Color = alphaColors[i]
		sk.DrawCircle(pos.X, pos.Y, 35, ap)
	}

	// 渐变透明度条
	for i := 0; i < 15; i++ {
		alpha := uint8(float64(i) / 14.0 * 255)
		ap := paint.DefaultPaint()
		ap.Color = types.Color{R: 66, G: 133, B: 244, A: alpha}
		sk.DrawRect(200+float64(i)*35, 1000, 30, 70, ap)
	}

	// 描边抗锯齿对比
	sk.Save()
	sk.Translate(0, 0)
	// 粗描边圆
	thickStroke := paint.DefaultStrokePaint()
	thickStroke.Color = types.Color{R: 244, G: 67, B: 54, A: 200}
	thickStroke.StrokeWidth = 8
	sk.DrawCircle(620, 1020, 30, thickStroke)

	thickStroke.Color = types.Color{R: 52, G: 168, B: 83, A: 200}
	sk.DrawCircle(680, 1020, 30, thickStroke)

	thickStroke.Color = types.Color{R: 66, G: 133, B: 244, A: 200}
	sk.DrawCircle(740, 1020, 30, thickStroke)
	sk.Restore()

	// ============================================================
	// 测试组 7: 图像绘制
	// ============================================================
	drawSectionTitle(sk, "7. 图像绘制（自绘渐变图像）", 1080)

	// 创建一个测试图像
	testImg := image.NewRGBA(image.Rect(0, 0, 100, 80))
	// 绘制彩色渐变
	for y := 0; y < 80; y++ {
		for x := 0; x < 100; x++ {
			r := uint8(float64(x) / 100.0 * 255)
			g := uint8(float64(y) / 80.0 * 255)
			b := uint8(255 - float64(x)/100.0*255)
			testImg.Set(x, y, color.RGBA{R: r, G: g, B: b, A: 255})
		}
	}
	// 绘制一些形状到测试图像上
	for x := 20; x < 40; x++ {
		for y := 20; y < 40; y++ {
			testImg.Set(x, y, color.RGBA{R: 255, G: 255, B: 255, A: 255})
		}
	}
	sk.DrawImage(testImg, types.Rect{X: 30, Y: 1120, Width: 150, Height: 120})

	// 绘制缩小的图像
	sk.DrawImage(testImg, types.Rect{X: 200, Y: 1120, Width: 75, Height: 60})

	// 绘制放大的图像
	sk.DrawImage(testImg, types.Rect{X: 300, Y: 1120, Width: 200, Height: 160})

	// 绘制带半透明的图像
	sk.Save()
	sk.Translate(530, 1120)
	alphaPaint := paint.DefaultPaint()
	alphaPaint.Color = types.Color{R: 0, G: 0, B: 0, A: 80}
	// 直接绘制图像
	sk.DrawImage(testImg, types.Rect{X: 0, Y: 0, Width: 100, Height: 80})
	sk.Restore()

	// ============================================================
	// 输出结果
	// ============================================================
	if err := sk.SaveToPNG("skiapaint_output.png"); err != nil {
		fmt.Fprintf(os.Stderr, "保存 PNG 失败: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("✅ skiapaint_output.png 已保存 — SkiaCanvas 功能测试完成")

	// 输出统计信息
	fmt.Printf("   画布尺寸: %dx%d\n", width, height)
	fmt.Println("   测试组: 基本形状 | 路径 | 文本 | 渐变 | 变换 | 透明度 | 图像")
}

// drawSectionTitle 绘制分节标题
func drawSectionTitle(sk *canvas.SkiaCanvas, title string, y float64) {
	// 灰色背景条
	bg := paint.DefaultPaint()
	bg.Color = types.Color{R: 240, G: 240, B: 245, A: 255}
	sk.DrawRect(0, y-5, 800, 30, bg)

	// 标题文本
	f := canvas.Font{
		Family: "sans-serif",
		Size:   16,
		Weight: canvas.FontWeightBold,
	}
	p := paint.DefaultPaint()
	p.Color = types.Color{R: 50, G: 50, B: 50, A: 255}
	sk.DrawText(title, 15, y+18, f, p)
}
