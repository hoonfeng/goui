// Package app 提供了 goui 应用框架的入口和生命周期管理。
package app

import (
	"fmt"
	"log"
	"os"
	"runtime"
	"runtime/debug"
	"time"

	"github.com/user/goui/internal/animation"
	"github.com/user/goui/internal/canvas"
	"github.com/user/goui/internal/event"
	"github.com/user/goui/internal/i18n"
	_ "github.com/user/goui/internal/platform"
	"github.com/user/goui/internal/render"
	"github.com/user/goui/internal/types"
	"github.com/user/goui/internal/widget"
	"github.com/user/goui/internal/window"
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
	capturedElement  widget.Element    // 鼠标捕获的 Element（按下时捕获，释放时解除）
	capturedOnButton event.MouseButton // 捕获时的按键

	// ShortcutManager 快捷键管理器，全局快捷键优先于焦点 Widget 处理
	ShortcutManager *event.ShortcutManager

	// 拖拽状态追踪
	dragThreshold     float64        // 拖拽触发阈值（像素），MouseDown 后移动超过此值触发 DragStart
	dragStartPos      types.Point    // 拖拽起始坐标（用于计算阈值和 Delta）
	isDragging        bool           // 当前是否处于拖拽状态
	dragSourceElement widget.Element // 拖拽发起 Element（发送 DragEnd/Drop 的目标）

	// Ready 首帧渲染完成后的回调（在 mainLoop 的同一 goroutine 中调用）
	Ready func()

	// subWindows 附属窗口（多窗口），主循环每帧处理其事件与渲染。
	subWindows []*SubWindow
}

// NewApplication 创建新的应用实例
func NewApplication() *Application {
	return &Application{
		ShortcutManager: event.NewShortcutManager(),
		dragThreshold:   5.0, // 5 像素移动阈值触发拖拽
	}
}

// Config 应用配置
type Config struct {
	Title      string
	Width      int
	Height     int
	Resizable  bool
	Borderless bool    // 无边框窗口
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

	// 2. 尝试绑定 OpenGL 上下文（某些平台不需要 GL 也能工作）
	glAvailable := false
	if err := app.Window.MakeCurrent(); err != nil {
		log.Printf("goui: OpenGL context not available, using software rendering only: %v", err)
	} else {
		glAvailable = true
	}

	// 3. 创建 Canvas：SkiaCanvas（Skia 硬件加速 2D 渲染，基于 goskia CGO 绑定）。
	//    有 GL 上下文时用 GPU 模式——Skia 经 OpenGL 直接渲染到窗口 framebuffer，
	//    省去「位图回读 → image.RGBA → 纹理上传」与每帧 PNG 编解码；否则回退 raster。
	var skCanvas *canvas.SkiaCanvas
	if glAvailable {
		if gpu, e := canvas.NewSkiaCanvasGPU(config.Width, config.Height, 0); e == nil {
			skCanvas = gpu
			log.Println("goui: SkiaCanvas GPU 模式（Skia 直接渲染到窗口 framebuffer，无位图上传）")
		} else {
			log.Printf("goui: GPU SkiaCanvas 创建失败(%v)，降级软件 raster（窗口显示可能受限）", e)
			skCanvas = canvas.NewSkiaCanvas(config.Width, config.Height)
		}
	} else {
		skCanvas = canvas.NewSkiaCanvas(config.Width, config.Height)
		log.Println("goui: OpenGL 不可用，SkiaCanvas 用软件渲染")
	}
	app.Canvas = skCanvas

	// 4. 创建渲染管线
	app.Pipeline = render.NewPipeline(config.Width, config.Height, app.Canvas)

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

	// 状态改变（SetState）触发重新布局：buildTree 复用 Element 不丢运行时状态，
	// 使深层组件的配置变化能正确传播（如 AnimatedContainer 的目标值、文本变化等）。
	widget.OnNeedsLayout = func() {
		if app.Pipeline != nil {
			app.Pipeline.MarkNeedsLayout()
		}
	}

	// 浮层增删(Tooltip/Popover/Select 等显示或隐藏)需要重新布局以定位浮层。
	widget.OnOverlayChanged = func() {
		if app.Pipeline != nil {
			app.Pipeline.MarkNeedsLayout()
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

	// 9.1 释放 Skia GPU 资源——必须在各窗口 GL 上下文销毁前（此时 GL 仍有效）。
	// 否则 goskia 的 GPU surface/context finalizer 会在进程退出时访问已销毁的 GL 上下文而崩溃（exit 1）。
	app.releaseGPUResources()

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
	const frameInterval = 16 * time.Millisecond       // 动画帧率节流目标，约 60fps（流畅）
	const cursorIdleInterval = 120 * time.Millisecond // 仅光标闪烁（无动画）时的帧率，约 8fps —— 光标闪一下不必满帧重绘
	firstFrame := true
	for app.Running {
		frameStart := time.Now()
		// 1. 泵送所有平台消息（非阻塞，PeekMessage）
		if !app.Window.ProcessEvents() {
			app.Running = false
			break
		}

		// 1.2 推进动画时间线：更新所有活跃动画的值（OnUpdate 内通常会触发
		//     SetState/MarkNeedsRepaint），使动画逐帧呈现。
		animation.Tick(time.Now())

		// 1.5 确保布局已执行，使 HitTest 能正确命中 Element
		//    事件处理需要正确的布局信息（Offset/Size），若在第一帧时布局未执行，
		//    HitTest 会返回 nil，导致 hover/enter/leave 检测失效。
		//    这里先检查 Pipeline 是否需要布局，需要则立即执行不等待 Render。
		if app.Pipeline != nil {
			app.Pipeline.EnsureLayout()
		}

		// 2. 先处理待处理 UI 事件，确保输入得到即时响应
		//    将事件处理放在渲染之前，减少输入到视觉反馈的延迟
		//    同时处理事件可能触发 SetState → MarkNeedsRepaint，确保渲染能捕获最新状态
		app.processPendingEvents()

		// 2.5 事件可能触发了重新布局（弹出/关闭浮层、切换标签等改变结构的操作）。
		//     渲染前再确保一次布局，使新出现/变化的内容在「首帧」就处于正确位置；
		//     否则会先按旧/默认布局画一帧、下一帧才纠正 → 视觉上「跳一下」（模态打开/切标签尤其明显）。
		if app.Pipeline != nil {
			app.Pipeline.EnsureLayout()
		}

		// 3. 持续重绘：有焦点 Element（光标闪烁）或有活跃动画时标记需要重绘，
		//    确保基于时间推进的视觉变化（光标闪烁、动画插值）能持续呈现。
		//    Pipeline.Render 内部检查 needsRepaint，若不标记则跳过渲染。
		if (app.focusedElement != nil && app.focusedElement.IsFocused()) || animation.HasActive() {
			app.Pipeline.MarkNeedsRepaint()
		}

		// 4. 渲染帧（需要布局或重绘时才实际执行）
		//    Pipeline.Render 内部检查 needsRepaint/needsLayout，无变化时直接返回
		rendered := false
		if app.Pipeline != nil {
			if err := app.Pipeline.Render(); err != nil {
				log.Printf("goui: Render error: %v", err)
			}
			rendered = app.Pipeline.DidRender()
		}

		// 5. 显示渲染结果到窗口
		if rendered {
			// GPU 模式：Skia 已直接渲染到窗口 framebuffer（Pipeline.Render → Flush → FlushAndSubmit），
			// 直接交换缓冲显示。
			app.Window.SwapBuffers()

			// 首帧渲染完成后调用 Ready 回调
			if firstFrame {
				firstFrame = false
				if app.Ready != nil {
					app.Ready()
				}
			}

			// 渲染后用最新缓存的光标位置刷新 IME 候选窗口，
			// 确保候选紧贴当前拼音末尾（InputElement.Paint 中光标位置缓存刚更新，
			// 使用与渲染一致的 Skia 测量，避免候选漂移）。
			if app.focusedElement != nil && app.focusedElement.IsFocused() {
				app.updateIMECandidatePos()
			}
		} else {
			// 6. 空闲：无渲染需要时，若既无焦点也无动画，则阻塞等待下一条消息（省电）。
			//    用 WaitMessage 真正让线程休眠，避免空转 CPU/GPU。
			//    持续渲染模式（动画/光标闪烁）的帧间隔由下面的帧率节流统一处理。
			if !((app.focusedElement != nil && app.focusedElement.IsFocused()) || animation.HasActive() || len(app.subWindows) > 0) {
				app.Window.WaitMessage()
			}
		}

		// 5.5 处理并渲染所有附属窗口（多窗口）
		app.processSubWindows()

		// 帧率节流：持续渲染（动画/光标闪烁）时把帧间隔稳定到约 60fps，
		// 消除 busy-loop 占满 CPU 导致的抖动与卡顿，让动画更顺滑。
		if (app.focusedElement != nil && app.focusedElement.IsFocused()) || animation.HasActive() {
			app.Window.ProcessEvents() // 睡前再泵一次消息，降低输入延迟
			// 动画需 ~60fps 流畅；仅光标闪烁则大幅降帧（~8fps）——光标闪一下不必满帧重绘整个 UI，
			// 显著减少持续渲染期的每帧 CGO 分配/GC 与 CPU/GPU 占用，也压低内存峰值。
			fi := frameInterval
			if !animation.HasActive() {
				fi = cursorIdleInterval
			}
			if elapsed := time.Since(frameStart); elapsed < fi {
				time.Sleep(fi - elapsed)
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
