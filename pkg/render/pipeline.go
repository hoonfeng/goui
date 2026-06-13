package render

import (
	"log"

	"github.com/hoonfeng/goui/internal/layout"
	"github.com/hoonfeng/goui/pkg/canvas"
	"github.com/hoonfeng/goui/pkg/paint"
	"github.com/hoonfeng/goui/pkg/types"
	"github.com/hoonfeng/goui/pkg/widget"
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
	log.Println("goui: PerformLayout: buildTree 开始")
	p.buildTree(p.rootElement)
	log.Println("goui: PerformLayout: buildTree 完成")

	// 使用窗口尺寸作为根约束
	ctx := &layout.LayoutContext{
		Constraints: layout.BoxConstraints{
			MinWidth:  0,
			MaxWidth:  float64(p.width),
			MinHeight: 0,
			MaxHeight: float64(p.height),
		},
	}
	log.Println("goui: PerformLayout: Layout 开始")
	p.rootElement.Layout(ctx)
	log.Println("goui: PerformLayout: Layout 完成")
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

	// 需要先布局再绘制
	if p.needsLayout {
		p.PerformLayout()
	}

	// 清空画布为白色背景
	p.clearCanvas()

	// 从根 Element 开始绘制（带视口裁剪）
	// Save/ClipRect 让 Skia GPU 只处理窗口可见区域的像素，off-screen 元素（如
	// 滚动视图外的聊天消息/文件树节点）由 Skia 自动裁切，大幅减少像素着色器调用。
	// RESTORE 确保裁剪不影响后续帧或 HitTest 的坐标变换栈。
	if p.rootElement != nil {
		log.Println("goui: Pipeline.Render: Paint 开始")
		p.finalCanvas.Save()
		p.finalCanvas.ClipRect(0, 0, float64(p.width), float64(p.height))
		p.rootElement.Paint(p.finalCanvas, types.Point{})
		p.finalCanvas.Restore()
		log.Println("goui: Pipeline.Render: Paint 完成")
	} else {
		log.Println("goui: Pipeline.Render: rootElement 为 nil，跳过 Paint")
	}

	// 刷新画布到屏幕
	log.Println("goui: Pipeline.Render: Flush 开始")
	if err := p.finalCanvas.Flush(); err != nil {
		log.Printf("goui: Pipeline.Render: Flush 错误: %v", err)
		return err
	}
	log.Println("goui: Pipeline.Render: Flush 完成")

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
