package app

import (

	"github.com/hoonfeng/goui/pkg/canvas"
	"github.com/hoonfeng/goui/pkg/event"
	"github.com/hoonfeng/goui/pkg/render"
	"github.com/hoonfeng/goui/pkg/widget"
	"github.com/hoonfeng/goui/pkg/window"
)

// SubWindow 附属顶层窗口：拥有独立的渲染管线与根 Element，共享主线程消息循环。
// 适合工具面板、第二视图、调色板等场景。基础鼠标 hover/点击与键盘可用；
// 复杂的拖放、IME、跨窗口焦点仍以主窗口为主（诚实边界）。
type SubWindow struct {
	Window   window.Window
	skia     *canvas.SkiaCanvas
	pipeline *render.Pipeline

	pending    []event.Event
	hovered    widget.Element
	captured   widget.Element
	focused    widget.Element
	closed     bool
	firstFrame bool
}

// OpenWindow 创建一个附属窗口并渲染 root，纳入主循环。
// 须在主线程调用（Run 之后，通常在 Ready 回调或事件处理器内）。
func (app *Application) OpenWindow(config window.WindowConfig, root widget.Widget) (*SubWindow, error) {
	if window.NewWindow == nil {
		return nil, ErrNoPlatform
	}
	w, err := window.NewWindow(config)
	if err != nil {
		return nil, err
	}
	sw := &SubWindow{Window: w, firstFrame: true}
	// 副窗口 Raster 模式：CPU SkiaCanvas，GDI 输出，不依赖 OpenGL
	sw.skia = canvas.NewSkiaCanvas(config.Width, config.Height)
	sw.pipeline = render.NewPipeline(config.Width, config.Height, sw.skia)
	sw.pipeline.SetRootElement(widget.CreateElementFor(root))
	sw.setupEvents()

	app.subWindows = append(app.subWindows, sw)
	return sw, nil
}

// Close 关闭附属窗口（主循环下一轮回收）。
func (sw *SubWindow) Close() {
	sw.closed = true
	if sw.Window != nil {
		sw.Window.Close()
	}
}

// IsClosed 报告窗口是否已关闭。
func (sw *SubWindow) IsClosed() bool { return sw.closed }

// ── 副窗口的无边框 / 自绘标题栏操作（用法同主窗口的 app.* 方法）──

// DragWindow 发起本副窗口拖动（自绘标题栏拖动区按下时调用）。
func (sw *SubWindow) DragWindow() {
	if window.DragWindow != nil && sw.Window != nil {
		window.DragWindow(sw.Window.NativeHandle())
	}
}

// Minimize 最小化本副窗口。
func (sw *SubWindow) Minimize() {
	if window.MinimizeWindow != nil && sw.Window != nil {
		window.MinimizeWindow(sw.Window.NativeHandle())
	}
}

// ToggleMaximize 本副窗口最大化/还原切换。
func (sw *SubWindow) ToggleMaximize() {
	if window.ToggleMaximizeWindow != nil && sw.Window != nil {
		window.ToggleMaximizeWindow(sw.Window.NativeHandle())
	}
}

// IsMaximized 报告本副窗口是否最大化。
func (sw *SubWindow) IsMaximized() bool {
	if window.IsWindowMaximized != nil && sw.Window != nil {
		return window.IsWindowMaximized(sw.Window.NativeHandle())
	}
	return false
}

// SetTitleBar 声明本副窗口自绘标题栏命中区（顶部 height 高、右侧 rightExclude 宽除外），
// 由系统接管拖动 / 双击最大化 / Aero Snap。在 OpenWindow 返回后调用。
func (sw *SubWindow) SetTitleBar(height, rightExclude int) {
	if window.SetTitleBarHitArea != nil && sw.Window != nil {
		window.SetTitleBarHitArea(sw.Window.NativeHandle(), height, rightExclude)
	}
}

// EnableEffects 给本副窗口启用 DWM 投影阴影 + Win11 圆角。
func (sw *SubWindow) EnableEffects() {
	if window.EnableWindowShadow != nil && sw.Window != nil {
		window.EnableWindowShadow(sw.Window.NativeHandle())
	}
}

// SetIcon 设置本副窗口图标（任务栏 / Alt+Tab / 标题栏），iconPath 为 .ico 文件路径。
func (sw *SubWindow) SetIcon(iconPath string) {
	if window.SetWindowIcon != nil && sw.Window != nil {
		window.SetWindowIcon(sw.Window.NativeHandle(), iconPath)
	}
}

// setupEvents 监听副窗口事件分发器，把事件排入本窗口队列。
func (sw *SubWindow) setupEvents() {
	d := sw.Window.EventDispatcher()
	enqueue := func(e event.Event) { sw.pending = append(sw.pending, e) }
	for _, t := range []event.Type{
		event.TypeMouseMove, event.TypeMouseDown, event.TypeMouseUp,
		event.TypeMouseWheel, event.TypeKeyDown, event.TypeKeyUp,
	} {
		d.AddEventListener(t, enqueue)
	}
	d.AddEventListener(event.TypeWindowClose, func(e event.Event) {
		sw.closed = true
		sw.Window.Close()
	})
}

// processEvents 处理副窗口排队事件（简化路由：hover/捕获/冒泡）。
func (sw *SubWindow) processEvents() {
	if len(sw.pending) == 0 {
		return
	}
	evs := sw.pending
	sw.pending = nil
	sw.pipeline.EnsureLayout()
	for _, ev := range evs {
		switch e := ev.(type) {
		case *event.MouseEvent:
			sw.routeMouse(e)
		case *event.KeyEvent:
			recv := sw.focused
			if recv == nil {
				recv = sw.hovered
			}
			for cur := recv; cur != nil; cur = cur.Parent() {
				if cur.HandleEvent(e) || e.IsPropagationStopped() {
					break
				}
			}
		}
	}
	// 全局重绘 hook 指向主窗口 Pipeline，副窗口交互后须自行标记重绘。
	sw.pipeline.MarkNeedsRepaint()
}

// routeMouse 简化鼠标路由：HitTest → hover enter/leave（复用 dispatchHoverChange）→ 捕获/冒泡。
func (sw *SubWindow) routeMouse(ev *event.MouseEvent) {
	target := sw.pipeline.HitTest(ev.X, ev.Y)
	switch ev.Type() {
	case event.TypeMouseMove:
		if target != sw.hovered {
			dispatchHoverChange(sw.hovered, target, ev.X, ev.Y, ev.Mods)
			sw.hovered = target
		}
	case event.TypeMouseDown:
		sw.captured = target
		sw.focused = target
	}
	recv := target
	if sw.captured != nil && (ev.Type() == event.TypeMouseUp || ev.Type() == event.TypeMouseMove) {
		recv = sw.captured // 按下后捕获，确保 up/move 送达发起元素
	}
	for cur := recv; cur != nil; cur = cur.Parent() {
		cur.HandleEvent(ev)
		if ev.IsPropagationStopped() {
			break
		}
	}
	if ev.Type() == event.TypeMouseUp {
		sw.captured = nil
	}
}

// renderAndPresent 渲染并显示副窗口一帧（pipeline 内部 dirty 检查，无变化跳过）。
func (sw *SubWindow) renderAndPresent() {
	if sw.closed {
		return
	}
	sw.pipeline.EnsureLayout()
	if sw.firstFrame {
		sw.pipeline.MarkNeedsRepaint()
		sw.firstFrame = false
	}
	if err := sw.pipeline.Render(); err != nil {
		return
	}
	if !sw.pipeline.DidRender() {
		return
	}
	// Raster 模式：用 GDI 直接输出像素
	sw.Window.PresentImage(sw.skia.Image())
}

// 

// processSubWindows 在主循环中处理所有附属窗口的事件与渲染，并回收已关闭者。
func (app *Application) processSubWindows() {
	if len(app.subWindows) == 0 {
		return
	}
	live := app.subWindows[:0]
	for _, sw := range app.subWindows {
		if sw.closed {
			continue
		}
		sw.processEvents()
		sw.renderAndPresent()
		live = append(live, sw)
	}
	app.subWindows = live
	// 切回主窗口 GL 上下文，保证主循环后续主窗口渲染正确
	if app.Window != nil {
		app.Window.MakeCurrent()
	}
}
