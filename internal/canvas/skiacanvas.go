// Package canvas 提供了 2D 绘制接口。
// SkiaCanvas 是基于 Skia 硬件加速的 Canvas 实现，使用 goskia CGO 绑定。
package canvas

import (
	"bytes"
	"fmt"
	"image"
	"image/draw"
	"image/png"
	"math"
	"os"
	"path/filepath"
	"strconv"
	"sync"

	goskia "github.com/hoonfeng/goskia/skia"
	"github.com/user/goui/internal/paint"
	"github.com/user/goui/internal/types"
)

// SkiaCanvas 是基于 Skia 硬件加速的 Canvas 2D 实现。
// 使用 goskia (github.com/hoonfeng/goskia) 作为 Skia 的 CGO 绑定，
// 将绘制操作转换为 Skia 调用，提供高质量抗锯齿渲染、硬件加速和跨平台一致性。
// 全局共享字体存储——Skia Typeface/Font 与 Canvas 实例无关、线程安全可跨 Canvas 共享，
// 避免每个 SkiaCanvas（主窗口 / 1×1 测量画布 / 副窗口）各加载一份字库（每份十几 MB）。
// 进程级常驻，不随单个 Canvas.Release 释放。
var (
	fontStoreMu     sync.Mutex
	sharedTypefaces = map[string]*goskia.Typeface{}
	sharedFonts     = map[string]*goskia.Font{}
)

type SkiaCanvas struct {
	width, height int
	surf          *goskia.Surface // Skia 渲染表面
	canvas        *goskia.Canvas  // Skia 画布（从 Surface 获取）
	img           *image.RGBA     // 缓存的 RGBA 图像（raster 模式供 PNG / 纹理上传）
	dirty         bool            // 是否需要刷新到 img

	// GPU 模式：Skia 经 OpenGL 直接渲染到窗口 framebuffer（FBO），
	// 省去「位图回读 → image.RGBA → 纹理上传」与每帧 PNG 编解码。
	gpu     bool
	glIface *goskia.GLInterface
	gpuCtx  *goskia.DirectContext
	fbo     uint32

	// 字体缓存 key -> *goskia.Font
	typefaceCache map[string]*goskia.Typeface
	fontCache     map[string]*goskia.Font
	mu            sync.Mutex

	initOnce sync.Once // 确保 skia.Init() 只调用一次
}

// NewSkiaCanvas 创建新的 Skia 硬件加速画布。
// width, height 为画布尺寸（像素）。
func NewSkiaCanvas(width, height int) *SkiaCanvas {
	c := &SkiaCanvas{
		width:         width,
		height:        height,
		dirty:         true,
		typefaceCache: sharedTypefaces, // 全局共享，避免每实例各加载一份字库
		fontCache:     sharedFonts,
	}

	// 初始化 Skia 全局状态（线程安全，只执行一次）
	c.initOnce.Do(func() {
		goskia.Init()
		goskia.SetFontCacheLimit(16 << 20) // 32 MiB 字形缓存上限（进程级）
	})

	// 创建 Raster Surface（CPU 渲染，N32 Premul = RGBA8888 格式）
	surf, err := goskia.NewRasterSurfaceN32Premul(width, height)
	if err != nil {
		panic("skia: failed to create raster surface: " + err.Error())
	}
	c.surf = surf
	c.canvas = surf.Canvas()

	// 填充白色背景
	c.canvas.Clear(goskia.ColorWhite)

	// 初始化缓存 RGBA
	c.img = image.NewRGBA(image.Rect(0, 0, width, height))

	return c
}

// NewSkiaCanvasGPU 创建 GPU 模式画布：Skia 经 OpenGL 直接渲染到窗口的默认 framebuffer（fboID，
// 通常为 0），彻底省去「Skia 位图 → image.RGBA → GL 纹理上传」的回读路径与每帧 PNG 编解码。
// 必须在目标窗口的 GL 上下文已 MakeCurrent 之后调用。
func NewSkiaCanvasGPU(width, height int, fboID uint32) (*SkiaCanvas, error) {
	c := &SkiaCanvas{
		width:         width,
		height:        height,
		dirty:         true,
		gpu:           true,
		fbo:           fboID,
		typefaceCache: sharedTypefaces, // 全局共享，避免每实例各加载一份字库
		fontCache:     sharedFonts,
	}
	c.initOnce.Do(func() {
		goskia.Init()
		goskia.SetFontCacheLimit(16 << 20)
	})

	iface, err := goskia.NewNativeGLInterface()
	if err != nil {
		return nil, err
	}
	ctx, err := goskia.NewGLContext(iface)
	if err != nil {
		iface.Release()
		return nil, err
	}
	c.glIface = iface
	c.gpuCtx = ctx
	// 约束 GPU 资源缓存（glyph atlas / 纹理 / 缓冲）上限，避免长时间运行后 GPU 端内存无界增长。
	ctx.SetResourceCacheLimit(48 << 20) // 48 MiB
	if err := c.makeGPUSurface(); err != nil {
		ctx.Release()
		iface.Release()
		return nil, err
	}
	return c, nil
}

// makeGPUSurface 把窗口 FBO 包装为 Skia GPU surface（在构造 / Resize 内调用，调用方持锁或单线程）。
func (c *SkiaCanvas) makeGPUSurface() error {
	// stencilBits=8 与窗口像素格式一致（供 Skia 裁剪）；BottomLeft 匹配 GL 窗口 framebuffer 原点。
	surf, err := goskia.NewGPUSurfaceFromFBO(c.gpuCtx, c.fbo, c.width, c.height,
		0, 8, goskia.GLRGBA8, goskia.ColorTypeRGBA8888, goskia.SurfaceOriginBottomLeft)
	if err != nil {
		return err
	}
	c.surf = surf
	c.canvas = surf.Canvas()
	c.canvas.Clear(goskia.ColorWhite)
	return nil
}

// Release 释放 Skia 资源。
func (c *SkiaCanvas) Release() {
	c.mu.Lock()
	defer c.mu.Unlock()

	// 字体（typeface/font）现为全局共享的进程级缓存，不随单个 Canvas 释放
	//（否则一个 Canvas 关闭会让其它 Canvas 持有的同一字体失效）。
	c.fontCache = nil
	c.typefaceCache = nil

	if c.surf != nil {
		c.surf.Release()
		c.surf = nil
		c.canvas = nil
	}
	if c.gpuCtx != nil {
		c.gpuCtx.Release()
		c.gpuCtx = nil
	}
	if c.glIface != nil {
		c.glIface.Release()
		c.glIface = nil
	}
}

// Image 返回底层的 RGBA 图像（供 OpenGL 纹理上传使用）。
func (c *SkiaCanvas) Image() *image.RGBA {
	return c.img
}

// Clear 用指定颜色替换整块画布的所有像素（kSrc）。alpha=0（如 types.Color{}）即清为**透明**，
// 用于离屏渲染透明背景的图（如把 SVG 渲到带圆角/透明边的图标）。默认构造时画布是白底。
func (c *SkiaCanvas) Clear(col types.Color) {
	c.canvas.Clear(toSkiaColor(col))
	c.dirty = true
}

// ============================================================================
// 状态栈
// ============================================================================

// Save 保存当前画布状态（矩阵和裁剪栈）。
func (c *SkiaCanvas) Save() {
	c.canvas.Save()
}

// SaveLayerAlpha 开启带整体透明度的离屏图层，Restore 时以 alpha 合成回画布。
func (c *SkiaCanvas) SaveLayerAlpha(alpha float64) {
	if alpha < 0 {
		alpha = 0
	} else if alpha > 1 {
		alpha = 1
	}
	p := goskia.NewPaint()
	p.SetColor(goskia.RGBA(255, 255, 255, uint8(alpha*255)))
	c.canvas.SaveLayer(nil, p)
	p.Release()
	c.dirty = true
}

// Restore 恢复上次保存的画布状态。
func (c *SkiaCanvas) Restore() {
	c.canvas.Restore()
}

// ============================================================================
// 变换
// ============================================================================

func (c *SkiaCanvas) Translate(x, y float64) {
	c.canvas.Translate(float32(x), float32(y))
	c.dirty = true
}

func (c *SkiaCanvas) Rotate(angle float64) {
	c.canvas.Rotate(float32(angle))
	c.dirty = true
}

func (c *SkiaCanvas) Scale(x, y float64) {
	c.canvas.Scale(float32(x), float32(y))
	c.dirty = true
}

// ============================================================================
// 裁剪
// ============================================================================

func (c *SkiaCanvas) Clip(path *Path) {
	sp := c.toSkiaPath(path)
	if sp == nil {
		return
	}
	defer sp.Release()
	c.canvas.ClipPath(sp, goskia.ClipOpIntersect, true)
	c.dirty = true
}

func (c *SkiaCanvas) ClipRect(x, y, w, h float64) {
	r := goskia.RectXYWH(float32(x), float32(y), float32(w), float32(h))
	c.canvas.ClipRect(r, goskia.ClipOpIntersect, true)
	c.dirty = true
}

// ============================================================================
// 颜色转换工具
// ============================================================================

// toSkiaColor 将 types.Color 转为 skia.Color（都是 0xAARRGGBB 格式）。
func toSkiaColor(clr types.Color) goskia.Color {
	return goskia.RGBA(clr.R, clr.G, clr.B, clr.A)
}

// toSkiaPaint 将 paint.Paint 转为 skia.Paint（需要调用者 Release）。
func toSkiaPaint(p paint.Paint) *goskia.Paint {
	sp := goskia.NewPaint()
	sp.SetAntialias(true)

	// 颜色：应用透明度
	clr := p.Color
	a := uint8(float64(clr.A) * p.Opacity)
	sp.SetColor(goskia.RGBA(clr.R, clr.G, clr.B, a))

	// 样式
	switch p.Style {
	case paint.StyleFill:
		sp.SetStyle(goskia.PaintStyleFill)
	case paint.StyleStroke:
		sp.SetStyle(goskia.PaintStyleStroke)
	case paint.StyleFillAndStroke:
		sp.SetStyle(goskia.PaintStyleStrokeAndFill)
	default:
		sp.SetStyle(goskia.PaintStyleFill)
	}

	// 描边宽度
	if p.StrokeWidth > 0 {
		sp.SetStrokeWidth(float32(p.StrokeWidth))
	} else {
		sp.SetStrokeWidth(1.0)
	}

	// 线性渐变
	if p.LinearGradient != nil {
		g := p.LinearGradient
		colors, positions := gradientToSkia(g.Stops)
		if shader := goskia.NewLinearGradient(
			goskia.Point{X: float32(g.Start.X), Y: float32(g.Start.Y)},
			goskia.Point{X: float32(g.End.X), Y: float32(g.End.Y)},
			colors, positions, goskia.TileModeClamp,
		); shader != nil {
			sp.SetShader(shader)
			shader.Release() // Paint 持有内部引用
		}
	}

	// 径向渐变
	if p.RadialGradient != nil {
		g := p.RadialGradient
		colors, positions := gradientToSkia(g.Stops)
		if shader := goskia.NewRadialGradient(
			goskia.Point{X: float32(g.Center.X), Y: float32(g.Center.Y)},
			float32(g.Radius), colors, positions, goskia.TileModeClamp,
		); shader != nil {
			sp.SetShader(shader)
			shader.Release()
		}
	}

	return sp
}

// gradientToSkia 将 paint.ColorStop 列表转为 skia 的颜色和位置数组。
func gradientToSkia(stops []paint.ColorStop) ([]goskia.Color, []float32) {
	n := len(stops)
	if n == 0 {
		return nil, nil
	}
	colors := make([]goskia.Color, n)
	positions := make([]float32, n)
	for i, stop := range stops {
		colors[i] = goskia.RGBA(stop.Color.R, stop.Color.G, stop.Color.B, stop.Color.A)
		positions[i] = float32(stop.Offset)
	}
	return colors, positions
}

// ============================================================================
// 路径转换
// ============================================================================

// toSkiaPath 将 canvas.Path 转换为 skia.Path（需要调用者 Release）。
func (c *SkiaCanvas) toSkiaPath(path *Path) *goskia.Path {
	if path == nil || len(path.Items) == 0 {
		return nil
	}

	sp := goskia.NewPath()
	for _, item := range path.Items {
		switch item.Op {
		case PathOpMoveTo:
			sp.MoveTo(float32(item.Points[0]), float32(item.Points[1]))
		case PathOpLineTo:
			sp.LineTo(float32(item.Points[0]), float32(item.Points[1]))
		case PathOpQuadTo:
			sp.QuadTo(
				float32(item.Points[0]), float32(item.Points[1]),
				float32(item.Points[2]), float32(item.Points[3]),
			)
		case PathOpCubicTo:
			sp.CubicTo(
				float32(item.Points[0]), float32(item.Points[1]),
				float32(item.Points[2]), float32(item.Points[3]),
				float32(item.Points[4]), float32(item.Points[5]),
			)
		case PathOpArc:
			cx, cy := float32(item.Points[0]), float32(item.Points[1])
			radius := float32(item.Points[2])
			startAngle := float32(item.Points[3])
			endAngle := float32(item.Points[4])
			clockwise := item.Points[5] > 0
			// 用线段近似圆弧
			segments := 24
			var angleRange float32
			if clockwise {
				angleRange = endAngle - startAngle
				for angleRange < 0 {
					angleRange += 2 * math.Pi
				}
			} else {
				angleRange = endAngle - startAngle
				for angleRange > 0 {
					angleRange -= 2 * math.Pi
				}
			}
			for i := 0; i <= segments; i++ {
				t := float32(i) / float32(segments)
				angle := startAngle + t*angleRange
				px := cx + radius*float32(math.Cos(float64(angle)))
				py := cy + radius*float32(math.Sin(float64(angle)))
				sp.LineTo(px, py)
			}
		case PathOpClose:
			sp.Close()
		}
	}
	return sp
}

// ============================================================================
// 绘制操作
// ============================================================================

func (c *SkiaCanvas) DrawPath(path *Path, p paint.Paint) {
	if path == nil || len(path.Items) == 0 {
		return
	}
	sp := c.toSkiaPath(path)
	if sp == nil {
		return
	}
	defer sp.Release()

	skPaint := toSkiaPaint(p)
	defer skPaint.Release()

	c.canvas.DrawPath(sp, skPaint)
	c.dirty = true
}

func (c *SkiaCanvas) DrawRect(x, y, w, h float64, p paint.Paint) {
	r := goskia.RectXYWH(float32(x), float32(y), float32(w), float32(h))
	skPaint := toSkiaPaint(p)
	defer skPaint.Release()
	c.canvas.DrawRect(r, skPaint)
	c.dirty = true
}

func (c *SkiaCanvas) DrawRoundedRect(x, y, w, h, r float64, p paint.Paint) {
	if r <= 0 {
		c.DrawRect(x, y, w, h, p)
		return
	}
	halfMin := math.Min(w, h) / 2
	if r > halfMin {
		r = halfMin
	}
	rect := goskia.RectXYWH(float32(x), float32(y), float32(w), float32(h))
	skPaint := toSkiaPaint(p)
	defer skPaint.Release()
	c.canvas.DrawRoundRect(rect, float32(r), float32(r), skPaint)
	c.dirty = true
}

func (c *SkiaCanvas) DrawCircle(cx, cy, radius float64, p paint.Paint) {
	skPaint := toSkiaPaint(p)
	defer skPaint.Release()
	c.canvas.DrawCircle(float32(cx), float32(cy), float32(radius), skPaint)
	c.dirty = true
}

func (c *SkiaCanvas) DrawLine(x1, y1, x2, y2 float64, p paint.Paint) {
	skPaint := toSkiaPaint(p)
	defer skPaint.Release()
	c.canvas.DrawLine(float32(x1), float32(y1), float32(x2), float32(y2), skPaint)
	c.dirty = true
}

func (c *SkiaCanvas) DrawText(text string, x, y float64, font Font, p paint.Paint) {
	if text == "" {
		return
	}
	skPaint := toSkiaPaint(p)
	defer skPaint.Release()
	// 斜体：中文字体无斜体变体，用 Skia skew 做 fake italic（以基线为原点，字形上部右倾）。
	italic := font.Style == FontStyleItalic
	if font.Weight == FontWeightBold {
		// fake bold：在粗体字体基础上再描边，与正文的区别更明显
		skPaint.SetStyle(goskia.PaintStyleStrokeAndFill)
		skPaint.SetStrokeWidth(float32(font.Size) * 0.028)
	}
	curX := float32(x)
	baseY := float32(y)
	if italic {
		c.canvas.Save()
		c.canvas.Translate(float32(x), float32(y))
		c.canvas.Skew(-0.22, 0)
		curX = 0
		baseY = 0
	}
	c.eachTextRun(text, font, func(seg string, skFont *goskia.Font) {
		if skFont == nil {
			return
		}
		c.canvas.DrawText(seg, curX, baseY, skFont, skPaint)
		w, _ := skFont.MeasureText(seg, skPaint)
		curX += w
	})
	if italic {
		c.canvas.Restore()
	}
	c.dirty = true
}

// eachTextRun 把文本按"主字体是否覆盖"分段，对每段调用 fn(segment, skFont)。
// 主字体覆盖的连续字符用主字体；缺失字符段改用对应脚本的系统 fallback 字体，
// 从而正确显示阿拉伯/希伯来/emoji 等（避免豆腐块）。
func (c *SkiaCanvas) eachTextRun(text string, font Font, fn func(seg string, skFont *goskia.Font)) {
	runes := []rune(text)
	i := 0
	for i < len(runes) {
		covered := RuneCoveredByPrimary(runes[i])
		j := i + 1
		for j < len(runes) && RuneCoveredByPrimary(runes[j]) == covered {
			j++
		}
		var skFont *goskia.Font
		if covered {
			skFont = c.getOrCreateSkiaFont(font)
		} else {
			skFont = c.getFallbackFont(font, runes[i])
		}
		fn(string(runes[i:j]), skFont)
		i = j
	}
}

// getFallbackFont 返回适配字符 r 所属脚本的系统 fallback 字体（按需缓存）。
func (c *SkiaCanvas) getFallbackFont(font Font, r rune) *goskia.Font {
	fontStoreMu.Lock()
	defer fontStoreMu.Unlock()
	fp := font.Size
	if fp <= 0 {
		fp = 16
	}
	family := FallbackFamily(r)
	key := fmt.Sprintf("fb|%s|%.1f|%d", family, fp, font.Weight)
	if f, ok := c.fontCache[key]; ok {
		return f
	}
	tfKey := fmt.Sprintf("fbtf|%s|%d", family, font.Weight)
	tf, ok := c.typefaceCache[tfKey]
	if !ok {
		style := goskia.FontStyleNormal
		if font.Weight == FontWeightBold {
			style = goskia.FontStyleBold
		}
		tf = goskia.NewTypeface(family, style)
		if tf == nil {
			tf = goskia.DefaultTypeface()
		}
		c.typefaceCache[tfKey] = tf
	}
	f := goskia.NewFont(tf, float32(fp))
	f.SetEdging(goskia.FontEdgingAntialias)
	f.SetSubpixel(true)
	c.fontCache[key] = f
	return f
}

func (c *SkiaCanvas) DrawImage(img image.Image, dst types.Rect) {
	bounds := img.Bounds()
	if bounds.Empty() {
		return
	}
	skImg := c.goImageToSkiaImage(img)
	if skImg == nil {
		return
	}
	defer skImg.Release()

	c.canvas.DrawImageRect(skImg,
		goskia.RectXYWH(0, 0, float32(bounds.Dx()), float32(bounds.Dy())),
		goskia.RectXYWH(float32(dst.X), float32(dst.Y), float32(dst.Width), float32(dst.Height)),
		goskia.SamplingOptions{},
		nil,
	)
	c.dirty = true
}

// goImageToSkiaImage 将 Go image.Image 转换为 skia.Image。
func (c *SkiaCanvas) goImageToSkiaImage(img image.Image) *goskia.Image {
	bounds := img.Bounds()
	w := bounds.Dx()
	h := bounds.Dy()
	if w <= 0 || h <= 0 {
		return nil
	}
	rgba := image.NewRGBA(bounds)
	draw.Draw(rgba, bounds, img, bounds.Min, draw.Src)

	info := goskia.NewImageInfo(w, h, goskia.ColorTypeRGBA8888, goskia.AlphaTypePremul)
	skImg, err := goskia.NewImageFromPixels(info, rgba.Pix, rgba.Stride)
	if err != nil {
		return nil
	}
	return skImg
}

// ============================================================================
// 文本测量
// ============================================================================

func (c *SkiaCanvas) MeasureText(text string, font Font) TextMetrics {
	if text == "" {
		return TextMetrics{}
	}
	mp := goskia.NewPaint()
	defer mp.Release()

	var total float32
	inkTop, inkBottom := float32(0), float32(0)
	haveInk := false
	c.eachTextRun(text, font, func(seg string, skFont *goskia.Font) {
		if skFont == nil {
			return
		}
		w, b := skFont.MeasureText(seg, mp) // b=tight 墨迹盒（相对基线）
		total += w
		if !b.IsEmpty() {
			if !haveInk || b.Top < inkTop {
				inkTop = b.Top
			}
			if !haveInk || b.Bottom > inkBottom {
				inkBottom = b.Bottom
			}
			haveInk = true
		}
	})

	ascent, descent, lineHeight := c.fontVMetrics(font)
	return TextMetrics{
		Width:     float64(total),
		Height:    lineHeight,
		Ascent:    ascent,
		Descent:   descent,
		Advance:   float64(total),
		InkTop:    float64(inkTop),
		InkBottom: float64(inkBottom),
	}
}

// fontVMetrics 返回字体的**真实**垂直度量，全部来自 Skia sk_font_get_metrics：
// ascent/descent 为正值（基线到顶/底的距离），lineHeight 为 Skia 推荐行距。
// 取代过去 size×0.8 / size×0.2 / size×1.3 的私有经验估算（那是文字垂直偏位的根因）。
func (c *SkiaCanvas) fontVMetrics(font Font) (ascent, descent, lineHeight float64) {
	skFont := c.getOrCreateSkiaFont(font)
	if skFont == nil { // 拿不到 Skia 字体时的极端兜底（正常不会发生）
		s := font.Size
		if s <= 0 {
			s = 16
		}
		return s * 0.8, s * 0.2, s * 1.2
	}
	m, spacing := skFont.Metrics()
	ascent = float64(-m.Ascent)   // fAscent 为负（基线以上），取正
	descent = float64(m.Descent)  // fDescent 为正（基线以下）
	lineHeight = float64(spacing) // = descent - ascent + leading
	if lineHeight <= 0 {
		lineHeight = ascent + descent
	}
	return ascent, descent, lineHeight
}

// ============================================================================
// 字体管理
// ============================================================================

// fontCacheKey 生成字体缓存键。
func fontCacheKey(f Font) string {
	family := f.Family
	if family == "" {
		family = "sans-serif"
	}
	return family + "-" + strconv.Itoa(int(f.Size*10)) + "-" + strconv.Itoa(int(f.Weight))
}

// getOrCreateSkiaFont 获取或创建 goskia Font。
func (c *SkiaCanvas) getOrCreateSkiaFont(font Font) *goskia.Font {
	fontStoreMu.Lock()
	defer fontStoreMu.Unlock()

	fp := font.Size
	if fp <= 0 {
		fp = 16
	}

	key := fontCacheKey(font)
	if f, ok := c.fontCache[key]; ok {
		return f
	}

	typeface := c.getOrCreateTypefaceLocked(font)
	skFont := goskia.NewFont(typeface, float32(fp))
	skFont.SetEdging(goskia.FontEdgingAntialias)
	skFont.SetHinting(goskia.FontHintingNormal)
	skFont.SetSubpixel(true)

	c.fontCache[key] = skFont
	return skFont
}

// getOrCreateTypefaceLocked 获取或创建 goskia Typeface（调用者需持有 mu 锁）。
func (c *SkiaCanvas) getOrCreateTypefaceLocked(font Font) *goskia.Typeface {
	weight := font.Weight
	// Typeface 与字号无关（字号是 NewFont(typeface,size) 那层的事），仅按字重缓存。
	// 否则每个不同字号都会各加载一份完整字库 Typeface，内存成倍膨胀。
	tfKey := fmt.Sprintf("typeface-w%d", weight)
	if tf, ok := c.typefaceCache[tfKey]; ok {
		return tf
	}

	var tf *goskia.Typeface

	// 从项目 fonts 目录加载
	if fontDir := getProjectFontDir(); fontDir != "" {
		var fontFiles []string
		switch weight {
		case FontWeightBold:
			fontFiles = []string{
				filepath.Join(fontDir, "AlibabaPuHuiTi-3-85-Bold.ttf"),
				filepath.Join(fontDir, "AlibabaPuHuiTi-3-65-Medium.ttf"),
				filepath.Join(fontDir, "AlibabaPuHuiTi-3-55-Regular.ttf"),
			}
		case FontWeightMedium:
			fontFiles = []string{
				filepath.Join(fontDir, "AlibabaPuHuiTi-3-65-Medium.ttf"),
				filepath.Join(fontDir, "AlibabaPuHuiTi-3-55-Regular.ttf"),
				filepath.Join(fontDir, "AlibabaPuHuiTi-3-85-Bold.ttf"),
			}
		default:
			fontFiles = []string{
				filepath.Join(fontDir, "AlibabaPuHuiTi-3-65-Medium.ttf"),
				filepath.Join(fontDir, "AlibabaPuHuiTi-3-55-Regular.ttf"),
				filepath.Join(fontDir, "AlibabaPuHuiTi-3-85-Bold.ttf"),
			}
		}
		for _, fp := range fontFiles {
			data, err := os.ReadFile(fp)
			if err == nil {
				tf = goskia.NewTypefaceFromData(data, 0)
				if tf != nil {
					break
				}
			}
		}
	}

	// 回退到系统默认字体
	if tf == nil {
		skStyle := goskia.FontStyleNormal
		if weight == FontWeightBold {
			skStyle = goskia.FontStyleBold
		}
		tf = goskia.NewTypeface(font.Family, skStyle)
		if tf == nil {
			tf = goskia.DefaultTypeface()
		}
	}

	if tf != nil {
		c.typefaceCache[tfKey] = tf
	}
	return tf
}

// getOrCreateTypefaceForCoverage 返回主字库 Normal 字重的全局共享 Typeface，
// 供字符覆盖检测复用——避免为覆盖检测单独再加载一份字库（原 primarySfnt）。
func (c *SkiaCanvas) getOrCreateTypefaceForCoverage() *goskia.Typeface {
	fontStoreMu.Lock()
	defer fontStoreMu.Unlock()
	return c.getOrCreateTypefaceLocked(Font{Weight: FontWeightNormal, Size: 16})
}

// ============================================================================
// 画布属性
// ============================================================================

// Resize 调整画布尺寸：重建 Skia surface 与缓存 RGBA 图像。
// 窗口大小变化时由 Pipeline.Resize 通过接口断言调用；
// 若不重建，画布会停在初始尺寸——放大窗口后新增区域无人绘制(背景铺不满、组件画到画布外)。
func (c *SkiaCanvas) Resize(width, height int) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if width <= 0 || height <= 0 || (width == c.width && height == c.height) {
		return
	}

	// 释放旧 surface
	if c.surf != nil {
		c.surf.Release()
		c.surf = nil
		c.canvas = nil
	}

	c.width = width
	c.height = height
	c.dirty = true

	if c.gpu {
		// GPU 模式：按新尺寸重新包装窗口 FBO（调用方须已 MakeCurrent + glViewport）
		if err := c.makeGPUSurface(); err != nil {
			panic("skia: failed to resize GPU surface: " + err.Error())
		}
		return
	}

	// 创建新尺寸的 Raster Surface（N32 Premul = RGBA8888）
	surf, err := goskia.NewRasterSurfaceN32Premul(width, height)
	if err != nil {
		panic("skia: failed to resize raster surface: " + err.Error())
	}
	c.surf = surf
	c.canvas = surf.Canvas()
	c.canvas.Clear(goskia.ColorWhite)

	// 重建缓存 RGBA（供 PNG / 纹理上传）
	c.img = image.NewRGBA(image.Rect(0, 0, width, height))
}

func (c *SkiaCanvas) Width() float64  { return float64(c.width) }
func (c *SkiaCanvas) Height() float64 { return float64(c.height) }

// Flush 完成绘制并将 Skia 渲染结果同步到内部 image.RGBA。
func (c *SkiaCanvas) Flush() error {
	if c.gpu {
		// GPU 模式：把绘制提交到窗口 framebuffer，由 app 随后 SwapBuffers 显示。无回读、无 PNG。
		c.gpuCtx.FlushAndSubmit(false)
		c.dirty = false
		return nil
	}
	if !c.dirty {
		return nil
	}

	snapshot := c.surf.Snapshot()
	if snapshot == nil {
		return nil
	}
	defer snapshot.Release()

	// 编码为 PNG 字节
	data, err := snapshot.Encode(goskia.EncodedFormatPNG, 100)
	if err != nil {
		return err
	}

	// 解码回 image.RGBA
	decoded, err := png.Decode(bytes.NewReader(data))
	if err != nil {
		return err
	}

	bounds := decoded.Bounds()
	if c.img == nil || c.img.Bounds().Size() != bounds.Size() {
		c.img = image.NewRGBA(bounds)
	}
	draw.Draw(c.img, bounds, decoded, bounds.Min, draw.Src)

	c.dirty = false
	return nil
}

// SaveToPNG 将当前帧缓冲保存为 PNG 文件（用于调试和视觉验证）。
func (c *SkiaCanvas) SaveToPNG(filename string) error {
	if err := c.Flush(); err != nil {
		return err
	}
	f, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer f.Close()
	return png.Encode(f, c.img)
}


