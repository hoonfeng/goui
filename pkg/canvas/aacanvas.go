// Package canvas 提供了 AACanvas 反走样画布实现。
// AACanvas 采用 SVG 风格的像素级抗锯齿渲染（coverage-based anti-aliasing），
// 使用 4x4 超采样 (MSAA) 和距离场计算覆盖率，实现平滑的图形边缘。
package canvas

import (
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"math"
	"os"
	"sort"

	gofont "golang.org/x/image/font"
	"golang.org/x/image/math/fixed"

	"github.com/hoonfeng/goui/pkg/paint"
	"github.com/hoonfeng/goui/pkg/types"
)

// AACanvas 是基于 image.RGBA 的反走样 Canvas 2D 实现。
// 使用 SVG 风格的覆盖率抗锯齿（coverage-based anti-aliasing），
// 通过 4x4 超采样计算每个像素被图形覆盖的比例，实现平滑边缘。
type AACanvas struct {
	width, height float64
	img           *image.RGBA           // 帧缓冲
	clipRect      image.Rectangle       // 裁剪矩形
	clipEnabled   bool
	stateStack    []aaCanvasState       // 状态栈
	curState      aaCanvasState         // 当前状态
}

// aaCanvasState 保存画布状态
type aaCanvasState struct {
	clipRect    image.Rectangle
	clipEnabled bool
}

// NewAACanvas 创建新的反走样画布
func NewAACanvas(width, height int) *AACanvas {
	c := &AACanvas{
		width:  float64(width),
		height: float64(height),
		img:    image.NewRGBA(image.Rect(0, 0, width, height)),
		clipRect: image.Rect(0, 0, width, height),
		curState: aaCanvasState{
			clipRect:    image.Rect(0, 0, width, height),
			clipEnabled: false,
		},
	}
	return c
}

// Image 返回底层的 RGBA 图像
func (c *AACanvas) Image() *image.RGBA {
	return c.img
}

// --- 状态栈 ---

func (c *AACanvas) Save() {
	c.stateStack = append(c.stateStack, c.curState)
}

// SaveLayerAlpha 软件画布暂不支持组透明度合成，退化为普通 Save（alpha 被忽略）。
func (c *AACanvas) SaveLayerAlpha(alpha float64) { c.Save() }

func (c *AACanvas) Restore() {
	if len(c.stateStack) > 0 {
		c.curState = c.stateStack[len(c.stateStack)-1]
		c.stateStack = c.stateStack[:len(c.stateStack)-1]
		c.clipRect = c.curState.clipRect
		c.clipEnabled = c.curState.clipEnabled
	}
}

// --- 变换（软件简化版本：仅记录偏移，完整仿射变换可后续扩展） ---

func (c *AACanvas) Translate(x, y float64) {
	// 软件渲染器中变换通过绘制时偏移实现
}

func (c *AACanvas) Rotate(angle float64) {}

func (c *AACanvas) Scale(x, y float64) {}

// --- 裁剪 ---

func (c *AACanvas) Clip(path *Path) {
	if path == nil || len(path.Items) == 0 {
		c.clipEnabled = false
		return
	}
	minX, minY := math.MaxFloat64, math.MaxFloat64
	maxX, maxY := -math.MaxFloat64, -math.MaxFloat64
	for _, item := range path.Items {
		var effectiveCoords []float64
		switch item.Op {
		case PathOpClose:
			continue
		case PathOpMoveTo, PathOpLineTo:
			effectiveCoords = item.Points[:2]
		case PathOpQuadTo:
			effectiveCoords = item.Points[:4]
		case PathOpCubicTo, PathOpArc:
			effectiveCoords = item.Points[:6]
		}
		for i := 0; i < len(effectiveCoords); i += 2 {
			x, y := effectiveCoords[i], effectiveCoords[i+1]
			if x < minX {
				minX = x
			}
			if x > maxX {
				maxX = x
			}
			if y < minY {
				minY = y
			}
			if y > maxY {
				maxY = y
			}
		}
	}
	r := image.Rect(
		int(math.Floor(minX)), int(math.Floor(minY)),
		int(math.Ceil(maxX)), int(math.Ceil(maxY)),
	)
	if c.clipEnabled {
		r = r.Intersect(c.clipRect)
	}
	c.clipRect = r
	c.clipEnabled = true
	c.curState.clipRect = r
	c.curState.clipEnabled = true
}

func (c *AACanvas) ClipRect(x, y, w, h float64) {
	r := image.Rect(
		int(math.Floor(x)), int(math.Floor(y)),
		int(math.Ceil(x+w)), int(math.Ceil(y+h)),
	)
	if c.clipEnabled {
		r = r.Intersect(c.clipRect)
	}
	c.clipRect = r
	c.clipEnabled = true
	c.curState.clipRect = r
	c.curState.clipEnabled = true
}

// inClip 检查像素是否在裁剪区域内
func (c *AACanvas) inClip(x, y int) bool {
	if !c.clipEnabled {
		return true
	}
	return x >= c.clipRect.Min.X && x < c.clipRect.Max.X &&
		y >= c.clipRect.Min.Y && y < c.clipRect.Max.Y
}

// --- 颜色和 Paint ---

// toRGBA 将 paint.Paint 转为 color.RGBA，考虑透明度
func toAARGBA(p paint.Paint) color.RGBA {
	clr := p.Color
	a := uint8(float64(clr.A) * p.Opacity)
	return color.RGBA{R: clr.R, G: clr.G, B: clr.B, A: a}
}

// setPixel 在帧缓冲中设置单个像素（带裁剪和透明度混合）
func (c *AACanvas) setPixel(x, y int, rgba color.RGBA) {
	if x < 0 || x >= int(c.width) || y < 0 || y >= int(c.height) {
		return
	}
	if !c.inClip(x, y) {
		return
	}
	if rgba.A == 255 {
		off := c.img.PixOffset(x, y)
		c.img.Pix[off+0] = rgba.R
		c.img.Pix[off+1] = rgba.G
		c.img.Pix[off+2] = rgba.B
		c.img.Pix[off+3] = rgba.A
	} else if rgba.A > 0 {
		off := c.img.PixOffset(x, y)
		sr := uint32(rgba.R)
		sg := uint32(rgba.G)
		sb := uint32(rgba.B)
		sa := uint32(rgba.A)
		dr := uint32(c.img.Pix[off+0])
		dg := uint32(c.img.Pix[off+1])
		db := uint32(c.img.Pix[off+2])
		da := uint32(c.img.Pix[off+3])
		ia := 255 - sa
		outA := sa + da*ia/255
		if outA > 0 {
			c.img.Pix[off+0] = uint8((sr*sa + dr*da*ia/255) / outA)
			c.img.Pix[off+1] = uint8((sg*sa + dg*da*ia/255) / outA)
			c.img.Pix[off+2] = uint8((sb*sa + db*da*ia/255) / outA)
			c.img.Pix[off+3] = uint8(outA)
		}
	}
}

// setPixelAlpha 绘制带覆盖率 alpha 的像素
// coverage 范围 0.0~1.0，表示像素被图形覆盖的比例
func (c *AACanvas) setPixelAlpha(x, y int, rgba color.RGBA, coverage float64) {
	if coverage <= 0 || coverage > 1 {
		return
	}
	// 限制 coverage 到 [0,1]
	if coverage > 1 {
		coverage = 1
	}
	// 将覆盖率应用到 alpha
	aa := uint8(float64(rgba.A) * coverage)
	if aa == 0 {
		return
	}
	aaRGBA := color.RGBA{R: rgba.R, G: rgba.G, B: rgba.B, A: aa}
	c.setPixel(x, y, aaRGBA)
}

// ============================================================================
// SVG 风格反走样绘制核心函数
// ============================================================================

// 4x4 子采样偏移（每个子像素中心在像素内的归一化位置）
var aaSubPixelOffsets = [16][2]float64{
	{0.125, 0.125}, {0.375, 0.125}, {0.625, 0.125}, {0.875, 0.125},
	{0.125, 0.375}, {0.375, 0.375}, {0.625, 0.375}, {0.875, 0.375},
	{0.125, 0.625}, {0.375, 0.625}, {0.625, 0.625}, {0.875, 0.625},
	{0.125, 0.875}, {0.375, 0.875}, {0.625, 0.875}, {0.875, 0.875},
}

// coverageRectFill 计算矩形填充的覆盖率（使用 4x4 超采样）
func coverageRectFill(px, py int, x, y, w, h float64) float64 {
	inside := 0
	for _, off := range aaSubPixelOffsets {
		sx := float64(px) + off[0]
		sy := float64(py) + off[1]
		if sx >= x && sx < x+w && sy >= y && sy < y+h {
			inside++
		}
	}
	return float64(inside) / 16.0
}

// coverageRectStroke 计算矩形描边的覆盖率
func coverageRectStroke(px, py int, x, y, w, h, strokeWidth float64) float64 {
	halfW := strokeWidth / 2
	inside := 0
	for _, off := range aaSubPixelOffsets {
		sx := float64(px) + off[0]
		sy := float64(py) + off[1]
		// 点在矩形边框附近（外边框到内边框之间的区域）
		// 在外部扩展区域内
		if sx >= x-halfW && sx <= x+w+halfW && sy >= y-halfW && sy <= y+h+halfW {
			// 且不在内部未描边区域
			if !(sx >= x+halfW && sx <= x+w-halfW && sy >= y+halfW && sy <= y+h-halfW) {
				inside++
			}
		}
	}
	return float64(inside) / 16.0
}

// coverageCircleFill 计算圆形填充的覆盖率
func coverageCircleFill(px, py int, cx, cy, radius float64) float64 {
	r2 := radius * radius
	inside := 0
	for _, off := range aaSubPixelOffsets {
		sx := float64(px) + off[0] - cx
		sy := float64(py) + off[1] - cy
		if sx*sx+sy*sy <= r2 {
			inside++
		}
	}
	return float64(inside) / 16.0
}

// coverageCircleStroke 计算圆形描边的覆盖率
func coverageCircleStroke(px, py int, cx, cy, radius, strokeWidth float64) float64 {
	innerR := radius - strokeWidth/2
	outerR := radius + strokeWidth/2
	innerR2 := innerR * innerR
	outerR2 := outerR * outerR
	inside := 0
	for _, off := range aaSubPixelOffsets {
		sx := float64(px) + off[0] - cx
		sy := float64(py) + off[1] - cy
		d2 := sx*sx + sy*sy
		if d2 >= innerR2 && d2 <= outerR2 {
			inside++
		}
	}
	return float64(inside) / 16.0
}

// coverageLine 计算线条的覆盖率（点到线段距离）
func coverageLine(px, py int, x1, y1, x2, y2, strokeWidth float64) float64 {
	halfW := strokeWidth / 2
	inside := 0
	for _, off := range aaSubPixelOffsets {
		sx := float64(px) + off[0]
		sy := float64(py) + off[1]
		dist := pointToLineDist(sx, sy, x1, y1, x2, y2)
		if dist <= halfW {
			inside++
		}
	}
	return float64(inside) / 16.0
}

// pointToLineDist 计算点到线段的最短距离
func pointToLineDist(px, py, x1, y1, x2, y2 float64) float64 {
	dx := x2 - x1
	dy := y2 - y1
	if dx == 0 && dy == 0 {
		dx2 := px - x1
		dy2 := py - y1
		return math.Sqrt(dx2*dx2 + dy2*dy2)
	}
	t := ((px-x1)*dx + (py-y1)*dy) / (dx*dx + dy*dy)
	if t < 0 {
		t = 0
	} else if t > 1 {
		t = 1
	}
	cx := x1 + t*dx
	cy := y1 + t*dy
	dx2 := px - cx
	dy2 := py - cy
	return math.Sqrt(dx2*dx2 + dy2*dy2)
}

// coverageRoundedRectFill 计算圆角矩形填充的覆盖率
func coverageRoundedRectFill(px, py int, x, y, w, h, r float64) float64 {
	inside := 0
	for _, off := range aaSubPixelOffsets {
		sx := float64(px) + off[0]
		sy := float64(py) + off[1]
		if isInsideRoundedRect(sx, sy, x, y, w, h, r) {
			inside++
		}
	}
	return float64(inside) / 16.0
}

// isInsideRoundedRect 判断点是否在圆角矩形内部
func isInsideRoundedRect(sx, sy, x, y, w, h, r float64) bool {
	// 快速排除：在矩形外部
	if sx < x || sx > x+w || sy < y || sy > y+h {
		return false
	}
	// 检查四个圆角区域
	// 左上角
	if sx < x+r && sy < y+r {
		dx := sx - (x + r)
		dy := sy - (y + r)
		return dx*dx+dy*dy <= r*r
	}
	// 右上角
	if sx > x+w-r && sy < y+r {
		dx := sx - (x + w - r)
		dy := sy - (y + r)
		return dx*dx+dy*dy <= r*r
	}
	// 左下角
	if sx < x+r && sy > y+h-r {
		dx := sx - (x + r)
		dy := sy - (y + h - r)
		return dx*dx+dy*dy <= r*r
	}
	// 右下角
	if sx > x+w-r && sy > y+h-r {
		dx := sx - (x + w - r)
		dy := sy - (y + h - r)
		return dx*dx+dy*dy <= r*r
	}
	return true
}

// ============================================================================
// Canvas 接口实现
// ============================================================================

func (c *AACanvas) DrawPath(path *Path, p paint.Paint) {
	if path == nil || len(path.Items) == 0 {
		return
	}
	rgba := toAARGBA(p)
	vertices := c.flattenPath(path)
	if p.Style == paint.StyleFill || p.Style == paint.StyleFillAndStroke {
		c.fillPolygonAA(vertices, rgba)
	}
	if p.Style == paint.StyleStroke || p.Style == paint.StyleFillAndStroke {
		strokeW := p.StrokeWidth
		if strokeW <= 0 {
			strokeW = 1
		}
		for i := 0; i < len(vertices); i++ {
			next := (i + 1) % len(vertices)
			c.drawThickLineAA(vertices[i], vertices[next], rgba, strokeW)
		}
	}
}

func (c *AACanvas) DrawRect(x, y, w, h float64, p paint.Paint) {
	rgba := toAARGBA(p)
	if p.Style == paint.StyleFill || p.Style == paint.StyleFillAndStroke {
		c.fillRectAA(x, y, w, h, rgba)
	}
	if p.Style == paint.StyleStroke || p.Style == paint.StyleFillAndStroke {
		strokeW := p.StrokeWidth
		if strokeW <= 0 {
			strokeW = 1
		}
		c.strokeRectAA(x, y, w, h, rgba, strokeW)
	}
}

func (c *AACanvas) DrawRoundedRect(x, y, w, h, r float64, p paint.Paint) {
	if r <= 0 {
		c.DrawRect(x, y, w, h, p)
		return
	}
	halfMin := math.Min(w, h) / 2
	if r > halfMin {
		r = halfMin
	}
	rgba := toAARGBA(p)
	if p.Style == paint.StyleFill || p.Style == paint.StyleFillAndStroke {
		c.fillRoundedRectAA(x, y, w, h, r, rgba)
	}
	if p.Style == paint.StyleStroke || p.Style == paint.StyleFillAndStroke {
		strokeW := p.StrokeWidth
		if strokeW <= 0 {
			strokeW = 1
		}
		// 对于描边，使用路径方式绘制（多个线段的抗锯齿）
		path := &Path{}
		segments := 12
		path.MoveTo(x+r, y)
		path.LineTo(x+w-r, y)
		for i := 0; i <= segments; i++ {
			angle := -math.Pi/2 + float64(i)/float64(segments)*math.Pi/2
			vx := x + w - r + r*math.Cos(angle)
			vy := y + r + r*math.Sin(angle)
			path.LineTo(vx, vy)
		}
		path.LineTo(x+w, y+h-r)
		for i := 0; i <= segments; i++ {
			angle := 0.0 + float64(i)/float64(segments)*math.Pi/2
			vx := x + w - r + r*math.Cos(angle)
			vy := y + h - r + r*math.Sin(angle)
			path.LineTo(vx, vy)
		}
		path.LineTo(x+r, y+h)
		for i := 0; i <= segments; i++ {
			angle := math.Pi/2 + float64(i)/float64(segments)*math.Pi/2
			vx := x + r + r*math.Cos(angle)
			vy := y + h - r + r*math.Sin(angle)
			path.LineTo(vx, vy)
		}
		path.LineTo(x, y+r)
		for i := 0; i <= segments; i++ {
			angle := math.Pi + float64(i)/float64(segments)*math.Pi/2
			vx := x + r + r*math.Cos(angle)
			vy := y + r + r*math.Sin(angle)
			path.LineTo(vx, vy)
		}
		path.Close()
		vertices := c.flattenPath(path)
		for i := 0; i < len(vertices); i++ {
			next := (i + 1) % len(vertices)
			c.drawThickLineAA(vertices[i], vertices[next], rgba, strokeW)
		}
	}
}

func (c *AACanvas) DrawCircle(cx, cy, radius float64, p paint.Paint) {
	if radius <= 0 {
		return
	}
	rgba := toAARGBA(p)
	if p.Style == paint.StyleFill || p.Style == paint.StyleFillAndStroke {
		c.fillCircleAA(cx, cy, radius, rgba)
	}
	if p.Style == paint.StyleStroke || p.Style == paint.StyleFillAndStroke {
		strokeW := p.StrokeWidth
		if strokeW <= 0 {
			strokeW = 1
		}
		c.strokeCircleAA(cx, cy, radius, rgba, strokeW)
	}
}

func (c *AACanvas) DrawLine(x1, y1, x2, y2 float64, p paint.Paint) {
	rgba := toAARGBA(p)
	strokeW := p.StrokeWidth
	if strokeW <= 0 {
		strokeW = 1
	}
	start := types.Point{X: x1, Y: y1}
	end := types.Point{X: x2, Y: y2}
	c.drawThickLineAA(start, end, rgba, strokeW)
}

func (c *AACanvas) DrawText(text string, x, y float64, font Font, p paint.Paint) {
	if text == "" {
		return
	}
	fp := font.Size
	if fp <= 0 {
		fp = 16
	}
	face := c.getFace(fp, font.Weight)
	if face == nil {
		return
	}
	rgba := toAARGBA(p)

	advance := gofont.MeasureString(face, text)
	metrics := face.Metrics()
	txtWidth := float64(advance) / 64.0
	txtHeight := float64(metrics.Height) / 64.0
	ascent := float64(metrics.Ascent) / 64.0

	if txtWidth <= 0 || txtHeight <= 0 {
		txtWidth = float64(len(text)) * 7
		txtHeight = 13
		ascent = 10
	}

	imgW := int(math.Ceil(txtWidth)) + 4
	imgH := int(math.Ceil(txtHeight)) + 4
	if imgW <= 0 || imgH <= 0 {
		return
	}

	tmpImg := image.NewRGBA(image.Rect(0, 0, imgW, imgH))
	txtColor := color.RGBA{R: rgba.R, G: rgba.G, B: rgba.B, A: rgba.A}

	d := &gofont.Drawer{
		Dst:  tmpImg,
		Src:  image.NewUniform(txtColor),
		Face: face,
		Dot:  fixed.Point26_6{X: fixed.I(2), Y: fixed.I(2) + metrics.Ascent},
	}
	d.DrawString(text)

	drawY := int(math.Floor(y - ascent - 2))
	drawX := int(math.Floor(x - 2))

	for py := 0; py < imgH; py++ {
		for px := 0; px < imgW; px++ {
			srcColor := tmpImg.RGBAAt(px, py)
			if srcColor.A > 0 {
				c.setPixel(drawX+px, drawY+py, srcColor)
			}
		}
	}
}

func (c *AACanvas) DrawImage(img image.Image, dst types.Rect) {
	bounds := img.Bounds()
	if bounds.Empty() {
		return
	}
	srcRGBA := image.NewRGBA(bounds)
	draw.Draw(srcRGBA, bounds, img, bounds.Min, draw.Src)

	dstX := int(math.Floor(dst.X))
	dstY := int(math.Floor(dst.Y))
	dstW := int(math.Ceil(dst.Width))
	dstH := int(math.Ceil(dst.Height))

	if dstW <= 0 || dstH <= 0 {
		return
	}

	srcW := bounds.Dx()
	srcH := bounds.Dy()

	for dy := 0; dy < dstH; dy++ {
		for dx := 0; dx < dstW; dx++ {
			sx := dx * srcW / dstW
			sy := dy * srcH / dstH
			if sx < 0 {
				sx = 0
			}
			if sy < 0 {
				sy = 0
			}
			if sx >= srcW {
				sx = srcW - 1
			}
			if sy >= srcH {
				sy = srcH - 1
			}
			srcColor := srcRGBA.RGBAAt(sx, sy)
			c.setPixel(dstX+dx, dstY+dy, srcColor)
		}
	}
}

func (c *AACanvas) MeasureText(text string, font Font) TextMetrics {
	if text == "" {
		return TextMetrics{}
	}
	fp := font.Size
	if fp <= 0 {
		fp = 16
	}
	face := c.getFace(fp, font.Weight)
	if face == nil {
		return TextMetrics{
			Width:   float64(len(text)) * 7,
			Height:  13,
			Ascent:  10,
			Descent: 3,
			Advance: float64(len(text)) * 7,
		}
	}
	advance := gofont.MeasureString(face, text)
	metrics := face.Metrics()
	width := float64(advance) / 64.0
	height := float64(metrics.Height) / 64.0
	ascent := float64(metrics.Ascent) / 64.0
	descent := float64(metrics.Descent) / 64.0
	return TextMetrics{
		Width:   width,
		Height:  height,
		Ascent:  ascent,
		Descent: descent,
		Advance: width,
	}
}

func (c *AACanvas) Width() float64  { return c.width }
func (c *AACanvas) Height() float64 { return c.height }

func (c *AACanvas) Flush() error {
	return nil
}

// SaveToPNG 将当前帧缓冲保存为 PNG 文件（用于调试和视觉验证）
func (c *AACanvas) SaveToPNG(filename string) error {
	f, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer f.Close()
	return png.Encode(f, c.img)
}

// ============================================================================
// 反走样填充/描边方法
// ============================================================================

// fillRectAA 反走样填充矩形
func (c *AACanvas) fillRectAA(x, y, w, h float64, rgba color.RGBA) {
	x0 := clampIntAA(int(math.Floor(x-1)), 0, int(c.width)-1)
	y0 := clampIntAA(int(math.Floor(y-1)), 0, int(c.height)-1)
	x1 := clampIntAA(int(math.Ceil(x+w+1)), 0, int(c.width))
	y1 := clampIntAA(int(math.Ceil(y+h+1)), 0, int(c.height))

	for py := y0; py < y1; py++ {
		for px := x0; px < x1; px++ {
			cov := coverageRectFill(px, py, x, y, w, h)
			if cov > 0 {
				c.setPixelAlpha(px, py, rgba, cov)
			}
		}
	}
}

// strokeRectAA 反走样描边矩形
func (c *AACanvas) strokeRectAA(x, y, w, h float64, rgba color.RGBA, width float64) {
	if width <= 0 {
		width = 1
	}
	halfW := width / 2
	x0 := clampIntAA(int(math.Floor(x-halfW-1)), 0, int(c.width)-1)
	y0 := clampIntAA(int(math.Floor(y-halfW-1)), 0, int(c.height)-1)
	x1 := clampIntAA(int(math.Ceil(x+w+halfW+1)), 0, int(c.width))
	y1 := clampIntAA(int(math.Ceil(y+h+halfW+1)), 0, int(c.height))

	for py := y0; py < y1; py++ {
		for px := x0; px < x1; px++ {
			cov := coverageRectStroke(px, py, x, y, w, h, width)
			if cov > 0 {
				c.setPixelAlpha(px, py, rgba, cov)
			}
		}
	}
}

// fillCircleAA 反走样填充圆形
func (c *AACanvas) fillCircleAA(cx, cy, radius float64, rgba color.RGBA) {
	x0 := clampIntAA(int(math.Floor(cx-radius-1)), 0, int(c.width)-1)
	y0 := clampIntAA(int(math.Floor(cy-radius-1)), 0, int(c.height)-1)
	x1 := clampIntAA(int(math.Ceil(cx+radius+1)), 0, int(c.width))
	y1 := clampIntAA(int(math.Ceil(cy+radius+1)), 0, int(c.height))

	for py := y0; py < y1; py++ {
		for px := x0; px < x1; px++ {
			cov := coverageCircleFill(px, py, cx, cy, radius)
			if cov > 0 {
				c.setPixelAlpha(px, py, rgba, cov)
			}
		}
	}
}

// strokeCircleAA 反走样描边圆形
func (c *AACanvas) strokeCircleAA(cx, cy, radius float64, rgba color.RGBA, width float64) {
	if width <= 0 {
		width = 1
	}
	outerR := radius + width/2
	x0 := clampIntAA(int(math.Floor(cx-outerR-1)), 0, int(c.width)-1)
	y0 := clampIntAA(int(math.Floor(cy-outerR-1)), 0, int(c.height)-1)
	x1 := clampIntAA(int(math.Ceil(cx+outerR+1)), 0, int(c.width))
	y1 := clampIntAA(int(math.Ceil(cy+outerR+1)), 0, int(c.height))

	for py := y0; py < y1; py++ {
		for px := x0; px < x1; px++ {
			cov := coverageCircleStroke(px, py, cx, cy, radius, width)
			if cov > 0 {
				c.setPixelAlpha(px, py, rgba, cov)
			}
		}
	}
}

// fillRoundedRectAA 反走样填充圆角矩形
func (c *AACanvas) fillRoundedRectAA(x, y, w, h, r float64, rgba color.RGBA) {
	x0 := clampIntAA(int(math.Floor(x-1)), 0, int(c.width)-1)
	y0 := clampIntAA(int(math.Floor(y-1)), 0, int(c.height)-1)
	x1 := clampIntAA(int(math.Ceil(x+w+1)), 0, int(c.width))
	y1 := clampIntAA(int(math.Ceil(y+h+1)), 0, int(c.height))

	for py := y0; py < y1; py++ {
		for px := x0; px < x1; px++ {
			cov := coverageRoundedRectFill(px, py, x, y, w, h, r)
			if cov > 0 {
				c.setPixelAlpha(px, py, rgba, cov)
			}
		}
	}
}

// drawThickLineAA 反走样绘制粗线
func (c *AACanvas) drawThickLineAA(start, end types.Point, rgba color.RGBA, width float64) {
	if width <= 0 {
		width = 1
	}
	halfW := width / 2
	minX := math.Min(start.X, end.X) - halfW - 1
	maxX := math.Max(start.X, end.X) + halfW + 1
	minY := math.Min(start.Y, end.Y) - halfW - 1
	maxY := math.Max(start.Y, end.Y) + halfW + 1

	x0 := clampIntAA(int(math.Floor(minX)), 0, int(c.width)-1)
	y0 := clampIntAA(int(math.Floor(minY)), 0, int(c.height)-1)
	x1 := clampIntAA(int(math.Ceil(maxX)), 0, int(c.width))
	y1 := clampIntAA(int(math.Ceil(maxY)), 0, int(c.height))

	for py := y0; py < y1; py++ {
		for px := x0; px < x1; px++ {
			cov := coverageLine(px, py, start.X, start.Y, end.X, end.Y, width)
			if cov > 0 {
				c.setPixelAlpha(px, py, rgba, cov)
			}
		}
	}
}

// fillPolygonAA 反走样填充多边形（扫描线 + 边缘抗锯齿）
func (c *AACanvas) fillPolygonAA(vertices []types.Point, rgba color.RGBA) {
	if len(vertices) < 3 {
		return
	}

	// 计算包围盒
	minY := math.MaxFloat64
	maxY := -math.MaxFloat64
	for _, v := range vertices {
		if v.Y < minY {
			minY = v.Y
		}
		if v.Y > maxY {
			maxY = v.Y
		}
	}

	y0 := int(math.Floor(minY))
	y1 := int(math.Ceil(maxY))
	if y0 < 0 {
		y0 = 0
	}
	if y1 > int(c.height) {
		y1 = int(c.height)
	}

	// 对每一行，计算与多边形的交点
	for y := y0; y < y1; y++ {
		var intersections []float64
		scanY := float64(y) + 0.5

		for i := 0; i < len(vertices); i++ {
			j := (i + 1) % len(vertices)
			p1, p2 := vertices[i], vertices[j]

			if (p1.Y < scanY && p2.Y >= scanY) || (p2.Y < scanY && p1.Y >= scanY) {
				t := (scanY - p1.Y) / (p2.Y - p1.Y)
				x := p1.X + t*(p2.X-p1.X)
				intersections = append(intersections, x)
			}
		}

		sort.Float64s(intersections)

		// 配对填充（内部像素用纯色，边缘用抗锯齿）
		for i := 0; i+1 < len(intersections); i += 2 {
			xStart := intersections[i]
			xEnd := intersections[i+1]
			startX := int(math.Ceil(xStart))
			endX := int(math.Floor(xEnd))

			// 左边缘抗锯齿
			if startX > int(math.Floor(xStart)) {
				cov := (float64(startX) - xStart)
				c.setPixelAlpha(startX-1, y, rgba, cov)
			}
			// 右边缘抗锯齿
			if endX < int(math.Ceil(xEnd)) {
				cov := (xEnd - float64(endX))
				c.setPixelAlpha(endX+1, y, rgba, cov)
			}
			// 内部像素
			for x := startX; x <= endX; x++ {
				if x >= 0 && x < int(c.width) {
					c.setPixel(x, y, rgba)
				}
			}
		}
	}
}

// ============================================================================
// 内部辅助方法
// ============================================================================

// flattenPath 将路径展平为顶点列表
func (c *AACanvas) flattenPath(path *Path) []types.Point {
	if len(path.Items) == 0 {
		return nil
	}
	var vertices []types.Point
	var current, start types.Point

	for _, item := range path.Items {
		switch item.Op {
		case PathOpMoveTo:
			current = types.Point{X: item.Points[0], Y: item.Points[1]}
			start = current
			vertices = append(vertices, current)

		case PathOpLineTo:
			current = types.Point{X: item.Points[0], Y: item.Points[1]}
			vertices = append(vertices, current)

		case PathOpQuadTo:
			cx, cy := item.Points[0], item.Points[1]
			ex, ey := item.Points[2], item.Points[3]
			points := quadraticBezierAA(current, types.Point{X: cx, Y: cy}, types.Point{X: ex, Y: ey}, 16)
			vertices = append(vertices, points[1:]...)
			current = types.Point{X: ex, Y: ey}

		case PathOpCubicTo:
			cx1, cy1 := item.Points[0], item.Points[1]
			cx2, cy2 := item.Points[2], item.Points[3]
			ex, ey := item.Points[4], item.Points[5]
			points := cubicBezierAA(current, types.Point{X: cx1, Y: cy1}, types.Point{X: cx2, Y: cy2}, types.Point{X: ex, Y: ey}, 16)
			vertices = append(vertices, points[1:]...)
			current = types.Point{X: ex, Y: ey}

		case PathOpArc:
			cx, cy := item.Points[0], item.Points[1]
			radius := item.Points[2]
			startAngle := item.Points[3]
			endAngle := item.Points[4]
			clockwise := item.Points[5] > 0
			points := arcVerticesAA(types.Point{X: cx, Y: cy}, radius, startAngle, endAngle, clockwise, 24)
			vertices = append(vertices, points...)
			if len(points) > 0 {
				current = points[len(points)-1]
			}

		case PathOpClose:
			vertices = append(vertices, start)
			current = start
		}
	}
	return vertices
}

// getFace 获取指定尺寸和字重的字体 face
func (c *AACanvas) getFace(size float64, weight FontWeight) gofont.Face {
	return GetFontFace(size, weight)
}

// ============================================================================
// 辅助函数
// ============================================================================

func clampIntAA(v, min, max int) int {
	if v < min {
		return min
	}
	if v > max {
		return max
	}
	return v
}

func quadraticBezierAA(p0, p1, p2 types.Point, segments int) []types.Point {
	points := make([]types.Point, segments+1)
	for i := 0; i <= segments; i++ {
		t := float64(i) / float64(segments)
		mt := 1 - t
		points[i] = types.Point{
			X: mt*mt*p0.X + 2*mt*t*p1.X + t*t*p2.X,
			Y: mt*mt*p0.Y + 2*mt*t*p1.Y + t*t*p2.Y,
		}
	}
	return points
}

func cubicBezierAA(p0, p1, p2, p3 types.Point, segments int) []types.Point {
	points := make([]types.Point, segments+1)
	for i := 0; i <= segments; i++ {
		t := float64(i) / float64(segments)
		mt := 1 - t
		points[i] = types.Point{
			X: mt*mt*mt*p0.X + 3*mt*mt*t*p1.X + 3*mt*t*t*p2.X + t*t*t*p3.X,
			Y: mt*mt*mt*p0.Y + 3*mt*mt*t*p1.Y + 3*mt*t*t*p2.Y + t*t*t*p3.Y,
		}
	}
	return points
}

func arcVerticesAA(center types.Point, radius, startAngle, endAngle float64, clockwise bool, segments int) []types.Point {
	if clockwise {
		for startAngle < endAngle {
			startAngle += 2 * math.Pi
		}
	} else {
		for endAngle < startAngle {
			endAngle += 2 * math.Pi
		}
	}
	count := segments
	if count < 4 {
		count = 4
	}
	points := make([]types.Point, count+1)
	angleRange := endAngle - startAngle
	if !clockwise && angleRange < 0 {
		angleRange += 2 * math.Pi
	} else if clockwise && angleRange > 0 {
		angleRange -= 2 * math.Pi
	}
	for i := 0; i <= count; i++ {
		t := float64(i) / float64(count)
		angle := startAngle + t*angleRange
		points[i] = types.Point{
			X: center.X + radius*math.Cos(angle),
			Y: center.Y + radius*math.Sin(angle),
		}
	}
	return points
}
