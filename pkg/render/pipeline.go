package render

import (
	"log"
	"time"

	"github.com/hoonfeng/goui/internal/layout"
	"github.com/hoonfeng/goui/pkg/canvas"
	"github.com/hoonfeng/goui/pkg/paint"
	"github.com/hoonfeng/goui/pkg/types"
	"github.com/hoonfeng/goui/pkg/widget"
)

// ── 帧率诊断统计 ──
var (
	DiagFrameCount   int64
	DiagTotalBuild   time.Duration
	DiagTotalLayout  time.Duration
	DiagTotalPaint   time.Duration
	DiagTotalFlush   time.Duration
	DiagTotalElapsed time.Duration
	DiagLastLog      time.Time
	DiagLastBuild    time.Duration
	DiagLastLayout   time.Duration
	DiagLastPaint    time.Duration
	DiagLastFlush    time.Duration
)

// Pipeline 是渲染管线，管理从控件树到最终显示的全过程。
type Pipeline struct {
	rootLayer    *Layer
	rootElement  widget.Element
	finalCanvas  canvas.Canvas
	width        int
	height       int
	needsRepaint bool
	needsLayout  bool
	didRender    bool // 上次 Render() 是否实际执行了绘制
}

// NewPipeline 创建渲染管线
func NewPipeline(width, height int, finalCanvas canvas.Canvas) *Pipeline {
	return &Pipeline{
		rootLayer: NewLayer(types.Rect{
			X: 0, Y: 0,
			Width: float64(width), Height: float64(height),
		}),
		finalCanvas:  finalCanvas,
		width:        width,
		height:       height,
		needsRepaint: true,
		needsLayout:  true,
	}
}

// SetRootElement 设置根 Element
func (p *Pipeline) SetRootElement(el widget.Element) {
	p.rootElement = el
	p.needsLayout = true
	p.needsRepaint = true
}

// SetRootLayer 设置根图层
func (p *Pipeline) SetRootLayer(layer *Layer) {
	p.rootLayer = layer
	p.needsRepaint = true
}

// MarkNeedsRepaint 标记需要重绘
func (p *Pipeline) MarkNeedsRepaint() {
	p.needsRepaint = true
}

// MarkNeedsLayout 标记需要重新布局
func (p *Pipeline) MarkNeedsLayout() {
	p.needsLayout = true
	p.needsRepaint = true
}

// Resize 调整渲染管线尺寸
func (p *Pipeline) Resize(width, height int) {
	p.width = width
	p.height = height
	p.rootLayer.Bounds = types.Rect{
		X: 0, Y: 0,
		Width: float64(width), Height: float64(height),
	}
	p.needsLayout = true
	p.needsRepaint = true

	// 调整 Canvas 尺寸（SkiaCanvas：GPU 模式重建 FBO surface / raster 模式重建位图）
	if resizer, ok := p.finalCanvas.(interface{ Resize(int, int) }); ok {
		resizer.Resize(width, height)
	}
}

// PerformLayout 执行布局计算并递归构建 Element 树
func (p *Pipeline) PerformLayout() {
	if p.rootElement == nil {
		return
	}

	// 递归构建 Element 树
	p.buildTree(p.rootElement)

	// 使用窗口尺寸作为根约束
	ctx := &layout.LayoutContext{
		Constraints: layout.BoxConstraints{
			MinWidth:  0,
			MaxWidth:  float64(p.width),
			MinHeight: 0,
			MaxHeight: float64(p.height),
		},
	}
	p.rootElement.Layout(ctx)
	p.needsLayout = false
}

// EnsureLayout 确保布局已执行（事件处理前调用，保证 HitTest 能正确工作）
func (p *Pipeline) EnsureLayout() {
	if p.needsLayout && p.rootElement != nil {
		p.PerformLayout()
	}
}

// buildTree 递归构建 Element 树
func (p *Pipeline) buildTree(el widget.Element) {
	children := el.Build()
	for _, child := range children {
		p.buildTree(child)
	}
}

// Render 执行渲染
func (p *Pipeline) Render() error {
	p.didRender = false
	if !p.needsRepaint && !p.needsLayout {
		return nil
	}

	// ── 帧率诊断计时 ──
	frameStart := time.Now()
	var buildDur, layoutDur, paintDur, flushDur time.Duration

	// 需要先布局再绘制
	if p.needsLayout {
		t0 := time.Now()
		p.PerformLayout()
		layoutDur = time.Since(t0)
		if !DiagLastLog.IsZero() {
			DiagTotalLayout += layoutDur
		}
	}

	// 清理画布
	p.clearCanvas()

	// 从根 Element 开始绘制（带视口裁剪）
	if p.rootElement != nil {
		t0 := time.Now()
		p.finalCanvas.Save()
		p.finalCanvas.ClipRect(0, 0, float64(p.width), float64(p.height))
		p.rootElement.Paint(p.finalCanvas, types.Point{})
		p.finalCanvas.Restore()
		paintDur = time.Since(t0)
		if !DiagLastLog.IsZero() {
			DiagTotalPaint += paintDur
		}
	}

	// 刷新画布到屏幕
	{
		t0 := time.Now()
		if err := p.finalCanvas.Flush(); err != nil {
			return err
		}
		flushDur = time.Since(t0)
		if !DiagLastLog.IsZero() {
			DiagTotalFlush += flushDur
		}
	}

	frameElapsed := time.Since(frameStart)
	if !DiagLastLog.IsZero() {
		DiagFrameCount++
		DiagTotalElapsed += frameElapsed
		DiagLastBuild = buildDur
		DiagLastLayout = layoutDur
		DiagLastPaint = paintDur
		DiagLastFlush = flushDur

		// 每 5 秒输出一次帧率诊断
		if time.Since(DiagLastLog) >= 5*time.Second {
			fps := float64(DiagFrameCount) / time.Since(DiagLastLog).Seconds()
			avgPaint := DiagTotalPaint / time.Duration(DiagFrameCount)
			avgFlush := DiagTotalFlush / time.Duration(DiagFrameCount)
			avgLayout := DiagTotalLayout / time.Duration(DiagFrameCount)
			log.Printf("[perf] %.1f fps | frames=%d | build=%v/layout=%v/paint=%v/flush=%v | lastBuild=%v",
				fps, DiagFrameCount,
				DiagTotalBuild/time.Duration(DiagFrameCount),
				avgLayout, avgPaint, avgFlush,
				DiagLastBuild)

			// 重置统计
			DiagFrameCount = 0
			DiagTotalBuild = 0
			DiagTotalLayout = 0
			DiagTotalPaint = 0
			DiagTotalFlush = 0
			DiagTotalElapsed = 0
			DiagLastLog = time.Now()
		}
	} else {
		DiagLastLog = time.Now()
	}

	p.needsRepaint = false
	p.didRender = true
	return nil
}

// DidRender 返回上次 Render() 调用是否实际执行了绘制
func (p *Pipeline) DidRender() bool {
	return p.didRender
}

// clearCanvas 清空画布
func (p *Pipeline) clearCanvas() {
	// 使用白色背景填充整个画布
	bgPaint := paint.DefaultPaint()
	bgPaint.Color = types.ColorWhite
	p.finalCanvas.DrawRect(0, 0, float64(p.width), float64(p.height), bgPaint)
}

// HitTest 对指定坐标进行命中测试，返回最内层的 Element。
// 坐标相对于窗口客户区（左上角为原点）。
// 返回 nil 表示没有命中任何 Element。
func (p *Pipeline) HitTest(x, y float64) widget.Element {
	if p.rootElement == nil {
		return nil
	}
	return hitTestRecursive(p.rootElement, x, y)
}

// hitTestRecursive 递归执行命中测试
func hitTestRecursive(el widget.Element, x, y float64) widget.Element {
	// 若元素对子树施加了坐标变换（如 Transform），先把命中点逆变换到子坐标系，
	// 使旋转/缩放后的子元素命中判定与视觉一致。
	childX, childY := x, y
	if tr, ok := el.(widget.HitTestTransformer); ok {
		childX, childY = tr.TransformHitPoint(x, y)
	}
	// 先检查子节点（逆序，后绘制的在上层）
	children := el.Children()
	for i := len(children) - 1; i >= 0; i-- {
		if result := hitTestRecursive(children[i], childX, childY); result != nil {
			return result
		}
	}

	// 装饰型元素（图标等）穿透命中测试：子元素已在上面测过，自身不拦截，
	// 使点击落到可交互的祖先（如 Button）。否则按钮内的图标会截走点击，按钮点不动。
	if ig, ok := el.(widget.HitTestIgnorer); ok && ig.IgnoreHitTest() {
		return nil
	}

	// 检查自身：是否在 Element 的边界内
	offset := el.Offset()
	size := el.Size()
	// 只对有效尺寸进行判断
	if size.Width <= 0 || size.Height <= 0 {
		return nil
	}
	if x >= offset.X && x <= offset.X+size.Width &&
		y >= offset.Y && y <= offset.Y+size.Height {
		return el
	}
	return nil
}

// Canvas 返回当前画布
func (p *Pipeline) Canvas() canvas.Canvas {
	return p.finalCanvas
}

// Width 返回管线宽度
func (p *Pipeline) Width() int { return p.width }

// Height 返回管线高度
func (p *Pipeline) Height() int { return p.height }

// RootElement 返回根 Element
func (p *Pipeline) RootElement() widget.Element {
	return p.rootElement
}
