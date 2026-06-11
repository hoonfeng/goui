package widget

import (
	"bytes"
	"image"
	"image/draw"
	"image/gif" // GIF 动画逐帧解码（同时注册 image.Decode 的 gif 解码器）
	_ "image/jpeg"
	_ "image/png"
	"os"
	"path/filepath"
	"strings"
	"time"

	_ "golang.org/x/image/bmp"  // 注册 BMP 解码器
	_ "golang.org/x/image/webp" // 注册 WebP 解码器

	"github.com/hoonfeng/svg"
	"github.com/hoonfeng/goui/pkg/animation"
	"github.com/hoonfeng/goui/pkg/canvas"
	"github.com/hoonfeng/goui/internal/layout"
	"github.com/hoonfeng/goui/pkg/paint"
	"github.com/hoonfeng/goui/pkg/types"
)

// ImageFit 图片填充模式
type ImageFit int

const (
	ImageFitFill    ImageFit = iota // 拉伸填充（可能变形）
	ImageFitContain                 // 保持比例，完整显示
	ImageFitCover                   // 保持比例，覆盖容器
)

// Image 图片显示控件
type Image struct {
	StatelessWidget
	Source      image.Image
	Width       float64 // 0 表示自动
	Height      float64 // 0 表示自动
	Fit         ImageFit
	BgColor     *types.Color // 背景色（可选）
	BorderColor *types.Color // 边框色（可选）

	frames []image.Image // GIF 动画完整帧（>1 时启用逐帧播放）；单帧/静态用 Source
	delays []int         // 各帧延迟（单位 1/100 秒，对齐 image/gif）
}

// NewImage 用已解码的 image.Image 创建图片组件。
func NewImage(src image.Image) *Image { return &Image{Source: src} }

// NewImageFromBytes 从字节解码图片（自动识别 PNG/JPEG/GIF）。
func NewImageFromBytes(data []byte) (*Image, error) {
	// GIF 多帧 → 动画（合成完整帧）
	if frames, delays := decodeGIFFrames(data); len(frames) > 1 {
		return &Image{Source: frames[0], frames: frames, delays: delays}, nil
	}
	img, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	return &Image{Source: img}, nil
}

// decodeGIFFrames 把 GIF 解码并合成为完整帧（处理 disposal None/Background）；
// 非 GIF 或单帧返回 nil。各帧可能只含变化区域，需在累积画布上叠加得到完整帧。
func decodeGIFFrames(data []byte) ([]image.Image, []int) {
	g, err := gif.DecodeAll(bytes.NewReader(data))
	if err != nil || len(g.Image) <= 1 {
		return nil, nil
	}
	bounds := image.Rect(0, 0, g.Config.Width, g.Config.Height)
	cv := image.NewRGBA(bounds)
	frames := make([]image.Image, len(g.Image))
	for i, fr := range g.Image {
		draw.Draw(cv, fr.Bounds(), fr, fr.Bounds().Min, draw.Over)
		full := image.NewRGBA(bounds)
		draw.Draw(full, bounds, cv, image.Point{}, draw.Src)
		frames[i] = full
		// 为下一帧处理 disposal：Background=清除本帧区域为透明
		if i < len(g.Disposal) && g.Disposal[i] == gif.DisposalBackground {
			draw.Draw(cv, fr.Bounds(), image.Transparent, image.Point{}, draw.Src)
		}
	}
	return frames, g.Delay
}

// NewImageFromFile 加载本地图片：.png/.jpg/.jpeg/.gif 解码，.svg 光栅化。
func NewImageFromFile(path string) (*Image, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	if strings.EqualFold(filepath.Ext(path), ".svg") {
		return NewImageFromSVG(string(data), 0, 0)
	}
	return NewImageFromBytes(data)
}

// NewImageFromSVG 把 SVG 内容光栅化为图片（w/h<=0 时默认 128×128 渲染分辨率，
// 显示尺寸再由 Width/Height 控制）。
func NewImageFromSVG(svgContent string, w, h int) (*Image, error) {
	doc, err := svg.Parse(svgContent)
	if err != nil {
		return nil, err
	}
	// w/h<=0 时 Render 内部回退到 SVG 自身 width/height（避免坐标不缩放导致的偏移）
	rgba, err := doc.Render(w, h)
	if err != nil {
		return nil, err
	}
	return &Image{Source: rgba}, nil
}

// WithFit 设置填充模式（链式）。
func (img *Image) WithFit(f ImageFit) *Image { img.Fit = f; return img }

// WithSize 设置显示尺寸（链式）。
func (img *Image) WithSize(w, h float64) *Image { img.Width, img.Height = w, h; return img }

// CreateElement 创建 ImageElement
func (img *Image) CreateElement() Element {
	return &ImageElement{
		BaseElement: BaseElement{widget: img},
		image:       img,
	}
}

// ImageElement 图片运行时 Element
type ImageElement struct {
	BaseElement
	image    *Image
	frameIdx int
	anim     *animation.Controller
}

// Build 返回 nil（没有子控件）
func (e *ImageElement) Build() []Element { return nil }

// Layout 测量尺寸
func (e *ImageElement) Layout(ctx *layout.LayoutContext) layout.LayoutResult {
	img := e.image

	// 根据源图计算尺寸
	srcW, srcH := 0.0, 0.0
	if img.Source != nil {
		bounds := img.Source.Bounds()
		srcW = float64(bounds.Dx())
		srcH = float64(bounds.Dy())
	}

	// 确定最终尺寸
	width := img.Width
	height := img.Height

	if width <= 0 && height <= 0 {
		// 两者都未指定，使用原始图片尺寸
		width = srcW
		height = srcH
	} else if width <= 0 {
		// 根据高度等比计算宽度
		if srcH > 0 {
			width = srcW * (height / srcH)
		}
	} else if height <= 0 {
		// 根据宽度等比计算高度
		if srcW > 0 {
			height = srcH * (width / srcW)
		}
	}

	if width <= 0 {
		width = 100
	}
	if height <= 0 {
		height = 100
	}

	e.size = ctx.Constraints.Constrain(types.Size{
		Width:  width,
		Height: height,
	})
	return layout.LayoutResult{Size: e.size}
}

// Paint 绘制图片
func (e *ImageElement) Paint(cvs canvas.Canvas, offset types.Point) {
	e.ensureAnim()
	pos := e.Offset()
	img := e.image

	// 当前帧（GIF 动画取 frames[frameIdx]，否则用 Source）
	src := img.Source
	if len(img.frames) > 1 && e.frameIdx >= 0 && e.frameIdx < len(img.frames) {
		src = img.frames[e.frameIdx]
	}

	// 绘制背景色
	if img.BgColor != nil {
		bgPaint := paint.DefaultPaint()
		bgPaint.Color = *img.BgColor
		cvs.DrawRect(pos.X, pos.Y, e.size.Width, e.size.Height, bgPaint)
	}

	// 绘制图片
	if src != nil {
		bounds := src.Bounds()
		srcW := float64(bounds.Dx())
		srcH := float64(bounds.Dy())

		if srcW > 0 && srcH > 0 {
			dstX, dstY, dstW, dstH := pos.X, pos.Y, e.size.Width, e.size.Height

			// 根据填充模式计算实际绘制区域
			switch img.Fit {
			case ImageFitContain:
				scale := min(e.size.Width/srcW, e.size.Height/srcH)
				dstW = srcW * scale
				dstH = srcH * scale
				dstX = pos.X + (e.size.Width-dstW)/2
				dstY = pos.Y + (e.size.Height-dstH)/2

			case ImageFitCover:
				scale := max(e.size.Width/srcW, e.size.Height/srcH)
				dstW = srcW * scale
				dstH = srcH * scale
				dstX = pos.X + (e.size.Width-dstW)/2
				dstY = pos.Y + (e.size.Height-dstH)/2
			}

			cvs.DrawImage(src, types.Rect{
				X: dstX, Y: dstY,
				Width: dstW, Height: dstH,
			})
		}
	}

	// 绘制边框
	if img.BorderColor != nil {
		borderPaint := paint.DefaultStrokePaint()
		borderPaint.Color = *img.BorderColor
		borderPaint.StrokeWidth = 1.0
		cvs.DrawRect(pos.X, pos.Y, e.size.Width, e.size.Height, borderPaint)
	}
}

// Update 更新 Image 配置
func (e *ImageElement) Update(newWidget Widget) {
	if newImg, ok := newWidget.(*Image); ok {
		e.image = newImg
		e.BaseElement.widget = newWidget
		e.dirty = true
	}
}

// ensureAnim 首次为 GIF 动画启动逐帧循环（按各帧 delay 切换 frameIdx）。
func (e *ImageElement) ensureAnim() {
	if e.anim != nil || len(e.image.frames) <= 1 {
		return
	}
	total := 0
	for _, d := range e.image.delays {
		total += d
	}
	if total <= 0 {
		total = len(e.image.frames) * 10 // 兜底：每帧 100ms
	}
	e.anim = animation.NewController(time.Duration(total)*10*time.Millisecond, nil)
	e.anim.Repeat = true
	e.anim.OnUpdate = func(v float64) {
		elapsed := v * float64(total) // 百分之一秒为单位
		acc, idx := 0.0, 0
		for i, d := range e.image.delays {
			acc += float64(d)
			idx = i
			if elapsed < acc {
				break
			}
		}
		if idx != e.frameIdx {
			e.frameIdx = idx
			if OnNeedsRepaint != nil {
				OnNeedsRepaint()
			}
		}
	}
	e.anim.Start()
}

// Unmount 停止动画，防止 controller 泄漏导致主循环空转。
func (e *ImageElement) Unmount() {
	if e.anim != nil {
		e.anim.Stop()
		e.anim = nil
	}
	e.BaseElement.Unmount()
}
