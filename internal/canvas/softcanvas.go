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

	"github.com/user/goui/internal/paint"
	"github.com/user/goui/internal/types"
)

// SoftCanvas 是基于 image.RGBA 的纯软件 Canvas 2D 实现。
// 不依赖 OpenGL，纯 Go 实现，可在任何平台上运行。
type SoftCanvas struct {
	width, height float64
	img           *image.RGBA           // 帧缓冲
	clipRect      image.Rectangle       // 裁剪矩形
	clipEnabled   bool
	stateStack    []softCanvasState     // 状态栈
	curState      softCanvasState       // 当前状态
}

// softCanvasState 保存画布状态
type softCanvasState struct {
	clipRect    image.Rectangle
	clipEnabled bool
}

// NewSoftCanvas 创建新的软件渲染画布
func NewSoftCanvas(width, height int) *SoftCanvas {
	c := &SoftCanvas{
		width:  float64(width),
		height: float64(height),
		img:    image.NewRGBA(image.Rect(0, 0, width, height)),
		clipRect: image.Rect(0, 0, width, height),
		curState: softCanvasState{
			clipRect:    image.Rect(0, 0, width, height),
			clipEnabled: false,
		},
	}
	return c
}

// Image 返回底层的 RGBA 图像
func (c *SoftCanvas) Image() *image.RGBA {
	return c.img
}

// --- 状态栈 ---

func (c *SoftCanvas) Save() {
	c.stateStack = append(c.stateStack, c.curState)
}

// SaveLayerAlpha 软件画布暂不支持组透明度合成，退化为普通 Save（alpha 被忽略）。
func (c *SoftCanvas) SaveLayerAlpha(alpha float64) { c.Save() }

func (c *SoftCanvas) Restore() {
	if len(c.stateStack) > 0 {
		c.curState = c.stateStack[len(c.stateStack)-1]
		c.stateStack = c.stateStack[:len(c.stateStack)-1]
		c.clipRect = c.curState.clipRect
		c.clipEnabled = c.curState.clipEnabled
	}
}

// --- 变换（软件简化版本：仅记录偏移，完整仿射变换可后续扩展） ---

func (c *SoftCanvas) Translate(x, y float64) {
	// 软件渲染器中变换通过绘制时偏移实现
	// 对于完整矩阵变换，需要复杂的像素级变换，暂不实现
}

func (c *SoftCanvas) Rotate(angle float64) {
	// TODO: 软件仿射变换
}

func (c *SoftCanvas) Scale(x, y float64) {
	// TODO: 软件仿射变换
}

// --- 裁剪 ---

func (c *SoftCanvas) Clip(path *Path) {
	// 简化：使用路径的包围盒作为裁剪矩形
	if path == nil || len(path.Items) == 0 {
		c.clipEnabled = false
		return
	}

	minX, minY := math.MaxFloat64, math.MaxFloat64
	maxX, maxY := -math.MaxFloat64, -math.MaxFloat64
	for _, item := range path.Items {
		// 根据操作类型确定有效的坐标范围
		var effectiveCoords []float64
		switch item.Op {
		case PathOpClose:
			continue // Close 操作不含有效坐标，跳过
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

func (c *SoftCanvas) ClipRect(x, y, w, h float64) {
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
func (c *SoftCanvas) inClip(x, y int) bool {
	if !c.clipEnabled {
		return true
	}
	return x >= c.clipRect.Min.X && x < c.clipRect.Max.X &&
		y >= c.clipRect.Min.Y && y < c.clipRect.Max.Y
}

// --- 颜色和 Paint ---

// toRGBA 将 types.Color 转为 color.RGBA，考虑透明度
func toSoftRGBA(p paint.Paint) color.RGBA {
	clr := p.Color
	a := uint8(float64(clr.A) * p.Opacity)
	return color.RGBA{R: clr.R, G: clr.G, B: clr.B, A: a}
}

// setPixel 在帧缓冲中设置单个像素（带裁剪和透明度混合）
func (c *SoftCanvas) setPixel(x, y int, rgba color.RGBA) {
	if x < 0 || x >= int(c.width) || y < 0 || y >= int(c.height) {
		return
	}
	if !c.inClip(x, y) {
		return
	}

	// 透明度混合
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

		// 标准 alpha 混合: result = src * sa/255 + dst * (1 - sa/255)
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

// --- 基本图形绘制 ---

// drawLineLow x 为主方向的直线绘制（|slope| <= 1）
func (c *SoftCanvas) drawLineLow(x0, y0, x1, y1 int, rgba color.RGBA) {
	dx := x1 - x0
	dy := y1 - y0
	yi := 1
	if dy < 0 {
		yi = -1
		dy = -dy
	}
	D := 2*dy - dx
	y := y0
	for x := x0; x <= x1; x++ {
		c.setPixel(x, y, rgba)
		if D > 0 {
			y += yi
			D -= 2 * dx
		}
		D += 2 * dy
	}
}

// drawLineHigh y 为主方向的直线绘制（|slope| > 1）
func (c *SoftCanvas) drawLineHigh(x0, y0, x1, y1 int, rgba color.RGBA) {
	dx := x1 - x0
	dy := y1 - y0
	xi := 1
	if dx < 0 {
		xi = -1
		dx = -dx
	}
	D := 2*dx - dy
	x := x0
	for y := y0; y <= y1; y++ {
		c.setPixel(x, y, rgba)
		if D > 0 {
			x += xi
			D -= 2 * dy
		}
		D += 2 * dx
	}
}

// bresenhamLine Bresenham 直线算法
func (c *SoftCanvas) bresenhamLine(x0, y0, x1, y1 int, rgba color.RGBA) {
	if x0 == x1 && y0 == y1 {
		c.setPixel(x0, y0, rgba)
		return
	}
	if math.Abs(float64(y1-y0)) <= math.Abs(float64(x1-x0)) {
		if x0 > x1 {
			c.drawLineLow(x1, y1, x0, y0, rgba)
		} else {
			c.drawLineLow(x0, y0, x1, y1, rgba)
		}
	} else {
		if y0 > y1 {
			c.drawLineHigh(x1, y1, x0, y0, rgba)
		} else {
			c.drawLineHigh(x0, y0, x1, y1, rgba)
		}
	}
}

// fillRect 填充矩形区域
func (c *SoftCanvas) fillRect(x, y, w, h float64, rgba color.RGBA) {
	x0 := clampInt(int(math.Floor(x)), 0, int(c.width)-1)
	y0 := clampInt(int(math.Floor(y)), 0, int(c.height)-1)
	x1 := clampInt(int(math.Ceil(x+w)), 0, int(c.width))
	y1 := clampInt(int(math.Ceil(y+h)), 0, int(c.height))

	for py := y0; py < y1; py++ {
		for px := x0; px < x1; px++ {
			c.setPixel(px, py, rgba)
		}
	}
}

// strokeRect 绘制矩形边框
func (c *SoftCanvas) strokeRect(x, y, w, h float64, rgba color.RGBA, width float64) {
	if width <= 0 {
		width = 1
	}
	// 上边
	c.fillRect(x, y, w, width, rgba)
	// 下边
	c.fillRect(x, y+h-width, w, width, rgba)
	// 左边
	c.fillRect(x, y, width, h, rgba)
	// 右边
	c.fillRect(x+w-width, y, width, h, rgba)
}

// --- Canvas 接口实现 ---

func (c *SoftCanvas) DrawPath(path *Path, p paint.Paint) {
	if path == nil || len(path.Items) == 0 {
		return
	}

	rgba := toSoftRGBA(p)
	vertices := c.flattenPath(path)

	if p.Style == paint.StyleFill || p.Style == paint.StyleFillAndStroke {
		c.fillPolygon(vertices, rgba)
	}

	if p.Style == paint.StyleStroke || p.Style == paint.StyleFillAndStroke {
		strokeW := p.StrokeWidth
		if strokeW <= 0 {
			strokeW = 1
		}
		for i := 0; i < len(vertices); i++ {
			next := (i + 1) % len(vertices)
			c.strokeThickLine(vertices[i], vertices[next], rgba, strokeW)
		}
	}
}

func (c *SoftCanvas) DrawRect(x, y, w, h float64, p paint.Paint) {
	rgba := toSoftRGBA(p)
	if p.Style == paint.StyleFill || p.Style == paint.StyleFillAndStroke {
		c.fillRect(x, y, w, h, rgba)
	}
	if p.Style == paint.StyleStroke || p.Style == paint.StyleFillAndStroke {
		strokeW := p.StrokeWidth
		if strokeW <= 0 {
			strokeW = 1
		}
		c.strokeRect(x, y, w, h, rgba, strokeW)
	}
}

func (c *SoftCanvas) DrawRoundedRect(x, y, w, h, r float64, p paint.Paint) {
	if r <= 0 {
		c.DrawRect(x, y, w, h, p)
		return
	}
	halfMin := math.Min(w, h) / 2
	if r > halfMin {
		r = halfMin
	}

	rgba := toSoftRGBA(p)

	// 构建圆角矩形路径（顺时针路径，屏幕 y 向下）
	// 路径顺序：顶部直线段 → 右上弧 → 右侧直线 → 右下弧 → 底部直线 → 左下弧 → 左侧直线 → 左上弧 → 闭合
	path := &Path{}
	segments := 12

	// 1. 从左上角圆弧顶部开始
	path.MoveTo(x+r, y)

	// 2. 顶部直线段到右上角弧起始点
	path.LineTo(x+w-r, y)

	// 3. 右上角弧: 从角度 -π/2 到 0（中心: x+w-r, y+r）
	for i := 0; i <= segments; i++ {
		angle := -math.Pi/2 + float64(i)/float64(segments)*math.Pi/2
		vx := x + w - r + r*math.Cos(angle)
		vy := y + r + r*math.Sin(angle)
		path.LineTo(vx, vy)
	}

	// 4. 右侧直线段到右下角弧起始点
	path.LineTo(x+w, y+h-r)

	// 5. 右下角弧: 从角度 0 到 π/2（中心: x+w-r, y+h-r）
	for i := 0; i <= segments; i++ {
		angle := 0.0 + float64(i)/float64(segments)*math.Pi/2
		vx := x + w - r + r*math.Cos(angle)
		vy := y + h - r + r*math.Sin(angle)
		path.LineTo(vx, vy)
	}

	// 6. 底部直线段到左下角弧起始点
	path.LineTo(x+r, y+h)

	// 7. 左下角弧: 从角度 π/2 到 π（中心: x+r, y+h-r）
	for i := 0; i <= segments; i++ {
		angle := math.Pi/2 + float64(i)/float64(segments)*math.Pi/2
		vx := x + r + r*math.Cos(angle)
		vy := y + h - r + r*math.Sin(angle)
		path.LineTo(vx, vy)
	}

	// 8. 左侧直线段到左上角弧起始点
	path.LineTo(x, y+r)

	// 9. 左上角弧: 从角度 π 到 3π/2（中心: x+r, y+r）
	for i := 0; i <= segments; i++ {
		angle := math.Pi + float64(i)/float64(segments)*math.Pi/2
		vx := x + r + r*math.Cos(angle)
		vy := y + r + r*math.Sin(angle)
		path.LineTo(vx, vy)
	}
	path.Close()

	vertices := c.flattenPath(path)
	if p.Style == paint.StyleFill || p.Style == paint.StyleFillAndStroke {
		c.fillPolygon(vertices, rgba)
	}
	if p.Style == paint.StyleStroke || p.Style == paint.StyleFillAndStroke {
		strokeW := p.StrokeWidth
		if strokeW <= 0 {
			strokeW = 1
		}
		for i := 0; i < len(vertices); i++ {
			next := (i + 1) % len(vertices)
			c.strokeThickLine(vertices[i], vertices[next], rgba, strokeW)
		}
	}
}

func (c *SoftCanvas) DrawCircle(cx, cy, radius float64, p paint.Paint) {
	if radius <= 0 {
		return
	}
	rgba := toSoftRGBA(p)

	// 使用圆的包围盒矩形区域，逐像素判断
	x0 := int(math.Floor(cx - radius))
	y0 := int(math.Floor(cy - radius))
	x1 := int(math.Ceil(cx + radius))
	y1 := int(math.Ceil(cy + radius))

	clampIntRange(&x0, &y0, &x1, &y1, int(c.width), int(c.height))

	r2 := radius * radius

	if p.Style == paint.StyleFill || p.Style == paint.StyleFillAndStroke {
		// 填充：对每一行，计算左右边界
		for py := y0; py <= y1; py++ {
			dy := float64(py) + 0.5 - cy
			if math.Abs(dy) > radius {
				continue
			}
			dx := math.Sqrt(r2 - dy*dy)
			leftX := int(math.Floor(cx - dx + 0.5))
			rightX := int(math.Floor(cx + dx + 0.5))
			for px := leftX; px <= rightX; px++ {
				c.setPixel(px, py, rgba)
			}
		}
	}

	if p.Style == paint.StyleStroke || p.Style == paint.StyleFillAndStroke {
		// 描边：使用中点圆算法
		strokeW := p.StrokeWidth
		if strokeW <= 0 {
			strokeW = 1
		}
		c.strokeCircle(cx, cy, radius, rgba, strokeW)
	}
}

func (c *SoftCanvas) DrawLine(x1, y1, x2, y2 float64, p paint.Paint) {
	rgba := toSoftRGBA(p)
	strokeW := p.StrokeWidth
	if strokeW <= 0 {
		strokeW = 1
	}
	start := types.Point{X: x1, Y: y1}
	end := types.Point{X: x2, Y: y2}
	c.strokeThickLine(start, end, rgba, strokeW)
}

func (c *SoftCanvas) DrawText(text string, x, y float64, font Font, p paint.Paint) {
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

	rgba := toSoftRGBA(p)

	// 使用 gofont.Drawer 绘制文本到临时图像
	advance := gofont.MeasureString(face, text)
	txtWidth := float64(advance) / 64.0
	metrics := face.Metrics()
	txtHeight := float64(metrics.Height) / 64.0
	ascent := float64(metrics.Ascent) / 64.0

	if txtWidth <= 0 || txtHeight <= 0 {
		// 回退
		txtWidth = float64(len(text)) * 7
		txtHeight = 13
		ascent = 10
	}

	imgW := int(math.Ceil(txtWidth)) + 4
	imgH := int(math.Ceil(txtHeight)) + 4
	if imgW <= 0 || imgH <= 0 {
		return
	}

	// 创建临时 RGBA 图像
	tmpImg := image.NewRGBA(image.Rect(0, 0, imgW, imgH))
	txtColor := color.RGBA{R: rgba.R, G: rgba.G, B: rgba.B, A: rgba.A}

	d := &gofont.Drawer{
		Dst:  tmpImg,
		Src:  image.NewUniform(txtColor),
		Face: face,
		Dot:  fixed.Point26_6{X: fixed.I(2), Y: fixed.I(2) + metrics.Ascent},
	}
	d.DrawString(text)

	// 将临时图像绘制到帧缓冲
	// 图像中文本绘制起始于 (2, 2+ascent)，需要减去 2px padding 使基线对齐到 y
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

func (c *SoftCanvas) DrawImage(img image.Image, dst types.Rect) {
	bounds := img.Bounds()
	if bounds.Empty() {
		return
	}

	// 转换为 RGBA
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

	// 逐个像素绘制（带简单的缩放）
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

func (c *SoftCanvas) MeasureText(text string, font Font) TextMetrics {
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
			Width: float64(len(text)) * 7,
			Height: 13,
			Ascent: 10,
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

func (c *SoftCanvas) Width() float64  { return c.width }
func (c *SoftCanvas) Height() float64 { return c.height }

func (c *SoftCanvas) Flush() error {
	return nil
}

// SaveToPNG 将当前帧缓冲保存为 PNG 文件（用于调试和视觉验证）
func (c *SoftCanvas) SaveToPNG(filename string) error {
	f, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer f.Close()
	return png.Encode(f, c.img)
}

// --- 内部辅助方法 ---

// flattenPath 将路径展平为顶点列表
func (c *SoftCanvas) flattenPath(path *Path) []types.Point {
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
			points := quadraticBezierSoft(current, types.Point{X: cx, Y: cy}, types.Point{X: ex, Y: ey}, 16)
			vertices = append(vertices, points[1:]...)
			current = types.Point{X: ex, Y: ey}

		case PathOpCubicTo:
			cx1, cy1 := item.Points[0], item.Points[1]
			cx2, cy2 := item.Points[2], item.Points[3]
			ex, ey := item.Points[4], item.Points[5]
			points := cubicBezierSoft(current, types.Point{X: cx1, Y: cy1}, types.Point{X: cx2, Y: cy2}, types.Point{X: ex, Y: ey}, 16)
			vertices = append(vertices, points[1:]...)
			current = types.Point{X: ex, Y: ey}

		case PathOpArc:
			cx, cy := item.Points[0], item.Points[1]
			radius := item.Points[2]
			startAngle := item.Points[3]
			endAngle := item.Points[4]
			clockwise := item.Points[5] > 0
			points := arcVerticesSoft(types.Point{X: cx, Y: cy}, radius, startAngle, endAngle, clockwise, 24)
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

// fillPolygon 使用扫描线算法填充多边形
func (c *SoftCanvas) fillPolygon(vertices []types.Point, rgba color.RGBA) {
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

			// 检查扫描线是否与边相交
			if (p1.Y < scanY && p2.Y >= scanY) || (p2.Y < scanY && p1.Y >= scanY) {
				// 计算交点 x 坐标
				t := (scanY - p1.Y) / (p2.Y - p1.Y)
				x := p1.X + t*(p2.X-p1.X)
				intersections = append(intersections, x)
			}
		}

		// 排序交点
		sort.Float64s(intersections)

		// 配对填充
		for i := 0; i+1 < len(intersections); i += 2 {
			x0 := int(math.Floor(intersections[i]))
			x1 := int(math.Ceil(intersections[i+1]))
			if x0 < 0 {
				x0 = 0
			}
			if x1 > int(c.width) {
				x1 = int(c.width)
			}
			for x := x0; x < x1; x++ {
				c.setPixel(x, y, rgba)
			}
		}
	}
}

// strokeThickLine 绘制具有指定宽度的直线
func (c *SoftCanvas) strokeThickLine(start, end types.Point, rgba color.RGBA, width float64) {
	if width <= 0 {
		width = 1
	}

	if width <= 1.5 {
		// 细线用 Bresenham
		c.bresenhamLine(int(math.Floor(start.X)), int(math.Floor(start.Y)),
			int(math.Floor(end.X)), int(math.Floor(end.Y)), rgba)
		return
	}

	// 粗线用矩形表示
	dx := end.X - start.X
	dy := end.Y - start.Y
	length := math.Sqrt(dx*dx + dy*dy)
	if length == 0 {
		return
	}

	// 计算垂直方向的单位向量
	nx := -dy / length * width / 2
	ny := dx / length * width / 2

	// 构建矩形四个顶点（沿线的垂直方向偏移）
	v1 := types.Point{X: start.X + nx, Y: start.Y + ny}
	v2 := types.Point{X: start.X - nx, Y: start.Y - ny}
	v3 := types.Point{X: end.X - nx, Y: end.Y - ny}
	v4 := types.Point{X: end.X + nx, Y: end.Y + ny}

	c.fillPolygon([]types.Point{v1, v2, v3, v4}, rgba)
}

// strokeCircle 绘制圆形描边
func (c *SoftCanvas) strokeCircle(cx, cy, radius float64, rgba color.RGBA, width float64) {
	if width <= 0 {
		width = 1
	}

	// 使用 Bresenham 圆算法生成圆上的点，然后绘制粗线
	segments := int(radius * math.Pi * 2)
	if segments < 16 {
		segments = 16
	}
	if segments > 256 {
		segments = 256
	}

	prev := types.Point{
		X: cx + radius,
		Y: cy,
	}
	for i := 1; i <= segments; i++ {
		angle := float64(i) / float64(segments) * 2 * math.Pi
		curr := types.Point{
			X: cx + radius*math.Cos(angle),
			Y: cy + radius*math.Sin(angle),
		}
		c.strokeThickLine(prev, curr, rgba, width)
		prev = curr
	}
}

// --- 字体管理（共享全局字体缓存） ---

func (c *SoftCanvas) getFace(size float64, weight FontWeight) gofont.Face {
	return GetFontFace(size, weight)
}

// --- 辅助函数 ---

func quadraticBezierSoft(p0, p1, p2 types.Point, segments int) []types.Point {
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

func cubicBezierSoft(p0, p1, p2, p3 types.Point, segments int) []types.Point {
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

func arcVerticesSoft(center types.Point, radius, startAngle, endAngle float64, clockwise bool, segments int) []types.Point {
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

func clampInt(v, min, max int) int {
	if v < min {
		return min
	}
	if v > max {
		return max
	}
	return v
}

func clampIntRange(x0, y0, x1, y1 *int, maxW, maxH int) {
	if *x0 < 0 {
		*x0 = 0
	}
	if *y0 < 0 {
		*y0 = 0
	}
	if *x1 > maxW {
		*x1 = maxW
	}
	if *y1 > maxH {
		*y1 = maxH
	}
}
