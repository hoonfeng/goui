// Package app 提供了 goui 应用框架的入口和生命周期管理。
package app

import (
	"fmt"
	"log"
	"os"
	"runtime"
	"runtime/debug"
	"time"

	"github.com/hoonfeng/goui/internal/i18n"
	"github.com/hoonfeng/goui/pkg/animation"
	"github.com/hoonfeng/goui/pkg/canvas"
	"github.com/hoonfeng/goui/pkg/event"
	_ "github.com/hoonfeng/goui/pkg/platform"
	"github.com/hoonfeng/goui/pkg/render"
	"github.com/hoonfeng/goui/pkg/types"
	"github.com/hoonfeng/goui/pkg/widget"
	"github.com/hoonfeng/goui/pkg/window"
)

// Application 是 goui 应用的基类
type Application struct {
	Window      window.Window
	Canvas      canvas.Canvas
	Pipeline    *render.Pipeline
	RootWidget  widget.Widget
	RootElement widget.Element
	Running     bool

	// 事件路由
	pendingEvents    []event.Event     // 待处理事件队列
	hoveredElement   widget.Element    // 当前鼠标悬停的 Element
	focusedElement   widget.Element    // 当前拥有焦点的 Element
	lastInputAt      time.Time         // 最近一次键盘/IME 输入时刻（用于编辑期间提速到满帧，详见主循环帧率节流）
	capturedElement  widget.Element    // 鼠标捕获的 Element（按下时捕获，释放时解除）
	capturedOnButton event.MouseButton // 捕获时的按键

	// IME 候选窗位置缓存：主循环每帧渲染后都会调用 updateIMECandidatePos，
	// 但 Win32 ClientToScreen + SetIMECandidatePos 不应在光标未移动时重复调用。
	lastIMECursorPosX float64
	lastIMECursorPosY float64

	// ShortcutManager 快捷键管理器，全局快捷键优先于焦点 Widget 处理
	ShortcutManager *event.ShortcutManager

	// 拖拽状态追踪
	dragThreshold     float64        // 拖拽触发阈值（像素），MouseDown 后移动超过此值触发 DragStart
	dragStartPos      types.Point    // 拖拽起始坐标（用于计算阈值和 Delta）
	isDragging        bool           // 当前是否处于拖拽状态
	dragSourceElement widget.Element // 拖拽发起 Element（发送 DragEnd/Drop 的目标）

	// 光标闪烁节流时间戳：主循环不再每帧标记重绘，而是每 500ms 触发一次光标闪烁重绘，
	// 大幅减少 idle 时的 CGO 边界穿越（runtime.cgocall），降低空转 CPU 占用。
	lastCursorBlink time.Time

	// isActive 窗口是否处于激活（前台）状态，由 win32 WM_ACTIVATE 事件更新。
	// 后台时跳过 SwapBuffers，避免 GPU 驱动阻塞导致主循环卡死。
	isActive bool

	// Ready 首帧渲染完成后的回调（在 mainLoop 的同一 goroutine 中调用）
	Ready func()

	// OnDataChange 【轮询模式】UI 线程每帧调用的数据变更检测回调。
	// 由外部设置（如 chat.ChatState），返回 true 表示数据已更新、需要重新布局渲染。
	// 适用于「Agent 只写数据、UI 自己读」架构：Agent goroutine 写 state 并递增版本号，
	// UI 线程每帧调用此函数检查版本号，只在变化时触发重建，避免事件驱动的卡顿。
	// 返回 true → 管线自动 MarkNeedsLayout → 走完整布局+渲染。
	OnDataChange func() bool

	// subWindows 附属窗口（多窗口），主循环每帧处理其事件与渲染。
	subWindows []*SubWindow
}

// NewApplication 创建新的应用实例
func NewApplication() *Application {
	return &Application{
		ShortcutManager: event.NewShortcutManager(),
		dragThreshold:   5.0,  // 5 像素移动阈值触发拖拽
		isActive:        true, // 初始默认前台激活
	}
}

// Config 应用配置
type Config struct {
	Title      string
	Width      int
	Height     int
	Resizable  bool
	Borderless bool
	BackgroundColor types.Color // 清屏底色，默认白色。深色主题设为此色避免 clear→paint 间闪烁    // 无边框窗口
	Opacity    float64 // 窗口整体不透明度 0~1（0 或 1=不透明；0<x<1=半透明）
}

// DefaultConfig 返回默认应用配置
func DefaultConfig() Config {
	return Config{
		Title:     "goui Application",
		Width:     1024,
		Height:    768,
		Resizable: true,
	}
}

// Run 启动应用主循环
func (app *Application) Run(config Config) error {
	// 锁定当前 goroutine 到操作系统线程，确保 OpenGL 上下文在同一线程中访问
	// 这是 Go + OpenGL 使用的关键要求：OpenGL 上下文与线程绑定，
	// 若 Go 运行时将 goroutine 迁移到其他线程，OpenGL 调用将静默失败。
	runtime.LockOSThread()

	// 全局 panic 捕获：防止未处理 panic 导致 exit status 2 无法排查
	defer func() {
		if r := recover(); r != nil {
			fmt.Fprintf(app.stderr(), "\n=== goui PANIC: %v ===\n", r)
			// 输出关键变量状态帮助定位根因
			fmt.Fprintf(app.stderr(), "--- Application state ---\n")
			if app.Window != nil {
				fmt.Fprintf(app.stderr(), "Window: %T\n", app.Window)
			} else {
				fmt.Fprintf(app.stderr(), "Window: nil\n")
			}
			if app.Canvas != nil {
				fmt.Fprintf(app.stderr(), "Canvas: %T, size=%.0fx%.0f\n", app.Canvas, app.Canvas.Width(), app.Canvas.Height())
			} else {
				fmt.Fprintf(app.stderr(), "Canvas: nil\n")
			}
			if app.RootWidget != nil {
				fmt.Fprintf(app.stderr(), "RootWidget: %T\n", app.RootWidget)
			} else {
				fmt.Fprintf(app.stderr(), "RootWidget: nil\n")
			}
			if app.RootElement != nil {
				fmt.Fprintf(app.stderr(), "RootElement: %T\n", app.RootElement)
			} else {
				fmt.Fprintf(app.stderr(), "RootElement: nil\n")
			}
			if app.Pipeline != nil {
				fmt.Fprintf(app.stderr(), "Pipeline: %T\n", app.Pipeline)
			} else {
				fmt.Fprintf(app.stderr(), "Pipeline: nil\n")
			}
			fmt.Fprintf(app.stderr(), "Running: %v\n", app.Running)
			fmt.Fprintf(app.stderr(), "--- Stack trace ---\n")
			debug.PrintStack()
			fmt.Fprintf(app.stderr(), "--- goui PANIC END (exit 1) ---\n")
		}
	}()

	// 1. 创建窗口
	winConfig := window.DefaultWindowConfig()
	winConfig.Title = config.Title
	winConfig.Width = config.Width
	winConfig.Height = config.Height
	winConfig.Resizable = config.Resizable
	winConfig.Borderless = config.Borderless
	winConfig.Opacity = config.Opacity

	if window.NewWindow == nil {
		return ErrNoPlatform
	}

	var err error
	app.Window, err = window.NewWindow(winConfig)
	if err != nil {
		return err
	}
	defer app.Window.Close()
	// GPU 资源释放必须在窗口 GL 上下文销毁（window.Close）之前执行。
	// defer 按 LIFO 顺序：此处 defer 先于上方的 defer Close 执行。
	defer func() {
		if sk, ok := app.Canvas.(*canvas.SkiaCanvas); ok {
			sk.Release()
		}
	}()

	// 2. 创建 Skia GPU 画布：经 OpenGL 直接渲染到窗口默认 framebuffer（FBO 0）。
	//    GPU 模式跨平台、性能好；窗口遮盖恢复后由应用层重绘驱动重置，不依赖 GPU FBO 持久。
	log.Println("goui: 使用 GPU 模式（Skia OpenGL 渲染 → wglSwapBuffers 屏幕输出）")
	if err := app.Window.MakeCurrent(); err != nil {
		return fmt.Errorf("goui: MakeCurrent failed: %w", err)
	}
	sk, err := canvas.NewSkiaCanvasGPU(config.Width, config.Height, 0)
	if err != nil {
		return fmt.Errorf("goui: NewSkiaCanvasGPU failed: %w", err)
	}
	app.Canvas = sk

	// 4. 创建渲染管线
	app.Pipeline = render.NewPipeline(config.Width, config.Height, app.Canvas)
	if config.BackgroundColor != (types.Color{}) {
		app.Pipeline.BackgroundColor = config.BackgroundColor
	}

	// 5. 初始化根控件
	if app.RootWidget != nil {
		// 用 OverlayHost 包裹根控件，使 Tooltip/Popover/Select 等浮层有统一渲染宿主。
		app.RootElement = widget.CreateElementFor(widget.NewOverlayHost(app.RootWidget))
		app.RootElement.Mount(nil, 0)
		app.Pipeline.SetRootElement(app.RootElement)
	}

	// 6. 设置全局重绘回调（将 SetState 的信号连接到 Pipeline）
	// 注意：这里只调用 MarkNeedsRepaint()，不调用 MarkNeedsLayout()。
	// 每次事件处理（鼠标点击、键盘输入、IME、悬停、焦点变化等）都会触发
	// OnNeedsRepaint，如果同时触发 MarkNeedsLayout() 会导致 PerformLayout()
	// 调用 buildTree() → 容器 Build() 重新创建子元素 → InputElement 等有状态
	// 组件的 focused/hovered/cursorPos/text 等运行时状态全部被重置。
	// 视觉状态变化（焦点、悬停、光标闪烁）只需要重绘，不需要重新布局。
	widget.OnNeedsRepaint = func() {
		if app.Pipeline != nil {
			app.Pipeline.MarkNeedsRepaint()
		}
	}

	// 状态改变（SetState）入队通知（队列机制，非直接 MarkNeedsLayout）：
	// SetState 只递增计数器，主循环每帧调用 ConsumePendingUpdates 统一消费一次。
	// 多次 SetState 压缩为一次布局重建，减少 60fps 流式事件下全树重建的频次。
	widget.OnNeedsLayout = func() {
		if app.Pipeline != nil {
			app.Pipeline.EnqueueUpdate()
		}
	}

	// 浮层增删(Tooltip/Popover/Select 等显示或隐藏)需要重新布局以定位浮层。
	widget.OnOverlayChanged = func() {
		if app.Pipeline != nil {
			app.Pipeline.MarkNeedsLayout()
			app.Pipeline.EnqueueUpdate()
		}
	}

	// 指针捕获：自绘滚动条等在 MouseDown 命中拖动区时把鼠标捕获抢到自己，
	// 否则滑块下方的内容元素(HitTest 命中者)会截走拖动的 Move/松手的 Up。
	widget.RequestPointerCapture = func(el widget.Element) {
		app.capturedElement = el
	}

	// 光标形状：桥接 widget 层的光标请求到平台层（按当前窗口句柄）。
	widget.SetCursorShape = func(shape int) {
		if window.SetCursor != nil && app.Window != nil {
			window.SetCursor(app.Window.NativeHandle(), shape)
		}
	}

	// 自绘标题栏拖动区按下 → 发起原生窗口拖动（无边框窗口）
	widget.OnWindowDrag = func() {
		app.DragWindow()
	}

	// 语言切换：强制全树重建并重新布局，使所有 i18n.T(...) 刷新为新语言。
	i18n.OnChange(func() {
		widget.BumpRebuild()
		if app.Pipeline != nil {
			app.Pipeline.MarkNeedsLayout()
		}
	})

	// 6.1 桥接平台剪贴板实现到 widget 层，供输入框复制/剪切/粘贴快捷键使用。
	if window.ClipboardRead != nil {
		widget.ClipboardRead = window.ClipboardRead
	}
	if window.ClipboardWrite != nil {
		widget.ClipboardWrite = window.ClipboardWrite
	}

	// 6.1.1 桥接“打开文件”对话框，供富文本插入图片等调用；调用时按当前窗口句柄定父窗口。
	if window.OpenFileDialog != nil {
		widget.OpenFileDialog = func(title, filter string) string {
			var hwnd uintptr
			if app.Window != nil {
				hwnd = app.Window.NativeHandle()
			}
			return window.OpenFileDialog(hwnd, title, filter)
		}
	}

	// 6.1.2 桥接“选择文件夹”对话框，供对话面板添加目录附件等调用。
	if window.OpenFolderDialog != nil {
		widget.OpenFolderDialog = func(title string) string {
			var hwnd uintptr
			if app.Window != nil {
				hwnd = app.Window.NativeHandle()
			}
			return window.OpenFolderDialog(hwnd, title)
		}
	}

	// 6.2 动画帧回调：动画注册/推进时标记需要重绘，驱动主循环连续出帧。
	animation.SetFrameCallback(func() {
		if app.Pipeline != nil {
			app.Pipeline.MarkNeedsRepaint()
		}
	})

	// 7. 设置事件处理
	app.setupEventHandlers()

	// 7.1 初始无文本焦点，禁用 IME，避免未聚焦输入框时输入法弹出候选
	app.updateIMEEnabled()

	// 8. 通知窗口就绪
	log.Println("goui: Window ready, entering main loop")

	// 9. 进入主循环
	app.Running = true
	app.mainLoop()

	// 清理：断开全局回调，避免 dangling 引用
	widget.OnNeedsRepaint = nil
	widget.OnOverlayChanged = nil
	widget.RequestPointerCapture = nil

	log.Println("goui: Run returning nil (exit code will be 0 from main)")
	return nil
}

// dispatchHoverChange 在悬停目标从 old 变为 newTarget 时，沿父链发送 MouseLeave/MouseEnter。
// 采用"子树进入/离开"语义：只有真正离开/进入某元素的子树时才通知它，
// 从而让 Tooltip/Popover 等包裹元素能感知 hover，且在子树内部移动时不误触发。
func dispatchHoverChange(old, newTarget widget.Element, x, y float64, mods event.ModifierKeys) {
	if old == newTarget {
		return
	}
	inNew := map[widget.Element]bool{}
	for e := newTarget; e != nil; e = e.Parent() {
		inNew[e] = true
	}
	// old 父链中不属于 new 子树的元素 → 离开
	for e := old; e != nil; e = e.Parent() {
		if !inNew[e] {
			e.HandleEvent(event.NewMouseEvent(event.TypeMouseLeave, x, y, event.ButtonNone, mods))
		}
	}
	inOld := map[widget.Element]bool{}
	for e := old; e != nil; e = e.Parent() {
		inOld[e] = true
	}
	// new 父链中不属于 old 子树的元素 → 进入（从外层到内层）
	var entered []widget.Element
	for e := newTarget; e != nil; e = e.Parent() {
		if !inOld[e] {
			entered = append(entered, e)
		}
	}
	for i := len(entered) - 1; i >= 0; i-- {
		entered[i].HandleEvent(event.NewMouseEvent(event.TypeMouseEnter, x, y, event.ButtonNone, mods))
	}
}

// stderr 返回标准错误输出 Writer，用于 panic 恢复中的日志
func (app *Application) stderr() *os.File {
	return os.Stderr
}

// SetRootWidget 设置根控件
func (app *Application) SetRootWidget(w widget.Widget) {
	app.RootWidget = w
}

// setupEventHandlers 设置事件处理器
func (app *Application) setupEventHandlers() {
	dispatcher := app.Window.EventDispatcher()

	// 窗口关闭事件
	dispatcher.AddEventListener(event.TypeWindowClose, func(e event.Event) {
		app.Running = false
	})

	// 窗口大小变化事件
	dispatcher.AddEventListener(event.TypeWindowResize, func(e event.Event) {
		if resizeEv, ok := e.(*event.ResizeEvent); ok {
			if app.Pipeline != nil {
				app.Pipeline.Resize(resizeEv.Width, resizeEv.Height) // SkiaCanvas（GPU）重建 FBO surface + 标记重新布局
			}
			// resize 拖动期间 Windows 进入系统模态消息循环、mainLoop 被阻塞，
			// 这里直接渲染一帧让内容跟手更新，而不是松手才刷新。
			app.renderFrame()
		}
	})

	// 窗口激活/失活事件：更新 isActive 状态。后台窗口时主循环跳过 SwapBuffers，
	// 避免 GPU 驱动在非前台窗口调用 SwapBuffers 阻塞数秒。
	dispatcher.AddEventListener(event.TypeWindowActivate, func(e event.Event) {
		app.isActive = true
		// 恢复前台：立即标记重新布局+重绘，使窗口内容即刻刷新
		if app.Pipeline != nil {
			app.Pipeline.MarkNeedsLayout()
		}
	})

	// 窗口失活：isActive=false，主循环后续不再调用 SwapBuffers
	dispatcher.AddEventListener(event.TypeWindowDeactivate, func(e event.Event) {
		app.isActive = false
	})

	// 重绘事件
	dispatcher.AddEventListener(event.TypeWindowPaint, func(e event.Event) {
		if app.Pipeline != nil {
			app.Pipeline.MarkNeedsRepaint()
		}
	})

	// ---- 鼠标事件路由（放入队列，在渲染管线中就绪后处理） ----

	dispatcher.AddEventListener(event.TypeMouseMove, func(e event.Event) {
		app.pendingEvents = append(app.pendingEvents, e)
	})
	dispatcher.AddEventListener(event.TypeMouseDown, func(e event.Event) {
		app.pendingEvents = append(app.pendingEvents, e)
	})
	dispatcher.AddEventListener(event.TypeMouseUp, func(e event.Event) {
		app.pendingEvents = append(app.pendingEvents, e)
	})
	dispatcher.AddEventListener(event.TypeMouseWheel, func(e event.Event) {
		app.pendingEvents = append(app.pendingEvents, e)
	})
	dispatcher.AddEventListener(event.TypeMouseEnter, func(e event.Event) {
		app.pendingEvents = append(app.pendingEvents, e)
	})
	dispatcher.AddEventListener(event.TypeMouseLeave, func(e event.Event) {
		app.pendingEvents = append(app.pendingEvents, e)
	})
	dispatcher.AddEventListener(event.TypeMouseDoubleClick, func(e event.Event) {
		app.pendingEvents = append(app.pendingEvents, e)
	})
	dispatcher.AddEventListener(event.TypeContextMenu, func(e event.Event) {
		app.pendingEvents = append(app.pendingEvents, e)
	})

	// ---- 窗口事件 ----
	dispatcher.AddEventListener(event.TypeWindowMove, func(e event.Event) {
		app.pendingEvents = append(app.pendingEvents, e)
	})

	// ---- 键盘事件路由（放入队列，在渲染管线中就绪后处理） ----

	dispatcher.AddEventListener(event.TypeKeyDown, func(e event.Event) {
		app.pendingEvents = append(app.pendingEvents, e)
	})
	dispatcher.AddEventListener(event.TypeKeyUp, func(e event.Event) {
		app.pendingEvents = append(app.pendingEvents, e)
	})
	dispatcher.AddEventListener(event.TypeKeyChar, func(e event.Event) {
		app.pendingEvents = append(app.pendingEvents, e)
	})

	// IME 组合（拼音预览）事件：必须入队，否则 processPendingEvents 中
	// 处理 IMECompositionEvent 的分支永远收不到事件，用户输入拼音时
	// 看不到正在组合的字符串（组合预览不渲染）。
	dispatcher.AddEventListener(event.TypeIMEComposition, func(e event.Event) {
		app.pendingEvents = append(app.pendingEvents, e)
	})
}

// mainLoop 主事件循环
func (app *Application) mainLoop() {
	log.Println("goui: Application started")
	const frameInterval60fps = 16 * time.Millisecond   // 动画/输入活跃时 60fps
	const frameIntervalCursor = 120 * time.Millisecond // 仅光标闪烁时 ~8fps
	const frameIntervalIdle = 200 * time.Millisecond   // 完全空闲时 5fps
	firstFrame := true

	frameIndex := 0
	for app.Running {
		frameIndex++
		frameStart := time.Now()
		// 1. 泵送所有平台消息（非阻塞，PeekMessage）
		if !app.Window.ProcessEvents() {
			app.Running = false
			break
		}

		// ★ 2. 先处理所有待处理 UI 事件（鼠标/键盘/IME）
		//    必须在 animation.Tick(drain→SetState→Rebuild) 之前执行，
		//    确保用户输入优先响应，不被 Agent 流式输出的全树重建阻塞。
		//    HitTest 使用前帧布局（当前树），对快速鼠标移动可能有 1 帧滞后，
		//    但在 60fps 下不可感知，且远优于输入被延迟数十帧的体验。
		app.processPendingEvents()

		// 1.2 推进动画时间线：更新所有活跃动画的值（OnUpdate 内通常会触发
		//     SetState/MarkNeedsRepaint），使动画逐帧呈现。
		animation.Tick(time.Now())

		// ★ 1.2.5 【轮询模式】UI 线程每帧检查外部数据是否变化。
		// Agent/后台 goroutine 只写数据（不调 SetState），UI 线程自己轮询读取。
		// 检测到数据变化时自动标记需要布局重建（走完整布局+渲染）。
		if app.Pipeline != nil && app.OnDataChange != nil && app.OnDataChange() {
			app.Pipeline.MarkNeedsLayout()
		}

		// 1.3 消费队列式状态更新通知：将本帧内所有 SetState 压缩为至多一次布局重建。
		// 动画 pump 中的 drain 可能多次调用 SetState（如 Agent 流式输出），
		// 但布局重建只执行一次，大幅降低 60fps 全树重建的开销。
		if app.Pipeline != nil {
			app.Pipeline.ConsumePendingUpdates()
		}

		// 1.5 确保布局已执行，使下一帧的 HitTest 能正确命中 Element
		if app.Pipeline != nil {
			app.Pipeline.EnsureLayout()
			// ★ 树重建后校验 focusedElement 是否仍在树中
			app.ensureValidFocus()
		}

		// 注：EnsureLayout() 只调一次（这里），不额外调用第二次。
		// 若事件处理触发了 SetState → MarkNeedsLayout()，flag 已设置，
		// Render() 中的 PerformLayout 会在同一帧内正确执行。
		// 每帧至多 1 次全树重建，避免两次 EnsureLayout 的冗余开销。

		// 3. 持续重绘标记
		// 优化：不再每帧为光标闪烁标记重绘（之前导致 runtime.cgocall 达 95%）。
		//   - 动画：每帧标记（流畅运行必需的 60fps）
		//   - 光标闪烁（有焦点、无动画）：每 500ms 标记一次（InputElement 的光标周期就是 500ms）
		//   - 无焦点无动画：不标记（Pipeline.Render 走早期返回，随后 WaitMessage 休眠）
		if animation.HasActive() {
			app.Pipeline.MarkNeedsRepaint() // 动画需要 60fps
		} else if app.focusedElement != nil && app.focusedElement.IsFocused() {
			// 光标闪烁：500ms 周期，由 InputElement.cursorVisible() 根据 focusTime + time.Now() 计算，
			// 不需要每帧重绘。每 500ms 触发一次即可让光标正常闪烁。
			if time.Since(app.lastCursorBlink) >= 500*time.Millisecond {
				app.Pipeline.MarkNeedsRepaint()
				app.lastCursorBlink = time.Now()
			}
		}

		// 4. 渲染帧
		rendered := false
		if app.Pipeline != nil {
			if err := app.Pipeline.Render(); err != nil {
				log.Printf("goui: Render error: %v", err)
			}

			rendered = app.Pipeline.DidRender()
		}

		// 5. 显示渲染结果到窗口（GPU 模式：wglSwapBuffers 交换 GL 前后缓冲区）
		if rendered {
			app.Window.SwapBuffers()

			if firstFrame {
				firstFrame = false
				if app.Ready != nil {
					app.Ready()
				}
			}

			if app.focusedElement != nil && app.focusedElement.IsFocused() {
				app.updateIMECandidatePos()
			}
		} else {
			// 检查是否有后台 goroutine（pty/AI/文件加载）标记了待渲染的工作，
			// 避免在 WaitMessage 中无限阻塞导致 UI 冻结直到用户移动鼠标。
			hasPendingWork := app.Pipeline != nil && (app.Pipeline.NeedsRepaint() || app.Pipeline.NeedsLayout())
			if !(hasPendingWork || (app.focusedElement != nil && app.focusedElement.IsFocused()) || animation.HasActive() || len(app.subWindows) > 0) {
				// 用短 Sleep 轮询替代 Win32 WaitMessage 无限阻塞：
				// 1. 后台 goroutine 标记重绘后，主循环在 ≤50ms 内响应渲染
				// 2. 窗口在后台且无工作时，20次/秒的轮询 CPU 占用可忽略
				// 3. 焦点恢复时 ProcessEvents 在循环顶部消费消息，无任何消息积压
				time.Sleep(50 * time.Millisecond)
			}
		}

		// 5.5 处理并渲染所有附属窗口
		app.processSubWindows()

		// 帧率节流（三级策略）
		//   1. 动画活跃 → 60fps（确保动画流畅）
		//   2. 用户输入后 600ms 内 → 60fps（确保快速响应）
		//   3. 仅光标闪烁 → ~8fps（减少空转 CPU）
		//   4. 完全空闲 → 5fps 轮询（兼顾响应与功耗）
		if animation.HasActive() {
			app.Window.ProcessEvents()
			if elapsed := time.Since(frameStart); elapsed < frameInterval60fps {
				time.Sleep(frameInterval60fps - elapsed)
			}
		} else if app.focusedElement != nil && app.focusedElement.IsFocused() && time.Since(app.lastInputAt) <= 600*time.Millisecond {
			// 输入后 600ms 内：60fps 确保快速响应
			app.Window.ProcessEvents()
			if elapsed := time.Since(frameStart); elapsed < frameInterval60fps {
				time.Sleep(frameInterval60fps - elapsed)
			}
		} else if app.focusedElement != nil && app.focusedElement.IsFocused() {
			// 仅有光标闪烁：~8fps
			app.Window.ProcessEvents()
			if elapsed := time.Since(frameStart); elapsed < frameIntervalCursor {
				time.Sleep(frameIntervalCursor - elapsed)
			}
		} else if !rendered {
			// 完全空闲（无焦点、无动画）：长休眠轮询
			if elapsed := time.Since(frameStart); elapsed < frameIntervalIdle {
				time.Sleep(frameIntervalIdle - elapsed)
			}
		}

	}
	log.Println("goui: Application stopped")
}

// renderFrame 立即渲染并显示一帧（供 resize 期间直接调用）。
// Windows 调整窗口大小时进入系统模态消息循环、mainLoop 被阻塞，
// 不在此直接渲染则窗口内容要等松手才刷新、且新增区域会留空。
func (app *Application) renderFrame() {
	if app.Pipeline == nil {
		return
	}
	if err := app.Pipeline.Render(); err != nil {
		log.Printf("goui: Render error: %v", err)
		return
	}
	if !app.Pipeline.DidRender() {
		return
	}
	app.Window.SwapBuffers()
}

// RequestFocus 请求将焦点设置到指定 Element
func (app *Application) RequestFocus(el widget.Element) {
	if app.focusedElement == el {
		return
	}
	// 使旧焦点失去焦点
	if app.focusedElement != nil {
		app.focusedElement.Blur()
	}
	// 设置新焦点
	app.focusedElement = el
	if el != nil {
		el.Focus()
	}
	// 仅在焦点为文本输入框时启用 IME，否则禁用，避免失焦后仍弹出候选
	app.updateIMEEnabled()
	// 更新 IME 候选窗口位置（如果焦点元素是输入框）
	app.updateIMECandidatePos()
}

// ClearFocus 清除焦点
func (app *Application) ClearFocus() {
	if app.focusedElement != nil {
		app.focusedElement.Blur()
		app.focusedElement = nil
	}
	// 焦点清空后禁用 IME，避免无文本焦点时仍弹出候选
	app.updateIMEEnabled()
}

// updateIMEEnabled 根据当前焦点是否为文本输入框，启用或禁用窗口 IME，
// 使焦点离开输入框后输入法不再接收按键、弹出候选。
func (app *Application) updateIMEEnabled() {
	if window.SetIMEEnabled == nil || app.Window == nil {
		return
	}
	hwnd := app.Window.NativeHandle()
	if hwnd == 0 {
		return
	}
	_, isInput := app.focusedElement.(imeCapable)
	window.SetIMEEnabled(hwnd, isInput)
}

// imeCapable 焦点元素若实现此接口（提供光标客户区坐标），即启用 IME 并把候选窗定位到光标处。
// InputElement、CodeEditorElement 均实现之。
type imeCapable interface {
	CursorClientPos() (x, y float64)
}

// updateIMECandidatePos 更新 IME 候选窗口位置到当前焦点输入框的光标处。
// 如果焦点元素不是 InputElement，则不操作。
// 内置节流：如果光标位置没有变化，跳过 Win32 API 调用。
func (app *Application) updateIMECandidatePos() {
	if app.focusedElement == nil {
		return
	}
	inputEl, ok := app.focusedElement.(imeCapable)
	if !ok {
		return
	}
	// 获取光标在客户区中的位置（像素坐标）
	cx, cy := inputEl.CursorClientPos()

	// 光标位置未变化 → 跳过 Win32 API 调用（主循环每帧都会调用此函数）
	if cx == app.lastIMECursorPosX && cy == app.lastIMECursorPosY {
		return
	}
	app.lastIMECursorPosX = cx
	app.lastIMECursorPosY = cy

	// 将客户区坐标转换为屏幕坐标（调用 Win32 ClientToScreen）
	// 通过 window 包提供的平台回调机制通知 win32 层设置候选窗口位置
	if window.SetIMECandidatePos != nil {
		// 先将 client 坐标转换为 int（四舍五入）
		sx := int(cx)
		sy := int(cy)

		// 获取窗口的客户区原点在屏幕上的位置
		// win32 的 SetIMECandidateWindow 使用屏幕坐标
		hwnd := app.Window.NativeHandle()
		if hwnd != 0 {
			// 使用 ClientToScreen 转换坐标
			// 但由于我们是跨平台的，让 win32 层处理坐标转换更方便
			window.SetIMECandidatePos(hwnd, sx, sy)
		}
	}
}

// processPendingEvents 处理所有待处理事件
func (app *Application) processPendingEvents() {
	events := app.pendingEvents
	app.pendingEvents = nil

	for _, ev := range events {
		switch e := ev.(type) {
		case *event.MouseEvent:
			app.routeMouseEvent(e)
		case *event.KeyEvent:
			app.lastInputAt = time.Now() // 编辑活动 → 主循环临时提速到满帧（长按退格/快速打字流畅）
			// 快捷键优先匹配：如果 ShortcutManager 匹配到快捷键，不再路由到焦点 Widget
			if e.Type() == event.TypeKeyDown && app.ShortcutManager != nil {
				if app.ShortcutManager.Match(e) {
					continue
				}
			}
			app.routeKeyEvent(e)
		case *event.DragEvent:
			// 拖拽事件直接冒泡到当前 hoveredElement
			if app.hoveredElement != nil {
				app.hoveredElement.HandleEvent(e)
			}
		case *event.IMECompositionEvent:
			app.lastInputAt = time.Now() // IME 输入活动 → 提速到满帧
			// IME 组合事件路由到焦点 Element
			if app.focusedElement != nil {
				app.focusedElement.HandleEvent(e)
				// IME 组合更新后立即刷新候选窗口位置，确保候选显示在光标处
				app.updateIMECandidatePos()
			}
		}
	}
}

// routeMouseEvent 路由鼠标事件到目标 Element
func (app *Application) routeMouseEvent(ev *event.MouseEvent) {
	if app.Pipeline == nil {
		return
	}
	defer app.applyCursor() // 事件处理后按 捕获/悬停 元素更新鼠标光标

	// ── 拖拽检测（捕获模式下，当 Element 支持拖拽时触发） ─────
	if app.capturedElement != nil {
		// 检查 capturedElement 是否支持拖拽
		isDraggable := false
		if d, ok := app.capturedElement.(widget.Draggable); ok {
			isDraggable = d.Draggable()
		}

		// MouseDown：记录拖拽起始坐标（重置拖拽状态）
		if ev.Type() == event.TypeMouseDown {
			app.dragStartPos = types.Point{X: ev.X, Y: ev.Y}
			app.isDragging = false
			app.dragSourceElement = nil
		}

		// MouseMove 且支持拖拽：检测是否超过拖拽阈值
		if ev.Type() == event.TypeMouseMove && isDraggable && !app.isDragging {
			dx := ev.X - app.dragStartPos.X
			dy := ev.Y - app.dragStartPos.Y
			if dx*dx+dy*dy > app.dragThreshold*app.dragThreshold {
				// 超过阈值，触发拖拽开始
				app.isDragging = true
				app.dragSourceElement = app.capturedElement
				dragEv := event.NewDragEvent(event.TypeDragStart, ev.X, ev.Y, ev.Button, ev.Mods)
				dragEv.DeltaX = dx
				dragEv.DeltaY = dy
				dragEv.Data = app.capturedElement
				app.capturedElement.HandleEvent(dragEv)
				// 拖拽开始后不再发送原始的 MouseMove
				return
			}
		}

		// 处于拖拽状态：将 MouseMove/MouseUp 转换为拖拽事件
		if app.isDragging {
			if ev.Type() == event.TypeMouseMove {
				dx := ev.X - app.dragStartPos.X
				dy := ev.Y - app.dragStartPos.Y
				app.dragStartPos = types.Point{X: ev.X, Y: ev.Y}

				// 发送 DragMove 到拖拽源
				dragEv := event.NewDragEvent(event.TypeDragMove, ev.X, ev.Y, ev.Button, ev.Mods)
				dragEv.DeltaX = dx
				dragEv.DeltaY = dy
				dragEv.Data = app.dragSourceElement
				if app.dragSourceElement != nil {
					app.dragSourceElement.HandleEvent(dragEv)
				}

				// 同时检测 DropTarget 悬停状态（DragEnter/DragLeave/DragOver）
				target := app.Pipeline.HitTest(ev.X, ev.Y)
				if target != app.hoveredElement {
					if app.hoveredElement != nil {
						leaveEv := event.NewDragEvent(event.TypeDragLeave, ev.X, ev.Y, event.ButtonNone, ev.Mods)
						leaveEv.Data = app.dragSourceElement
						app.hoveredElement.HandleEvent(leaveEv)
					}
					if target != nil {
						enterEv := event.NewDragEvent(event.TypeDragEnter, ev.X, ev.Y, event.ButtonNone, ev.Mods)
						enterEv.Data = app.dragSourceElement
						target.HandleEvent(enterEv)
					}
					app.hoveredElement = target
				} else if target != nil {
					overEv := event.NewDragEvent(event.TypeDragOver, ev.X, ev.Y, event.ButtonNone, ev.Mods)
					overEv.Data = app.dragSourceElement
					target.HandleEvent(overEv)
				}
				return
			}

			if ev.Type() == event.TypeMouseUp {
				// 发送 DragEnd 到拖拽源
				dragEv := event.NewDragEvent(event.TypeDragEnd, ev.X, ev.Y, ev.Button, ev.Mods)
				dragEv.DeltaX = ev.X - app.dragStartPos.X
				dragEv.DeltaY = ev.Y - app.dragStartPos.Y
				dragEv.Data = app.dragSourceElement
				if app.dragSourceElement != nil {
					app.dragSourceElement.HandleEvent(dragEv)
				}

				// 检测 DropTarget 并发送 Drop 事件
				target := app.Pipeline.HitTest(ev.X, ev.Y)
				if target != nil && target != app.dragSourceElement {
					dropEv := event.NewDragEvent(event.TypeDrop, ev.X, ev.Y, ev.Button, ev.Mods)
					dropEv.Data = app.dragSourceElement
					// 冒泡 Drop 事件
					current := target
					for current != nil {
						handled := current.HandleEvent(dropEv)
						if handled || dropEv.IsPropagationStopped() {
							break
						}
						current = current.Parent()
					}
				}

				// 清理拖拽状态
				app.isDragging = false
				app.dragSourceElement = nil
				app.capturedElement = nil
				return
			}
		}

		// ── 捕获模式（非拖拽）：原有逻辑 ──────────────
		// 捕获模式下，鼠标移动事件仍然更新 hover 状态（视觉反馈）
		// 但事件本身直接发送到捕获的元素
		if ev.Type() == event.TypeMouseMove {
			// 仍然检测 hover 变化（让按钮外的元素也能收到 enter/leave）
			target := app.Pipeline.HitTest(ev.X, ev.Y)
			if target != app.hoveredElement {
				dispatchHoverChange(app.hoveredElement, target, ev.X, ev.Y, ev.Mods)
				app.hoveredElement = target
			}
		}

		// MouseUp 释放捕获
		if ev.Type() == event.TypeMouseUp || ev.Type() == event.TypeMouseDown {
			// 沿父链冒泡（与非捕获路径一致）：捕获的可能是 Button 内的装饰子
			// （如「图标+文字」的 Div），点击/松手必须冒泡到祖先 Button 才能触发 OnClick。
			// 否则只有「捕获元素恰好就是 Button」（如标题栏单图标按钮借 HitTestIgnorer 穿透）才有效。
			for c := app.capturedElement; c != nil; c = c.Parent() {
				c.HandleEvent(ev)
				if ev.IsPropagationStopped() {
					break
				}
			}
			// MouseUp 时解除捕获（无论是否为目标元素）
			if ev.Type() == event.TypeMouseUp {
				app.capturedElement = nil
			}
			return
		}

		// 其他事件（MOVE、WHEEL 等）直接发给捕获的元素
		app.capturedElement.HandleEvent(ev)
		return
	}

	// ── 正常模式（无捕获）：执行命中测试 ──────────
	target := app.Pipeline.HitTest(ev.X, ev.Y)

	// 处理 MouseEnter/MouseLeave（基于 hover 状态变化）
	if ev.Type() == event.TypeMouseMove {
		if target != app.hoveredElement {
			dispatchHoverChange(app.hoveredElement, target, ev.X, ev.Y, ev.Mods)
			app.hoveredElement = target
		}
	}

	// 鼠标按下时：设置鼠标捕获 + 切换焦点
	if ev.Type() == event.TypeMouseDown {
		if target != nil {
			// 设置鼠标捕获，确保后续 MouseUp 即使发生在元素外也能送达
			app.capturedElement = target
			app.capturedOnButton = ev.Button
			app.RequestFocus(target)
			// 同时记录拖拽起始坐标
			app.dragStartPos = types.Point{X: ev.X, Y: ev.Y}
			app.isDragging = false
			app.dragSourceElement = nil
		} else {
			app.ClearFocus()
		}
	}

	// MouseUp 释放捕获（仅在非拖拽状态下，因为拖拽已在上面处理）
	if ev.Type() == event.TypeMouseUp {
		app.capturedElement = nil
	}

	// 双击也切换焦点（同单击行为）
	if ev.Type() == event.TypeMouseDoubleClick && target != nil {
		app.RequestFocus(target)
	}

	// 没有命中任何 Element，不继续分发
	if target == nil {
		return
	}

	// 对 ContextMenu 事件特殊处理：如果 HitTest 没有命中任何 Element，
	// 仍将事件发送到根 Element 或 hoveredElement
	if ev.Type() == event.TypeContextMenu && target == nil {
		if app.hoveredElement != nil {
			target = app.hoveredElement
		} else if app.RootElement != nil {
			target = app.RootElement
		}
	}

	// 事件冒泡：从目标 Element 沿父链冒泡到根（Web/DOM 语义）。
	// 不因某个元素"处理了"就停止——这样 Tooltip/Popover 等包裹元素、以及顶层
	// OverlayHost(点击浮层外关闭) 都能收到 MouseDown/Up，即使内部 Button 也消费了它。
	// 元素若要阻断冒泡，显式调用 ev.StopPropagation()。
	current := target
	for current != nil {
		current.HandleEvent(ev)
		if ev.IsPropagationStopped() {
			break
		}
		current = current.Parent()
	}

	// 鼠标按下或双击后，如果焦点是输入框，更新 IME 候选窗口位置
	if ev.Type() == event.TypeMouseDown || ev.Type() == event.TypeMouseDoubleClick {
		app.updateIMECandidatePos()
	}
}

// applyCursor 按当前 捕获/悬停 的 Element 更新窗口鼠标光标形状。
// 优先捕获元素（拖动中持续显示其光标），否则悬停元素；都不指定则默认箭头。
func (app *Application) applyCursor() {
	el := app.capturedElement
	if el == nil {
		el = app.hoveredElement
	}
	shape := widget.CursorDefault
	if cp, ok := el.(widget.CursorProvider); ok {
		shape = cp.Cursor()
	}
	if widget.SetCursorShape != nil {
		widget.SetCursorShape(shape)
	}
}

// routeKeyEvent 路由键盘事件到当前焦点 Element，并向上冒泡
// ensureValidFocus 校验 focusedElement 是否仍在 Element 树中。
// 树重建（buildTree）可能导致旧元素被 Unmount、新元素创建，
// 若 focusedElement 的父链无法追溯到 rootElement，则清除焦点。
func (app *Application) ensureValidFocus() {
	if app.focusedElement == nil || app.Pipeline == nil {
		return
	}
	root := app.Pipeline.RootElement()
	if root == nil {
		app.ClearFocus()
		return
	}
	// 沿父链上溯，看能否到达 rootElement
	found := false
	for el := app.focusedElement; el != nil; el = el.Parent() {
		if el == root {
			found = true
			break
		}
	}
	if !found {
		// 元素已不在树中 → 清除焦点
		app.focusedElement.Blur()
		app.focusedElement = nil
		app.updateIMEEnabled()
	}
}

func (app *Application) routeKeyEvent(ev *event.KeyEvent) {
	if app.focusedElement != nil {
		current := app.focusedElement
		for current != nil {
			handled := current.HandleEvent(ev)
			if handled || ev.IsPropagationStopped() {
				break
			}
			current = current.Parent()
		}

		// 键盘事件处理后更新 IME 候选窗口位置（光标可能已移动）
		if ev.Type() == event.TypeKeyDown || ev.Type() == event.TypeKeyChar {
			app.updateIMECandidatePos()
		}
	}
}

// Quit 退出应用
func (app *Application) Quit() {
	app.Running = false
}

// ErrNoPlatform 表示当前平台不支持
var ErrNoPlatform = &PlatformError{"当前平台不支持 goui"}

// PlatformError 平台错误
type PlatformError struct {
	Message string
}

func (e *PlatformError) Error() string {
	return e.Message
}
