// Package visual 提供组件视觉验证工具（Visual Validation Suite）。
//
// 本包通过 SoftCanvas 的软件渲染能力，对组件树执行"无头渲染"
// （headless rendering），捕获渲染结果并逐像素分析，以判断组件
// 是否被正确绘制、位置是否准确、颜色是否符合预期。
//
// 典型使用流程：
//
//	ctx := visual.NewVisualTestContext(800, 600)
//	frame, err := ctx.Render(rootWidget)
//
//	// 方式1：使用 PixelAnalyzer 直接分析
//	pa := frame.Analyze()
//	if pa.IsBlank() { ... }
//	c := pa.ColorAt(100, 100)
//
//	// 方式2：使用 VisualSuite 注册多个检查项
//	suite := visual.NewVisualSuite()
//	suite.AddCheck("非空白", visual.CheckNonBlank())
//	report := suite.Run(frame)
package visual

import (
	"fmt"
	"image"
	"image/color"
	"log"

	"github.com/user/goui/internal/canvas"
	"github.com/user/goui/internal/render"
	"github.com/user/goui/internal/widget"
)

// ─────────────────────────────────────────────────────────────
// 核心类型
// ─────────────────────────────────────────────────────────────

// VisualTestContext 管理视觉测试的渲染和捕获流程。
// 它创建一个离屏 SoftCanvas + Pipeline，将组件树渲染到内存
// 中的 RGBA 图像，供后续像素分析。
type VisualTestContext struct {
	Width    int
	Height   int
	Canvas   *canvas.SoftCanvas
	Pipeline *render.Pipeline
}

// NewVisualTestContext 创建新的视觉测试上下文。
// width 和 height 指定渲染画布的尺寸（像素）。
func NewVisualTestContext(width, height int) *VisualTestContext {
	softCanvas := canvas.NewSoftCanvas(width, height)
	pipeline := render.NewPipeline(width, height, softCanvas)

	return &VisualTestContext{
		Width:    width,
		Height:   height,
		Canvas:   softCanvas,
		Pipeline: pipeline,
	}
}

// Render 将指定的 Widget 树渲染到画布，并返回捕获的帧数据。
// 流程：
//  1. 使用 widget.CreateElementFor 创建根 Element
//  2. 挂载并构建 Element 树
//  3. 通过 Pipeline 执行布局和绘制
//  4. 从 SoftCanvas 获取像素数据
func (ctx *VisualTestContext) Render(root widget.Widget) (*VisualFrame, error) {
	if root == nil {
		return nil, fmt.Errorf("visual: Render called with nil root Widget")
	}

	// 1. 创建 Element
	rootEl := widget.CreateElementFor(root)
	if rootEl == nil {
		return nil, fmt.Errorf("visual: CreateElementFor returned nil")
	}

	// 2. 挂载
	rootEl.Mount(nil, 0)

	// 3. 设置到管线
	ctx.Pipeline.SetRootElement(rootEl)

	// 4. 执行渲染（含布局 + 绘制）
	if err := ctx.Pipeline.Render(); err != nil {
		return nil, fmt.Errorf("visual: Pipeline.Render failed: %w", err)
	}

	// 5. 从 SoftCanvas 获取像素数据
	img := ctx.Canvas.Image()
	if img == nil {
		return nil, fmt.Errorf("visual: SoftCanvas.Image() returned nil")
	}

	// 6. 复制像素数据（防止后续修改影响）
	bounds := img.Bounds()
	clone := image.NewRGBA(bounds)
	copy(clone.Pix, img.Pix)

	log.Printf("[visual] Rendered frame: %dx%d, total pixels=%d",
		clone.Bounds().Dx(), clone.Bounds().Dy(), len(clone.Pix)/4)

	return &VisualFrame{
		Image:  clone,
		Width:  ctx.Width,
		Height: ctx.Height,
	}, nil
}

// RenderElement 直接将已有的 Element 树渲染到画布并返回帧数据。
// 适用于已经构建好 Element 树的场景，避免重复创建。
func (ctx *VisualTestContext) RenderElement(root widget.Element) (*VisualFrame, error) {
	if root == nil {
		return nil, fmt.Errorf("visual: RenderElement called with nil root Element")
	}

	ctx.Pipeline.SetRootElement(root)

	if err := ctx.Pipeline.Render(); err != nil {
		return nil, fmt.Errorf("visual: Pipeline.Render failed: %w", err)
	}

	img := ctx.Canvas.Image()
	if img == nil {
		return nil, fmt.Errorf("visual: SoftCanvas.Image() returned nil")
	}

	bounds := img.Bounds()
	clone := image.NewRGBA(bounds)
	copy(clone.Pix, img.Pix)

	return &VisualFrame{
		Image:  clone,
		Width:  ctx.Width,
		Height: ctx.Height,
	}, nil
}

// Reset 重置画布为空白状态（全白），清除所有绘制内容。
func (ctx *VisualTestContext) Reset() {
	ctx.Canvas = canvas.NewSoftCanvas(ctx.Width, ctx.Height)
	ctx.Pipeline = render.NewPipeline(ctx.Width, ctx.Height, ctx.Canvas)
}

// ─────────────────────────────────────────────────────────────
// VisualFrame — 单帧渲染结果的像素数据
// ─────────────────────────────────────────────────────────────

// VisualFrame 封装一次渲染结果的全部像素数据。
// 它包含完整的 RGBA 图像，是像素分析的输入。
type VisualFrame struct {
	Image  *image.RGBA
	Width  int
	Height int
}

// Analyze 返回当前帧的像素分析器。
func (vf *VisualFrame) Analyze() *PixelAnalyzer {
	return &PixelAnalyzer{frame: vf}
}

// PixelAt 返回指定坐标的颜色值。
// 若坐标越界，返回全透明黑色（Color{0,0,0,0}）。
func (vf *VisualFrame) PixelAt(x, y int) color.RGBA {
	if x < 0 || x >= vf.Width || y < 0 || y >= vf.Height {
		return color.RGBA{0, 0, 0, 0}
	}
	return vf.Image.RGBAAt(x, y)
}

// SubImage 返回指定矩形区域的子图。
func (vf *VisualFrame) SubImage(rect image.Rectangle) *image.RGBA {
	rect = rect.Intersect(vf.Image.Bounds())
	if rect.Empty() {
		return image.NewRGBA(image.Rect(0, 0, 0, 0))
	}
	return vf.Image.SubImage(rect).(*image.RGBA)
}

// ─────────────────────────────────────────────────────────────
// PixelAnalyzer — 像素分析器
// ─────────────────────────────────────────────────────────────

// PixelAnalyzer 提供对 VisualFrame 的逐像素分析能力。
// 所有分析基于帧的 RGBA 像素数据，不依赖 widget 树信息。
type PixelAnalyzer struct {
	frame *VisualFrame
}

// NewPixelAnalyzer 从指定帧创建分析器。
func NewPixelAnalyzer(frame *VisualFrame) *PixelAnalyzer {
	return &PixelAnalyzer{frame: frame}
}

// ColorAt 返回画布上 (x, y) 坐标的颜色。
// 坐标原点为画布左上角 (0,0)，x 向右，y 向下。
func (pa *PixelAnalyzer) ColorAt(x, y int) color.RGBA {
	return pa.frame.PixelAt(x, y)
}

// IsBlank 检查画布是否"空白"——即没有任何非白色像素。
// 空白判断标准：所有像素的 R=255, G=255, B=255, A=255。
// 这是验证组件是否真正被渲染的最基础检查。
func (pa *PixelAnalyzer) IsBlank() bool {
	return pa.CountNonWhitePixels(pa.frame.Image.Bounds()) == 0
}

// HasNonWhiteContent 检查画布上是否有任何非白色绘制内容。
// 等价于 !IsBlank()。
func (pa *PixelAnalyzer) HasNonWhiteContent() bool {
	return !pa.IsBlank()
}

// CountColor 统计画布上指定颜色（在容差范围内）的像素数量。
// target 是目标颜色，tolerance 是每个通道的容差（0-255）。
// tolerance=0 表示精确匹配。
func (pa *PixelAnalyzer) CountColor(target color.RGBA, tolerance uint8) int {
	return pa.CountColorInRect(target, tolerance, pa.frame.Image.Bounds())
}

// CountColorInRect 统计指定矩形区域中目标颜色的像素数量。
func (pa *PixelAnalyzer) CountColorInRect(target color.RGBA, tolerance uint8, rect image.Rectangle) int {
	rect = rect.Intersect(pa.frame.Image.Bounds())
	if rect.Empty() {
		return 0
	}

	count := 0
	for y := rect.Min.Y; y < rect.Max.Y; y++ {
		for x := rect.Min.X; x < rect.Max.X; x++ {
			c := pa.frame.Image.RGBAAt(x, y)
			if colorSimilar(c, target, tolerance) {
				count++
			}
		}
	}
	return count
}

// CountNonWhitePixels 统计指定矩形区域内的非白色像素数量。
// 非白色指 R≠255 或 G≠255 或 B≠255 或 A≠255 的像素。
func (pa *PixelAnalyzer) CountNonWhitePixels(rect image.Rectangle) int {
	white := color.RGBA{R: 255, G: 255, B: 255, A: 255}
	whiteCount := pa.CountColorInRect(white, 0, rect)
	totalPixels := rect.Dx() * rect.Dy()
	return totalPixels - whiteCount
}

// HasColorInRect 检查指定矩形区域是否包含目标颜色（至少1个像素）。
func (pa *PixelAnalyzer) HasColorInRect(target color.RGBA, tolerance uint8, rect image.Rectangle) bool {
	return pa.CountColorInRect(target, tolerance, rect) > 0
}

// FindColorRect 在画布中搜索包含目标颜色的最小矩形区域。
// 返回第一个包含目标颜色的像素区域的边界矩形。
// 如果未找到，返回 image.ZR。
func (pa *PixelAnalyzer) FindColorRect(target color.RGBA, tolerance uint8) image.Rectangle {
	bounds := pa.frame.Image.Bounds()
	minX, minY := bounds.Max.X, bounds.Max.Y
	maxX, maxY := bounds.Min.X, bounds.Min.Y
	found := false

	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			c := pa.frame.Image.RGBAAt(x, y)
			if colorSimilar(c, target, tolerance) {
				if x < minX {
					minX = x
				}
				if y < minY {
					minY = y
				}
				if x > maxX {
					maxX = x
				}
				if y > maxY {
					maxY = y
				}
				found = true
			}
		}
	}

	if !found {
		return image.Rectangle{}
	}

	return image.Rect(minX, minY, maxX+1, maxY+1)
}

// FindColorClusters 查找画布中所有指定颜色的连续区域（连通分量）。
// minArea 指定区域的最小像素面积，小于此值的区域将被忽略。
// 返回每个连通区域的边界矩形列表。
func (pa *PixelAnalyzer) FindColorClusters(target color.RGBA, tolerance uint8, minArea int) []image.Rectangle {
	bounds := pa.frame.Image.Bounds()
	visited := make(map[image.Point]bool)
	var clusters []image.Rectangle

	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			pt := image.Point{X: x, Y: y}
			if visited[pt] {
				continue
			}

			c := pa.frame.Image.RGBAAt(x, y)
			if !colorSimilar(c, target, tolerance) {
				continue
			}

			// BFS 搜索连通区域
			area := 0
			minCX, minCY := x, y
			maxCX, maxCY := x, y

			queue := []image.Point{pt}
			visited[pt] = true

			for len(queue) > 0 {
				p := queue[0]
				queue = queue[1:]
				area++

				if p.X < minCX {
					minCX = p.X
				}
				if p.Y < minCY {
					minCY = p.Y
				}
				if p.X > maxCX {
					maxCX = p.X
				}
				if p.Y > maxCY {
					maxCY = p.Y
				}

				// 检查四邻域
				neighbors := []image.Point{
					{X: p.X - 1, Y: p.Y},
					{X: p.X + 1, Y: p.Y},
					{X: p.X, Y: p.Y - 1},
					{X: p.X, Y: p.Y + 1},
				}
				for _, n := range neighbors {
					if n.X < bounds.Min.X || n.X >= bounds.Max.X ||
						n.Y < bounds.Min.Y || n.Y >= bounds.Max.Y {
						continue
					}
					if visited[n] {
						continue
					}
					nc := pa.frame.Image.RGBAAt(n.X, n.Y)
					if colorSimilar(nc, target, tolerance) {
						visited[n] = true
						queue = append(queue, n)
					}
				}
			}

			if area >= minArea {
				clusters = append(clusters, image.Rect(minCX, minCY, maxCX+1, maxCY+1))
			}
		}
	}

	return clusters
}

// HorizontalLineColor 检查指定水平扫描线上是否只包含目标颜色。
// 用于快速检测行方向的颜色一致性（如分割线、背景填充）。
func (pa *PixelAnalyzer) HorizontalLineColor(y int, target color.RGBA, tolerance uint8) bool {
	if y < 0 || y >= pa.frame.Height {
		return false
	}
	for x := 0; x < pa.frame.Width; x++ {
		c := pa.frame.Image.RGBAAt(x, y)
		if !colorSimilar(c, target, tolerance) {
			return false
		}
	}
	return true
}

// VerticalLineColor 检查指定垂直扫描线上是否只包含目标颜色。
func (pa *PixelAnalyzer) VerticalLineColor(x int, target color.RGBA, tolerance uint8) bool {
	if x < 0 || x >= pa.frame.Width {
		return false
	}
	for y := 0; y < pa.frame.Height; y++ {
		c := pa.frame.Image.RGBAAt(x, y)
		if !colorSimilar(c, target, tolerance) {
			return false
		}
	}
	return true
}

// ─────────────────────────────────────────────────────────────
// 内部工具函数
// ─────────────────────────────────────────────────────────────

// colorSimilar 判断两个颜色是否在容差范围内相似。
func colorSimilar(a, b color.RGBA, tolerance uint8) bool {
	diffR := absDiffUint8(a.R, b.R)
	diffG := absDiffUint8(a.G, b.G)
	diffB := absDiffUint8(a.B, b.B)
	diffA := absDiffUint8(a.A, b.A)
	return diffR <= tolerance && diffG <= tolerance && diffB <= tolerance && diffA <= tolerance
}

func absDiffUint8(a, b uint8) uint8 {
	if a > b {
		return a - b
	}
	return b - a
}
