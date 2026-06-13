package widget

import (
	"fmt"
	"reflect"
	"time"

	"github.com/hoonfeng/goui/pkg/animation"
	"github.com/hoonfeng/goui/pkg/canvas"
	"github.com/hoonfeng/goui/pkg/event"
	"github.com/hoonfeng/goui/internal/layout"
	"github.com/hoonfeng/goui/pkg/paint"
	"github.com/hoonfeng/goui/pkg/types"
)

// ScrollView 可滚动的视图控件（滚动地基）。
// 子内容在滚动方向上获得无界约束(可无限延伸)，超出视口的部分被裁剪；
// 滚轮改变 scrollOffset → 偏移重绘。OnScroll 回调供 Affix/Backtop 等监听滚动位置。
type ScrollView struct {
	SingleChildWidget
	ScrollDirection layout.FlexDirection     // 滚动方向(默认 column 竖向)
	ShowBar         bool                     // 显示滚动条
	OnScroll        func(offset types.Point) // 滚动回调(scrollOffset 变化时；供 Affix/Backtop 监听)
	ScrollEndToken  int                      // 改变此值(+SetState)→ 下次布局滚到末尾（聊天追加消息后滚到底）
}

// NewScrollView 创建竖向滚动视图(默认显示滚动条)。
func NewScrollView(child Widget) *ScrollView {
	return &ScrollView{
		SingleChildWidget: SingleChildWidget{Child: child},
		ScrollDirection:   layout.FlexColumn,
		ShowBar:           true,
	}
}

// Horizontal 改为水平滚动。
func (s *ScrollView) Horizontal() *ScrollView { s.ScrollDirection = layout.FlexRow; return s }

// WithBar 控制滚动条显示。
func (s *ScrollView) WithBar(show bool) *ScrollView { s.ShowBar = show; return s }

// WithOnScroll 注册滚动回调(scrollOffset 变化时触发)。
func (s *ScrollView) WithOnScroll(fn func(types.Point)) *ScrollView { s.OnScroll = fn; return s }

// ScrollViewElement ScrollView 的运行时 Element
type ScrollViewElement struct {
	BaseElement
	scrollView   *ScrollView
	child        Element
	scrollOffset  types.Point
	maxScroll     types.Point
	lastScrollEnd int             // 上次见到的 ScrollEndToken（受控滚到底用）
	fixedAffixes  []*AffixElement // 本帧已固定的 Affix，content 画完后在最上层重画(解决吸顶层级)

	// 竖向滚动条拖动状态
	draggingBar      bool
	dragStartMouseY  float64
	dragStartScrollY float64

	// 平滑滚动动画（Y轴）
	scrollAnimY     *animation.Controller
	scrollFromY     float64
	scrollTargetY   float64
	// 平滑滚动动画（X轴）
	scrollAnimX     *animation.Controller
	scrollFromX     float64
	scrollTargetX   float64
}

// CreateElement 创建 ScrollViewElement
func (s *ScrollView) CreateElement() Element {
	return &ScrollViewElement{
		BaseElement: BaseElement{widget: s},
		scrollView:  s,
	}
}

// Update 同步 widget 指针(relayout 复用时保留 scrollOffset 等运行时状态)。
func (e *ScrollViewElement) Update(newWidget Widget) {
	if ns, ok := newWidget.(*ScrollView); ok {
		e.scrollView = ns
	}
	e.BaseElement.widget = newWidget
	e.dirty = true
}

// ScrollOffset 返回当前滚动偏移(供外部如 Affix/Backtop 读取)。
func (e *ScrollViewElement) ScrollOffset() types.Point { return e.scrollOffset }

// MaxScroll 返回最大可滚动距离。
func (e *ScrollViewElement) MaxScroll() types.Point { return e.maxScroll }

// ScrollTo 设置滚动偏移(clamp 到合法范围)，供 Backtop 回顶等程序滚动使用。
func (e *ScrollViewElement) ScrollTo(offset types.Point) {
	e.scrollOffset.X = clamp(offset.X, 0, e.maxScroll.X)
	e.scrollOffset.Y = clamp(offset.Y, 0, e.maxScroll.Y)
	if e.child != nil {
		e.child.SetPosition(types.Point{X: -e.scrollOffset.X, Y: -e.scrollOffset.Y})
	}
	e.fireScroll()
	if OnNeedsRepaint != nil {
		OnNeedsRepaint()
	}
}

func (e *ScrollViewElement) fireScroll() {
	if e.scrollView.OnScroll != nil {
		e.scrollView.OnScroll(e.scrollOffset)
	}
	// 锚点随内容滚走 → 收起下拉/日历/级联等锚点浮层(主流交互)
	CloseAnchoredOverlays()
}

// Build 构建子 Element（reconcile：类型匹配则复用 + Update，保留 content 内运行时状态）
func (e *ScrollViewElement) Build() []Element {
	c := e.scrollView.Child
	if c == nil {
		if e.child != nil {
			e.child.Unmount()
			e.child = nil
		}
		e.children = nil
		return nil
	}
	if e.child != nil && reflect.TypeOf(e.child.Widget()) == reflect.TypeOf(c) {
		e.child.Update(c)
	} else {
		if e.child != nil {
			e.child.Unmount()
		}
		e.child = CreateElementFor(c)
		e.child.Mount(e, 0)
	}
	e.children = []Element{e.child}
	return e.children
}

// Layout 执行布局
func (e *ScrollViewElement) Layout(ctx *layout.LayoutContext) layout.LayoutResult {
	e.size = ctx.Constraints.Constrain(types.Size{
		Width:  ctx.Constraints.MaxWidth,
		Height: ctx.Constraints.MaxHeight,
	})

	if e.child != nil {
		// 子控件在滚动方向上获得无界约束（可以无限延伸）
		childConstraints := layout.BoxConstraints{
			MinWidth:  0,
			MaxWidth:  ctx.Constraints.MaxWidth,
			MinHeight: 0,
			MaxHeight: ctx.Constraints.MaxHeight,
		}
		if e.scrollView.ScrollDirection == layout.FlexColumn {
			childConstraints.MaxHeight = float64(1 << 29) // 垂直滚动时高度无界
		} else {
			childConstraints.MaxWidth = float64(1 << 29) // 水平滚动时宽度无界
		}

		result := e.child.Layout(&layout.LayoutContext{Constraints: childConstraints})

		// 计算最大滚动范围
		e.maxScroll = types.Point{
			X: max(0, result.Size.Width-e.size.Width),
			Y: max(0, result.Size.Height-e.size.Height),
		}

		// 受控滚到末尾：ScrollEndToken 变化时跳到底部（聊天追加消息后保持看见最新）。
		// 此机制优先于平滑动画，在 Layout 中直接跳转，确保流式输出时消息能及时出现在底部。
		if e.scrollView.ScrollEndToken != e.lastScrollEnd {
			e.stopSmoothScroll()
			e.scrollOffset.Y = e.maxScroll.Y
			e.scrollOffset.X = e.maxScroll.X
			e.lastScrollEnd = e.scrollView.ScrollEndToken
		} else if !e.isScrollAnimating() {
			// 非动画状态：把现有 offset clamp 回合法区间（内容变短时收回）
			e.scrollOffset.X = clamp(e.scrollOffset.X, 0, e.maxScroll.X)
			e.scrollOffset.Y = clamp(e.scrollOffset.Y, 0, e.maxScroll.Y)
		} else {
			// 动画进行中：不 clamp 当前位置（避免截断动画造成跳动），
			// 但若 maxScroll 变小（内容变短），更新目标位置防止越界。
			if e.scrollTargetY > e.maxScroll.Y {
				e.scrollTargetY = e.maxScroll.Y
				if e.scrollAnimY != nil && e.scrollAnimY.IsRunning() {
					e.scrollFromY = e.scrollOffset.Y
					e.restartAnimY()
				}
			}
			if e.scrollTargetX > e.maxScroll.X {
				e.scrollTargetX = e.maxScroll.X
				if e.scrollAnimX != nil && e.scrollAnimX.IsRunning() {
					e.scrollFromX = e.scrollOffset.X
					e.restartAnimX()
				}
			}
		}
		e.child.SetPosition(types.Point{X: -e.scrollOffset.X, Y: -e.scrollOffset.Y})

		layout.LogLayoutDetailed("ScrollView", fmt.Sprintf("%T", e.Widget()),
			ctx.Constraints, e.size,
			fmt.Sprintf("childSize=(%.0fx%.0f) maxScroll=(%.0f,%.0f) scrollOffset=(%.0f,%.0f)",
				result.Size.Width, result.Size.Height,
				e.maxScroll.X, e.maxScroll.Y,
				e.scrollOffset.X, e.scrollOffset.Y),
		)
	}

	return layout.LayoutResult{Size: e.size}
}

// Paint 绘制（裁剪到滚动区域，并在边缘叠加滚动条）
func (e *ScrollViewElement) Paint(cvs canvas.Canvas, offset types.Point) {
	pos := e.Offset()
	e.fixedAffixes = e.fixedAffixes[:0] // 重置本帧固定 Affix 收集
	// 裁剪到滚动区域
	cvs.Save()
	cvs.ClipRect(pos.X, pos.Y, e.size.Width, e.size.Height)
	if e.child != nil {
		e.child.Paint(cvs, offset) // 固定的 Affix 在此期间登记(自身不画)
	}
	// 固定的 Affix 在裁剪区内、所有内容之上重画(吸顶不被后续内容覆盖)
	for _, af := range e.fixedAffixes {
		af.paintFixed(cvs, offset)
	}
	cvs.Restore()

	// 滚动条（在裁剪外画，贴右/下边缘）
	if e.scrollView.ShowBar {
		e.paintScrollbar(cvs, pos)
	}
}

// registerFixedAffix 由子树中已触发固定的 Affix 调用，登记到本帧最上层重画队列。
func (e *ScrollViewElement) registerFixedAffix(a *AffixElement) {
	e.fixedAffixes = append(e.fixedAffixes, a)
}

const (
	scrollBarW      = 6.0
	scrollWheelStep = 50.0 // 每个滚轮格滚动的像素(win32 已把 DeltaY 归一化为 ±1/格)
)

// scrollAnimDuration 平滑滚动动画时长。
const scrollAnimDuration = 180 * time.Millisecond

// vThumbRect 返回竖向滚动条滑块的全局矩形(绘制与拖动命中共用)；无可滚动时 ok=false。
func (e *ScrollViewElement) vThumbRect() (types.Rect, bool) {
	if !e.scrollView.ShowBar || e.maxScroll.Y <= 0 {
		return types.Rect{}, false
	}
	pos := e.Offset()
	contentH := e.size.Height + e.maxScroll.Y
	thumbH := e.size.Height * e.size.Height / contentH
	if thumbH < 30 {
		thumbH = 30
	}
	thumbY := pos.Y + (e.size.Height-thumbH)*(e.scrollOffset.Y/e.maxScroll.Y)
	bx := pos.X + e.size.Width - scrollBarW - 2
	return types.Rect{X: bx, Y: thumbY, Width: scrollBarW, Height: thumbH}, true
}

// paintScrollbar 画半透明圆角滑块，长度/位置反映可视比例与滚动进度(el-scrollbar 风格)。
func (e *ScrollViewElement) paintScrollbar(cvs canvas.Canvas, pos types.Point) {
	tp := paint.DefaultPaint()
	tp.Color = types.ColorFromRGBA(144, 147, 153, 130) // 半透明灰滑块

	// 竖向滑块
	if r, ok := e.vThumbRect(); ok {
		cvs.DrawRoundedRect(r.X, r.Y, r.Width, r.Height, scrollBarW/2, tp)
	}

	// 横向滑块
	if e.maxScroll.X > 0 {
		contentW := e.size.Width + e.maxScroll.X
		thumbW := e.size.Width * e.size.Width / contentW
		if thumbW < 30 {
			thumbW = 30
		}
		thumbX := pos.X + (e.size.Width-thumbW)*(e.scrollOffset.X/e.maxScroll.X)
		by := pos.Y + e.size.Height - scrollBarW - 2
		cvs.DrawRoundedRect(thumbX, by, thumbW, scrollBarW, scrollBarW/2, tp)
	}
}

// HandleEvent 处理滚轮滚动 + 滚动条滑块拖动。
func (e *ScrollViewElement) HandleEvent(ev event.Event) bool {
	switch ev.Type() {
	case event.TypeMouseWheel:
		mouseEv := ev.(*event.MouseEvent)
		// 计算目标滚动位置
		targetX := clamp(e.scrollOffset.X-mouseEv.DeltaX*scrollWheelStep, 0, e.maxScroll.X)
		targetY := clamp(e.scrollOffset.Y-mouseEv.DeltaY*scrollWheelStep, 0, e.maxScroll.Y)
		// 使用平滑动画滚动到目标位置（替代原来的直接跳转）
		e.smoothScrollTo(targetX, targetY)
		// 自己消费滚轮后阻断冒泡，避免外层 ScrollView 同时滚动(嵌套时只滚最内层)
		ev.StopPropagation()
		if OnNeedsRepaint != nil {
			OnNeedsRepaint()
		}
		return true

	case event.TypeMouseDown:
		// 仅当按在竖向滑块上才进入拖动；否则返回 false 让事件穿透给内容
		me, ok := ev.(*event.MouseEvent)
		if !ok {
			return false
		}
		if r, okk := e.vThumbRect(); okk {
			// 滑块窄，命中区四周略放宽，便于点中
			if me.X >= r.X-6 && me.X < r.X+r.Width+2 && me.Y >= r.Y-2 && me.Y < r.Y+r.Height+2 {
				e.stopSmoothScroll() // 拖动滚动条时停止平滑动画
				e.draggingBar = true
				e.dragStartMouseY = me.Y
				e.dragStartScrollY = e.scrollOffset.Y
				// 把指针捕获抢到自己：滑块下方是内容元素，否则它会截走拖动 Move/松手 Up
				if RequestPointerCapture != nil {
					RequestPointerCapture(e)
				}
				ev.StopPropagation()
				return true
			}
		}
		return false

	case event.TypeMouseMove:
		if e.draggingBar {
			if me, ok := ev.(*event.MouseEvent); ok {
				e.dragBarTo(me.Y)
			}
			return true
		}
		return false

	case event.TypeMouseUp:
		if e.draggingBar {
			e.draggingBar = false
			if OnNeedsRepaint != nil {
				OnNeedsRepaint()
			}
			return true
		}
		return false

	case event.TypeMouseLeave:
		if !e.draggingBar { // 拖动中不重置，否则鼠标移出滑块区域（仍在窗口内）会释放拖动
			e.draggingBar = false
		}
		return false
	}
	return false
}

// ── 平滑滚动动画 ────────────────────────────────────────────

// smoothScrollTo 启动平滑滚动动画，将 scrollOffset 从当前位置过渡到 targetX/targetY。
// 如果已有动画在运行，自动重设为从当前位置到新目标的动画（不产生跳变）。
func (e *ScrollViewElement) smoothScrollTo(targetX, targetY float64) {
	// 处理 Y 轴
	if e.maxScroll.Y > 0 && targetY != e.scrollOffset.Y {
		if e.scrollAnimY != nil && e.scrollAnimY.IsRunning() {
			e.scrollAnimY.Stop()
		}
		e.scrollFromY = e.scrollOffset.Y
		e.scrollTargetY = targetY
		e.scrollAnimY = animation.NewController(scrollAnimDuration, animation.EaseOutCubic)
		e.scrollAnimY.OnUpdate = func(v float64) {
			e.scrollOffset.Y = animation.LerpFloat(e.scrollFromY, e.scrollTargetY, v)
			if e.child != nil {
				e.child.SetPosition(types.Point{X: -e.scrollOffset.X, Y: -e.scrollOffset.Y})
			}
			e.fireScroll()
		}
		e.scrollAnimY.OnDone = func() {
			// 最终确保精确到位（浮点累计误差补偿）
			e.scrollOffset.Y = e.scrollTargetY
			if e.child != nil {
				e.child.SetPosition(types.Point{X: -e.scrollOffset.X, Y: -e.scrollOffset.Y})
			}
			e.fireScroll()
			if OnNeedsRepaint != nil {
				OnNeedsRepaint()
			}
		}
		e.scrollAnimY.Start()
	}

	// 处理 X 轴
	if e.maxScroll.X > 0 && targetX != e.scrollOffset.X {
		if e.scrollAnimX != nil && e.scrollAnimX.IsRunning() {
			e.scrollAnimX.Stop()
		}
		e.scrollFromX = e.scrollOffset.X
		e.scrollTargetX = targetX
		e.scrollAnimX = animation.NewController(scrollAnimDuration, animation.EaseOutCubic)
		e.scrollAnimX.OnUpdate = func(v float64) {
			e.scrollOffset.X = animation.LerpFloat(e.scrollFromX, e.scrollTargetX, v)
			if e.child != nil {
				e.child.SetPosition(types.Point{X: -e.scrollOffset.X, Y: -e.scrollOffset.Y})
			}
			e.fireScroll()
		}
		e.scrollAnimX.OnDone = func() {
			e.scrollOffset.X = e.scrollTargetX
			if e.child != nil {
				e.child.SetPosition(types.Point{X: -e.scrollOffset.X, Y: -e.scrollOffset.Y})
			}
			e.fireScroll()
			if OnNeedsRepaint != nil {
				OnNeedsRepaint()
			}
		}
		e.scrollAnimX.Start()
	}
}

// isScrollAnimating 返回是否有任何方向的平滑滚动动画正在运行。
func (e *ScrollViewElement) isScrollAnimating() bool {
	return (e.scrollAnimY != nil && e.scrollAnimY.IsRunning()) ||
		(e.scrollAnimX != nil && e.scrollAnimX.IsRunning())
}

// stopSmoothScroll 停止所有正在运行的平滑滚动动画。
func (e *ScrollViewElement) stopSmoothScroll() {
	if e.scrollAnimY != nil && e.scrollAnimY.IsRunning() {
		e.scrollAnimY.Stop()
	}
	if e.scrollAnimX != nil && e.scrollAnimX.IsRunning() {
		e.scrollAnimX.Stop()
	}
}

// restartAnimY 重启 Y 轴平滑动画（使用当前的 scrollFromY / scrollTargetY）。
func (e *ScrollViewElement) restartAnimY() {
	if e.scrollAnimY != nil {
		e.scrollAnimY.Stop()
	}
	e.scrollAnimY = animation.NewController(scrollAnimDuration, animation.EaseOutCubic)
	e.scrollAnimY.OnUpdate = func(v float64) {
		e.scrollOffset.Y = animation.LerpFloat(e.scrollFromY, e.scrollTargetY, v)
		if e.child != nil {
			e.child.SetPosition(types.Point{X: -e.scrollOffset.X, Y: -e.scrollOffset.Y})
		}
		e.fireScroll()
	}
	e.scrollAnimY.OnDone = func() {
		e.scrollOffset.Y = e.scrollTargetY
		if e.child != nil {
			e.child.SetPosition(types.Point{X: -e.scrollOffset.X, Y: -e.scrollOffset.Y})
		}
		e.fireScroll()
		if OnNeedsRepaint != nil {
			OnNeedsRepaint()
		}
	}
	e.scrollAnimY.Start()
}

// restartAnimX 重启 X 轴平滑动画（使用当前的 scrollFromX / scrollTargetX）。
func (e *ScrollViewElement) restartAnimX() {
	if e.scrollAnimX != nil {
		e.scrollAnimX.Stop()
	}
	e.scrollAnimX = animation.NewController(scrollAnimDuration, animation.EaseOutCubic)
	e.scrollAnimX.OnUpdate = func(v float64) {
		e.scrollOffset.X = animation.LerpFloat(e.scrollFromX, e.scrollTargetX, v)
		if e.child != nil {
			e.child.SetPosition(types.Point{X: -e.scrollOffset.X, Y: -e.scrollOffset.Y})
		}
		e.fireScroll()
	}
	e.scrollAnimX.OnDone = func() {
		e.scrollOffset.X = e.scrollTargetX
		if e.child != nil {
			e.child.SetPosition(types.Point{X: -e.scrollOffset.X, Y: -e.scrollOffset.Y})
		}
		e.fireScroll()
		if OnNeedsRepaint != nil {
			OnNeedsRepaint()
		}
	}
	e.scrollAnimX.Start()
}

// dragBarTo 按鼠标 Y 把竖向滑块拖到对应位置(轨道位移按比例换算成 scrollOffset)。
func (e *ScrollViewElement) dragBarTo(mouseY float64) {
	contentH := e.size.Height + e.maxScroll.Y
	thumbH := e.size.Height * e.size.Height / contentH
	if thumbH < 30 {
		thumbH = 30
	}
	travel := e.size.Height - thumbH
	if travel <= 0 {
		return
	}
	delta := (mouseY - e.dragStartMouseY) * e.maxScroll.Y / travel
	e.scrollOffset.Y = clamp(e.dragStartScrollY+delta, 0, e.maxScroll.Y)
	if e.child != nil {
		e.child.SetPosition(types.Point{X: -e.scrollOffset.X, Y: -e.scrollOffset.Y})
	}
	e.fireScroll()
	if OnNeedsRepaint != nil {
		OnNeedsRepaint()
	}
}

func clamp(v, min, max float64) float64 {
	if v < min {
		return min
	}
	if v > max {
		return max
	}
	return v
}

func max(a, b float64) float64 {
	if a > b {
		return a
	}
	return b
}
