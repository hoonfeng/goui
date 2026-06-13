//go:build windows

// Package win32 提供了 Windows 平台的原生窗口实现。
package win32

import (
	"log"
	"strings"
	"sync"
	"syscall"
	"unsafe"

	"github.com/hoonfeng/goui/pkg/event"
	"github.com/hoonfeng/goui/pkg/window"
)

// Win32 API DLLs
var (
	user32   = syscall.NewLazyDLL("user32.dll")
	gdi32    = syscall.NewLazyDLL("gdi32.dll")
	kernel32 = syscall.NewLazyDLL("kernel32.dll")
	opengl32 = syscall.NewLazyDLL("opengl32.dll")
	imm32    = syscall.NewLazyDLL("imm32.dll")
	comdlg32 = syscall.NewLazyDLL("comdlg32.dll")
	dwmapi   = syscall.NewLazyDLL("dwmapi.dll")
)

// Win32 API 函数
var (
	procCreateWindowExW  = user32.NewProc("CreateWindowExW")
	procDestroyWindow    = user32.NewProc("DestroyWindow")
	procShowWindow       = user32.NewProc("ShowWindow")
	procUpdateWindow     = user32.NewProc("UpdateWindow")
	procSetWindowTextW   = user32.NewProc("SetWindowTextW")
	procGetClientRect    = user32.NewProc("GetClientRect")
	procSetWindowPos      = user32.NewProc("SetWindowPos")
	procMonitorFromWindow = user32.NewProc("MonitorFromWindow")
	procGetMonitorInfoW   = user32.NewProc("GetMonitorInfoW")
	procDefWindowProcW   = user32.NewProc("DefWindowProcW")
	procRegisterClassExW = user32.NewProc("RegisterClassExW")
	procGetMessageW      = user32.NewProc("GetMessageW")
	procTranslateMessage = user32.NewProc("TranslateMessage")
	procDispatchMessageW = user32.NewProc("DispatchMessageW")
	procPostQuitMessage  = user32.NewProc("PostQuitMessage")
	procPeekMessageW     = user32.NewProc("PeekMessageW")
	procWaitMessage      = user32.NewProc("WaitMessage")
	procGetDC            = user32.NewProc("GetDC")
	procReleaseDC        = user32.NewProc("ReleaseDC")
	procLoadCursorW      = user32.NewProc("LoadCursorW")
	procLoadIconW        = user32.NewProc("LoadIconW")
	procAdjustWindowRect           = user32.NewProc("AdjustWindowRect")
	procSetLayeredWindowAttributes = user32.NewProc("SetLayeredWindowAttributes")
	procGetWindowLongW             = user32.NewProc("GetWindowLongW")
	procSetWindowLongW             = user32.NewProc("SetWindowLongW")
	procSendMessageW               = user32.NewProc("SendMessageW")
	procReleaseCapture             = user32.NewProc("ReleaseCapture")
	procSetCapture                 = user32.NewProc("SetCapture")
	procSetCursor                  = user32.NewProc("SetCursor")
	procIsZoomed                     = user32.NewProc("IsZoomed")
	procDwmExtendFrameIntoClientArea = dwmapi.NewProc("DwmExtendFrameIntoClientArea")
	procDwmSetWindowAttribute        = dwmapi.NewProc("DwmSetWindowAttribute")

	procSetPixelFormat    = gdi32.NewProc("SetPixelFormat")
	procChoosePixelFormat = gdi32.NewProc("ChoosePixelFormat")
	procSwapBuffers       = gdi32.NewProc("SwapBuffers")
	procValidateRect      = user32.NewProc("ValidateRect")
	procScreenToClient    = user32.NewProc("ScreenToClient")
	procClientToScreen    = user32.NewProc("ClientToScreen")
	procGetDpiForWindow   = user32.NewProc("GetDpiForWindow")
	procWglCreateContext  = opengl32.NewProc("wglCreateContext")
	procWglDeleteContext  = opengl32.NewProc("wglDeleteContext")
	procWglMakeCurrent    = opengl32.NewProc("wglMakeCurrent")
	// IMM32 输入法 API
	procImmGetContext            = imm32.NewProc("ImmGetContext")
	procImmReleaseContext        = imm32.NewProc("ImmReleaseContext")
	procImmGetCompositionStringW = imm32.NewProc("ImmGetCompositionStringW")
	procImmSetCompositionString  = imm32.NewProc("ImmSetCompositionString")
	procImmSetCandidateWindow    = imm32.NewProc("ImmSetCandidateWindow")
	procImmSetCompositionWindow  = imm32.NewProc("ImmSetCompositionWindow")
	procImmAssociateContext      = imm32.NewProc("ImmAssociateContext")
	procImmAssociateContextEx    = imm32.NewProc("ImmAssociateContextEx")
	// 剪贴板 API
	procOpenClipboard    = user32.NewProc("OpenClipboard")
	procCloseClipboard   = user32.NewProc("CloseClipboard")
	procEmptyClipboard   = user32.NewProc("EmptyClipboard")
	procGetClipboardData = user32.NewProc("GetClipboardData")
	procSetClipboardData = user32.NewProc("SetClipboardData")
	procGlobalAlloc      = kernel32.NewProc("GlobalAlloc")
	procGlobalLock       = kernel32.NewProc("GlobalLock")
	procGlobalUnlock     = kernel32.NewProc("GlobalUnlock")
	// 通用对话框 API（打开文件）
	procGetOpenFileNameW = comdlg32.NewProc("GetOpenFileNameW")
	procGlobalFree       = kernel32.NewProc("GlobalFree")
)

// Windows 常量
const (
	WS_OVERLAPPEDWINDOW = 0x00CF0000
	WS_VISIBLE          = 0x10000000
	WS_EX_LAYERED       = 0x00080000
	WS_EX_TOOLWINDOW    = 0x00000080
	LWA_ALPHA           = 0x02
	LWA_COLORKEY        = 0x01
	GWL_EXSTYLE         = 0xFFFFFFEC // -20（取窗口扩展样式）
	WS_POPUP            = 0x80000000
	WS_MINIMIZEBOX      = 0x00020000
	WS_CAPTION          = 0x00C00000
	SW_MINIMIZE         = 6
	SW_MAXIMIZE         = 3
	SW_RESTORE          = 9
	WM_NCCALCSIZE       = 0x0083
	WM_GETMINMAXINFO    = 0x0024
	WM_NCLBUTTONDOWN    = 0x00A1
	WM_NCHITTEST        = 0x0084
	WM_SETICON          = 0x0080
	HTCLIENT            = 1
	HTCAPTION           = 2
	ICON_SMALL          = 0
	ICON_BIG            = 1
	WS_EX_APPWINDOW     = 0x00040000
	WS_CLIPCHILDREN     = 0x02000000
	WS_CLIPSIBLINGS     = 0x04000000
	WS_SIZEBOX          = 0x00040000
	WS_MAXIMIZEBOX      = 0x00010000
	CS_OWNDC            = 0x0020
	CS_HREDRAW          = 0x0001
	CS_VREDRAW          = 0x0002
	CS_DBLCLKS          = 0x0008
	SW_SHOWNORMAL       = 1
	SW_HIDE             = 0
	PFD_TYPE_RGBA       = 0
	PFD_MAIN_PLANE      = 0
	PFD_DRAW_TO_WINDOW  = 4
	PFD_SUPPORT_OPENGL  = 32
	PFD_DOUBLEBUFFER    = 1

	WM_DESTROY              = 0x0002
	WM_MOVE                 = 0x0003
	WM_QUIT                 = 0x0012
	WM_SIZE                 = 0x0005
	WM_PAINT                = 0x000F
	WM_CLOSE                = 0x0010
	WM_ERASEBKGND           = 0x0014
	WM_SETCURSOR           = 0x0020
	WM_MOUSEMOVE            = 0x0200
	WM_LBUTTONDOWN          = 0x0201
	WM_LBUTTONUP            = 0x0202
	WM_LBUTTONDBLCLK        = 0x0203
	WM_RBUTTONDOWN          = 0x0204
	WM_RBUTTONUP            = 0x0205
	WM_RBUTTONDBLCLK        = 0x0206
	WM_MBUTTONDOWN          = 0x0207
	WM_MBUTTONUP            = 0x0208
	WM_MBUTTONDBLCLK        = 0x0209
	WM_MOUSEWHEEL           = 0x020A
	WM_MOUSEHOVER           = 0x02A1
	WM_DPICHANGED           = 0x02E0
	WM_KEYDOWN              = 0x0100
	WM_KEYUP                = 0x0101
	WM_CHAR                 = 0x0102
	WM_SYSKEYDOWN           = 0x0104
	WM_SYSKEYUP             = 0x0105
	WM_IME_STARTCOMPOSITION = 0x010D
	WM_IME_ENDCOMPOSITION   = 0x010E
	WM_IME_COMPOSITION      = 0x010F
	WM_IME_SETCONTEXT       = 0x0281
	WM_IME_NOTIFY           = 0x0282
)

// IME 组成字符串标志
const (
	GCS_COMPSTR   = 0x0008
	GCS_RESULTSTR = 0x0800
	GCS_CURSORPOS = 0x0080
)

// ISC_SHOWUICOMPOSITIONWINDOW 是 WM_IME_SETCONTEXT 中控制"系统绘制组合窗口"的位。
// 清除该位可阻止系统显示自己的拼音组合串（goui 在输入框内自绘组合预览）。
const ISC_SHOWUICOMPOSITIONWINDOW = 0x80000000

// IACE_DEFAULT 用于 ImmAssociateContextEx：恢复窗口默认的 IME 上下文。
const IACE_DEFAULT = 0x0010

// IME 候选窗口样式
const (
	CFS_DEFAULT = 0x0000
	CFS_POINT   = 0x0002
)

// 剪贴板格式与全局内存标志
const (
	CF_UNICODETEXT = 13
	GMEM_MOVEABLE  = 0x0002
)

// CANDIDATEFORM 用于 ImmSetCandidateWindow 设置候选窗口位置。
// 字段顺序与内存布局必须严格匹配 Win32 的 tagCANDIDATEFORM：
//   DWORD dwIndex; DWORD dwStyle; POINT ptCurrentPos; RECT rcArea;
// 此前缺少了 dwIndex 和 rcArea，导致 dwStyle 错位为 0（CFS_DEFAULT），
// IME 因此忽略 ptCurrentPos 并把候选窗口放在默认位置。
type CANDIDATEFORM struct {
	Index     uint32 // dwIndex：候选列表索引（通常为 0）
	Style     uint32 // dwStyle：CFS_POINT 等
	X, Y      int32  // ptCurrentPos：候选窗口左上角（客户区坐标）
	RcLeft    int32  // rcArea.left
	RcTop     int32  // rcArea.top
	RcRight   int32  // rcArea.right
	RcBottom  int32  // rcArea.bottom
}

// COMPOSITIONFORM 用于 ImmSetCompositionWindow 设置组合窗口位置。
// 现代 IME（微软拼音、搜狗等）通常依据组合窗口位置来定位候选/组合 UI，
// 仅设置 CANDIDATEFORM 往往不生效，因此两者都要设置。
//   DWORD dwStyle; POINT ptCurrentPos; RECT rcArea;
type COMPOSITIONFORM struct {
	Style    uint32 // dwStyle：CFS_POINT 等
	X, Y     int32  // ptCurrentPos：组合窗口位置（客户区坐标）
	RcLeft   int32
	RcTop    int32
	RcRight  int32
	RcBottom int32
}

// lastCandidatePos 缓存最近一次设置的候选窗口客户区坐标。
// 用于在 IME 开始组合时（WM_IME_STARTCOMPOSITION）立即设置位置，
// 避免候选窗口在默认位置（窗口左上角）闪一下才跳转到正确位置。
var lastCandidateX, lastCandidateY int32

// PIXELFORMATDESCRIPTOR 描述像素格式
type PIXELFORMATDESCRIPTOR struct {
	Size           uint16
	Version        uint16
	Flags          uint32
	PixelType      uint8
	ColorBits      uint8
	RedBits        uint8
	RedShift       uint8
	GreenBits      uint8
	GreenShift     uint8
	BlueBits       uint8
	BlueShift      uint8
	AlphaBits      uint8
	AlphaShift     uint8
	AccumBits      uint8
	AccumRedBits   uint8
	AccumGreenBits uint8
	AccumBlueBits  uint8
	AccumAlphaBits uint8
	DepthBits      uint8
	StencilBits    uint8
	AuxBuffers     uint8
	LayerType      uint8
	Reserved       uint8
	VisibleMask    uint32
	DamageMask     uint32
}

// RECT Win32 矩形结构
type RECT struct {
	Left, Top, Right, Bottom int32
}

// POINT Win32 点结构
type POINT struct {
	X, Y int32
}

// MONITORINFO 显示器信息（GetMonitorInfoW；RcWork 是不含任务栏的工作区）。
type MONITORINFO struct {
	CbSize    uint32
	RcMonitor RECT
	RcWork    RECT
	DwFlags   uint32
}

// MINMAXINFO 窗口最大化/最小尺寸信息（WM_GETMINMAXINFO 的 lParam）。
type MINMAXINFO struct {
	PtReserved     POINT
	PtMaxSize      POINT
	PtMaxPosition  POINT
	PtMinTrackSize POINT
	PtMaxTrackSize POINT
}

// MARGINS DWM 边框边距（DwmExtendFrameIntoClientArea）
type MARGINS struct {
	CxLeftWidth, CxRightWidth, CyTopHeight, CyBottomHeight int32
}

// MSG Win32 消息结构
type MSG struct {
	Hwnd    uintptr
	Message uint32
	WParam  uintptr
	LParam  uintptr
	Time    uint32
	Pt      struct{ X, Y int32 }
}

// WNDCLASSEXW 窗口类结构
type WNDCLASSEXW struct {
	Size       uint32
	Style      uint32
	WndProc    uintptr
	ClsExtra   int32
	WndExtra   int32
	Instance   uintptr
	Icon       uintptr
	Cursor     uintptr
	Background uintptr
	MenuName   *uint16
	ClassName  *uint16
	IconSm     uintptr
}

// windowMap 存储 hwnd 到 Win32Window 的映射（避免 unsafe.Pointer 在 Go + Windows API 中的 vet 警告）
var windowMap sync.Map

// primaryHwnd 是第一个创建的（主）窗口句柄。仅主窗口 WM_DESTROY 时 PostQuitMessage 结束应用，
// 附属窗口（多窗口）关闭不应退出整个进程。
var primaryHwnd uintptr

// Win32Window 实现 window.Window 接口
type Win32Window struct {
	hwnd        uintptr
	instance    uintptr
	hdc         uintptr
	glContext   uintptr
	title       string
	width       int
	height      int
	closed      bool
	borderless  bool // 无边框（自绘标题栏）：WM_NCCALCSIZE 移除系统标题栏
	captionH    int  // 自绘标题栏高度（client 顶部该范围 WM_NCHITTEST 返回 HTCAPTION → 系统处理拖动/双击最大化/Snap）
	captionR    int  // 标题栏右侧排除宽度（按钮区，返回 HTCLIENT 让 goui 处理点击）
	dispatcher  *event.Dispatcher
	vSync       bool
	pixelFormat int
	mu          sync.Mutex
}

// NewWindow 创建新的 Win32 窗口
func NewWindow(config window.WindowConfig) (window.Window, error) {
	instance, _, _ := kernel32.NewProc("GetModuleHandleW").Call(0)

	// 使用非游戏风格的窗口类名，避免被 NVIDIA GeForce Experience 等工具误识别为游戏
	className, _ := syscall.UTF16PtrFromString("goui_ui_frame_v1")

	// 注册窗口类（CS_OWNDC 确保每个窗口拥有专用 DC，避免 OpenGL 像素格式被覆盖）
	wc := WNDCLASSEXW{
		Size:      uint32(unsafe.Sizeof(WNDCLASSEXW{})),
		Style:     CS_OWNDC | CS_HREDRAW | CS_VREDRAW | CS_DBLCLKS,
		WndProc:   syscall.NewCallback(windowProc),
		Instance:  instance,
		ClassName: className,
	}

	cursor, _, _ := procLoadCursorW.Call(0, 32512) // IDC_ARROW
	wc.Cursor = cursor

	icon, _, _ := procLoadIconW.Call(0, 32512) // IDI_APPLICATION
	wc.Icon = icon

	ret, _, _ := procRegisterClassExW.Call(uintptr(unsafe.Pointer(&wc)))
	if ret == 0 {
		// 注册失败，可能是之前已注册过（程序异常退出后未清理），尝试获取已存在的类信息后忽略
		log.Printf("win32: RegisterClassExW failed (class may already exist), continuing...")
	}

	// 计算窗口尺寸
	windowRect := RECT{
		Left: 0, Top: 0,
		Right: int32(config.Width), Bottom: int32(config.Height),
	}
	windowStyle := uintptr(WS_OVERLAPPEDWINDOW | WS_VISIBLE | WS_CLIPCHILDREN | WS_CLIPSIBLINGS)
	if config.Borderless {
		// 无边框（隐藏系统标题栏，供自绘标题栏）：保留 thickframe(边缘 resize) 与 min/max box
		//（系统最小化/最大化动画、Aero Snap），经 WM_NCCALCSIZE 移除标题栏使客户区占满。
		windowStyle = WS_POPUP | WS_VISIBLE | WS_CLIPCHILDREN | WS_CLIPSIBLINGS |
			WS_SIZEBOX | WS_MINIMIZEBOX | WS_MAXIMIZEBOX
	}
	if !config.Resizable {
		windowStyle &^= WS_SIZEBOX
		windowStyle &^= WS_MAXIMIZEBOX
	}
	// 透明窗口：半透明时启用分层窗口扩展样式（创建后用 SetLayeredWindowAttributes 设 alpha）
	var exStyle uintptr
	if config.Opacity > 0 && config.Opacity < 1 {
		exStyle |= WS_EX_LAYERED
	}
	if config.Borderless {
		exStyle |= WS_EX_APPWINDOW // 无边框窗口默认无任务栏按钮，显式声明任务栏显示
	}
	procAdjustWindowRect.Call(uintptr(unsafe.Pointer(&windowRect)), windowStyle, 0)

	winWidth := int(windowRect.Right - windowRect.Left)
	winHeight := int(windowRect.Bottom - windowRect.Top)

	titleUTF16, _ := syscall.UTF16PtrFromString(config.Title)

	hwnd, _, _ := procCreateWindowExW.Call(
		exStyle,
		uintptr(unsafe.Pointer(className)),
		uintptr(unsafe.Pointer(titleUTF16)),
		windowStyle,
		0, 0,
		uintptr(winWidth),
		uintptr(winHeight),
		0, 0, instance, 0,
	)
	if hwnd == 0 {
		return nil, log.Output(2, "win32: failed to create window")
	}

	hdc, _, _ := procGetDC.Call(hwnd)

	win := &Win32Window{
		hwnd:       hwnd,
		instance:   instance,
		hdc:        hdc,
		title:      config.Title,
		width:      config.Width,
		height:     config.Height,
		vSync:      config.VSync,
		borderless: config.Borderless,
		dispatcher: event.NewDispatcher(),
	}

	if err := win.setupPixelFormat(); err != nil {
		return nil, err
	}
	// OpenGL 上下文延迟创建：首次需要时通过 MakeCurrent 或 SwapBuffers 创建
	// 这样做可以避免在窗口创建阶段就触发 NVIDIA 驱动的游戏检测逻辑

	// 存储窗口指针到全局映射表（避免 unsafe.Pointer）
	windowMap.Store(hwnd, win)
	if primaryHwnd == 0 {
		primaryHwnd = hwnd // 第一个创建的窗口为主窗口
	}

	// 透明窗口：设置整窗 alpha（创建时已挂 WS_EX_LAYERED）
	if config.Opacity > 0 && config.Opacity < 1 {
		setWindowOpacityWin32(hwnd, config.Opacity)
	}

	procShowWindow.Call(hwnd, SW_SHOWNORMAL)
	procUpdateWindow.Call(hwnd)

	return win, nil
}

func (w *Win32Window) setupPixelFormat() error {
	pfd := PIXELFORMATDESCRIPTOR{
		Size:        uint16(unsafe.Sizeof(PIXELFORMATDESCRIPTOR{})),
		Version:     1,
		Flags:       PFD_DRAW_TO_WINDOW | PFD_SUPPORT_OPENGL | PFD_DOUBLEBUFFER,
		PixelType:   PFD_TYPE_RGBA,
		ColorBits:   32,
		DepthBits:   24,
		StencilBits: 8,
		LayerType:   PFD_MAIN_PLANE,
	}

	pixelFormat, _, _ := procChoosePixelFormat.Call(w.hdc, uintptr(unsafe.Pointer(&pfd)))
	if pixelFormat == 0 {
		return log.Output(2, "win32: ChoosePixelFormat failed")
	}

	ret, _, _ := procSetPixelFormat.Call(w.hdc, pixelFormat, uintptr(unsafe.Pointer(&pfd)))
	if ret == 0 {
		return log.Output(2, "win32: SetPixelFormat failed")
	}

	w.pixelFormat = int(pixelFormat)
	return nil
}

func (w *Win32Window) createGLContext() error {
	glCtx, _, _ := procWglCreateContext.Call(w.hdc)
	if glCtx == 0 {
		return log.Output(2, "win32: wglCreateContext failed")
	}

	ret, _, _ := procWglMakeCurrent.Call(w.hdc, glCtx)
	if ret == 0 {
		return log.Output(2, "win32: wglMakeCurrent failed")
	}

	w.glContext = glCtx
	return nil
}

func (w *Win32Window) Title() string { return w.title }
func (w *Win32Window) SetTitle(title string) {
	w.title = title
	t, _ := syscall.UTF16PtrFromString(title)
	procSetWindowTextW.Call(w.hwnd, uintptr(unsafe.Pointer(t)))
}
func (w *Win32Window) Size() (int, int) { return w.width, w.height }
func (w *Win32Window) SetSize(width, height int) {
	w.width = width
	w.height = height
	procSetWindowPos.Call(w.hwnd, 0, 0, 0, uintptr(width), uintptr(height), 0x0004)
}
func (w *Win32Window) Position() (int, int) {
	r := RECT{}
	procGetClientRect.Call(w.hwnd, uintptr(unsafe.Pointer(&r)))
	return int(r.Left), int(r.Top)
}
func (w *Win32Window) SetPosition(x, y int) {
	procSetWindowPos.Call(w.hwnd, 0, uintptr(x), uintptr(y), 0, 0, 0x0005)
}
func (w *Win32Window) Show()                              { procShowWindow.Call(w.hwnd, SW_SHOWNORMAL) }
func (w *Win32Window) Hide()                              { procShowWindow.Call(w.hwnd, SW_HIDE) }
func (w *Win32Window) EventDispatcher() *event.Dispatcher { return w.dispatcher }
func (w *Win32Window) NativeHandle() uintptr              { return w.hwnd }
func (w *Win32Window) IsClosed() bool                     { return w.closed }
func (w *Win32Window) SetVSync(enabled bool)              { w.vSync = enabled }

func (w *Win32Window) Close() {
	w.mu.Lock()
	defer w.mu.Unlock()
	if !w.closed {
		w.closed = true
		if w.glContext != 0 {
			procWglMakeCurrent.Call(0, 0)
			procWglDeleteContext.Call(w.glContext)
			w.glContext = 0
		}
		if w.hdc != 0 {
			procReleaseDC.Call(w.hwnd, w.hdc)
			w.hdc = 0
		}
		hwnd := w.hwnd
		procDestroyWindow.Call(hwnd)
		w.hwnd = 0
		windowMap.Delete(hwnd)
	}
}

func (w *Win32Window) SwapBuffers() {
	if w.glContext == 0 {
		log.Println("goui: win32.SwapBuffers 跳过（无 GL 上下文）")
		return // 无 OpenGL 上下文，不执行交换
	}
	log.Println("goui: win32.SwapBuffers 调用 procSwapBuffers")
	procSwapBuffers.Call(w.hdc)
	log.Println("goui: win32.SwapBuffers 返回")
}

// ensureGLContext 延迟创建 OpenGL 上下文（首次使用时创建）
// 避免在窗口创建阶段就创建 GL 上下文，减少被 NVIDIA 驱动误检测为游戏的概率
func (w *Win32Window) ensureGLContext() error {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.glContext != 0 {
		return nil
	}
	return w.createGLContext()
}

func (w *Win32Window) MakeCurrent() error {
	if err := w.ensureGLContext(); err != nil {
		return err
	}
	ret, _, _ := procWglMakeCurrent.Call(w.hdc, w.glContext)
	if ret == 0 {
		return log.Output(2, "win32: wglMakeCurrent failed")
	}
	return nil
}

// WaitMessage 阻塞等待下一条 Windows 消息，避免忙等待空转 CPU
// 仅在无待处理消息时调用，配合 WaitMessage 可显著降低空闲时 CPU 占用率
func (w *Win32Window) WaitMessage() {
	log.Println("goui: win32.WaitMessage 调用（阻塞直到下一条 Windows 消息）")
	procWaitMessage.Call()
	log.Println("goui: win32.WaitMessage 返回")
}

// ProcessEvents 泵送 Windows 消息
func (w *Win32Window) ProcessEvents() bool {
	var msg MSG
	// PM_REMOVE = 1
	msgCount := 0
	for {
		ret, _, _ := procPeekMessageW.Call(
			uintptr(unsafe.Pointer(&msg)),
			0, 0, 0, 1,
		)
		if ret == 0 {
			break
		}
		msgCount++
		if msgCount == 1 {
			log.Printf("goui: win32.ProcessEvents 泵到第 1 条消息, msg=%d", msg.Message)
		}
		if msg.Message == WM_QUIT {
			quitExitCode := int(msg.WParam)
			log.Printf("win32: WM_QUIT received (exitCode=%d, hwnd=%x)", quitExitCode, msg.Hwnd)
			return false
		}
		procTranslateMessage.Call(uintptr(unsafe.Pointer(&msg)))
		procDispatchMessageW.Call(uintptr(unsafe.Pointer(&msg)))
	}
	if msgCount > 0 {
		log.Printf("goui: win32.ProcessEvents 完成, 共处理 %d 条消息", msgCount)
	}
	return !w.closed
}

// setWindowOpacityWin32 设置窗口整体不透明度（0~1，整窗 alpha 混合）。
// opacity<=0 或 >=1 视为不透明并关闭分层窗口；中间值启用 WS_EX_LAYERED + LWA_ALPHA。
func setWindowOpacityWin32(hwnd uintptr, opacity float64) {
	exStyle, _, _ := procGetWindowLongW.Call(hwnd, GWL_EXSTYLE)
	if opacity <= 0 || opacity >= 1 {
		procSetWindowLongW.Call(hwnd, GWL_EXSTYLE, exStyle&^WS_EX_LAYERED)
		return
	}
	if exStyle&WS_EX_LAYERED == 0 {
		procSetWindowLongW.Call(hwnd, GWL_EXSTYLE, exStyle|WS_EX_LAYERED)
	}
	alpha := uintptr(byte(opacity * 255))
	procSetLayeredWindowAttributes.Call(hwnd, 0, alpha, LWA_ALPHA)
}

// dragWindow 发起原生窗口拖动（自绘标题栏按住时调用）：先释放 goui 的鼠标捕获，
// 再用 WM_NCLBUTTONDOWN+HTCAPTION 让系统进入标准窗口移动循环（无边框也可拖）。
func dragWindow(hwnd uintptr) {
	procReleaseCapture.Call()
	procSendMessageW.Call(hwnd, WM_NCLBUTTONDOWN, HTCAPTION, 0)
}

// minimizeWindow 最小化窗口。
func minimizeWindow(hwnd uintptr) { procShowWindow.Call(hwnd, SW_MINIMIZE) }

// toggleMaximizeWindow 在最大化与还原之间切换。
func toggleMaximizeWindow(hwnd uintptr) {
	if isZoomedWin32(hwnd) {
		procShowWindow.Call(hwnd, SW_RESTORE)
	} else {
		procShowWindow.Call(hwnd, SW_MAXIMIZE)
	}
}

// isZoomedWin32 窗口是否处于最大化状态。
func isZoomedWin32(hwnd uintptr) bool {
	r, _, _ := procIsZoomed.Call(hwnd)
	return r != 0
}

// setTitleBarHitArea 声明自绘标题栏命中区：client 顶部 height 高、右侧 rightExclude 宽（按钮区）除外，
// 该范围在 WM_NCHITTEST 返回 HTCAPTION，交由系统处理拖动 / 双击最大化 / Aero Snap。
func setTitleBarHitArea(hwnd uintptr, height, rightExclude int) {
	if w, ok := windowMap.Load(hwnd); ok {
		win := w.(*Win32Window)
		win.captionH = height
		win.captionR = rightExclude
	}
}

// enableWindowShadowRound 给（无边框）窗口启用 DWM 投影阴影 + Win11 圆角。
func enableWindowShadowRound(hwnd uintptr) {
	// 阴影：扩展 1px DWM frame 到客户区，DWM 据此在窗口外绘制柔和投影
	m := MARGINS{CxLeftWidth: 1, CxRightWidth: 1, CyTopHeight: 1, CyBottomHeight: 1}
	procDwmExtendFrameIntoClientArea.Call(hwnd, uintptr(unsafe.Pointer(&m)))
	// Win11 圆角（DWMWA_WINDOW_CORNER_PREFERENCE=33, DWMWCP_ROUND=2）；旧系统忽略该属性，安全
	pref := int32(2)
	procDwmSetWindowAttribute.Call(hwnd, 33, uintptr(unsafe.Pointer(&pref)), 4)
}

// setWindowIconWin32 从 .ico 文件设置窗口图标：大图标用于任务栏/Alt+Tab，小图标用于标题栏左上。
func setWindowIconWin32(hwnd uintptr, iconPath string) {
	p, err := syscall.UTF16PtrFromString(iconPath)
	if err != nil {
		return
	}
	pp := uintptr(unsafe.Pointer(p))
	if hBig, _, _ := procLoadImageW.Call(0, pp, IMAGE_ICON, 32, 32, LR_LOADFROMFILE); hBig != 0 {
		procSendMessageW.Call(hwnd, WM_SETICON, ICON_BIG, hBig)
	}
	if hSmall, _, _ := procLoadImageW.Call(0, pp, IMAGE_ICON, 16, 16, LR_LOADFROMFILE); hSmall != 0 {
		procSendMessageW.Call(hwnd, WM_SETICON, ICON_SMALL, hSmall)
	}
}

// windowProc 是 Win32 窗口消息处理回调
func windowProc(hwnd uintptr, msg uint32, wParam, lParam uintptr) uintptr {
	// 从全局映射表获取 Win32Window 对象
	winObj, ok := windowMap.Load(hwnd)
	if !ok {
		return defWindowProc(hwnd, msg, wParam, lParam)
	}
	win := winObj.(*Win32Window)

	switch msg {
	case WM_TRAYICON:
		// 托盘回调：wParam=托盘 uID，lParam 低字=触发的鼠标消息（WM_LBUTTONUP/WM_RBUTTONUP）
		handleTrayMessage(uint32(wParam), uint32(lParam&0xFFFF))
		return 0

	case WM_GETMINMAXINFO:
		// 无边框窗口最大化默认覆盖整屏（盖住任务栏）。把最大尺寸/位置限制到监视器工作区
		// （rcWork 不含任务栏），使最大化不再遮挡任务栏。
		hMon, _, _ := procMonitorFromWindow.Call(hwnd, 2) // MONITOR_DEFAULTTONEAREST
		var mi MONITORINFO
		mi.CbSize = uint32(unsafe.Sizeof(mi))
		if r, _, _ := procGetMonitorInfoW.Call(hMon, uintptr(unsafe.Pointer(&mi))); r != 0 {
			mmi := (*MINMAXINFO)(unsafe.Pointer(lParam))
			mmi.PtMaxPosition.X = mi.RcWork.Left - mi.RcMonitor.Left
			mmi.PtMaxPosition.Y = mi.RcWork.Top - mi.RcMonitor.Top
			mmi.PtMaxSize.X = mi.RcWork.Right - mi.RcWork.Left
			mmi.PtMaxSize.Y = mi.RcWork.Bottom - mi.RcWork.Top
			return 0
		}

	case WM_NCCALCSIZE:
		// 无边框窗口：wParam=TRUE 时返回 0，让客户区占满整个窗口（移除系统标题栏/边框的非客户区）；
		// WS_SIZEBOX 的边缘 resize 命中仍由末尾 DefWindowProc 的 WM_NCHITTEST 处理。
		if win.borderless && wParam != 0 {
			return 0
		}

	case WM_NCHITTEST:
		// 无边框 + 已声明标题栏区：先让系统判边缘 resize（thickframe），非客户区命中优先；
		// 否则落在 client 顶部标题栏高度内、且不在右侧按钮排除区，则返回 HTCAPTION，
		// 由系统接管拖动 / 双击最大化 / Aero Snap（双击最大化因此天然支持）。
		if win.borderless && win.captionH > 0 {
			res := defWindowProc(hwnd, msg, wParam, lParam)
			var pt POINT
			pt.X = int32(int16(lParam & 0xFFFF))
			pt.Y = int32(int16((lParam >> 16) & 0xFFFF))
			procScreenToClient.Call(hwnd, uintptr(unsafe.Pointer(&pt)))
			inCaption := int(pt.Y) >= 0 && int(pt.Y) < win.captionH
			// 标题栏右侧按钮区：强制 HTCLIENT 交 goui 处理点击——必须优先于系统的“顶部 resize 边”
			// （WS_SIZEBOX 上边框在 high-DPI 下可达 10+px，会盖住贴顶按钮，使其点击变成拉伸窗口上边）。
			inButtons := inCaption && int(pt.X) >= win.width-win.captionR
			if inButtons {
				return HTCLIENT
			}
			// 其余：尊重系统边缘 resize 判定（左/右/下、标题栏左侧顶边仍可拉伸窗口）。
			if res != HTCLIENT {
				return res
			}
			// 标题栏内：逐点问 goui——命中可点控件(菜单/按钮)→HTCLIENT 交 goui，空白→HTCAPTION 系统拖动。
			if inCaption && window.TitleBarHitTest != nil && window.TitleBarHitTest(int(pt.X), int(pt.Y)) {
				return HTCLIENT
			}
			if inCaption {
				return HTCAPTION
			}
			return HTCLIENT
		}

	case WM_CLOSE:
		log.Printf("win32: >>> WM_CLOSE received (hwnd=%x)", hwnd)
		win.dispatcher.Dispatch(event.NewBaseEvent(event.TypeWindowClose))
		log.Printf("win32: <<< WM_CLOSE dispatched (hwnd=%x)", hwnd)
		return 0

	case WM_DESTROY:
		log.Printf("win32: >>> WM_DESTROY received (hwnd=%x)", hwnd)
		windowMap.Delete(hwnd)
		// 仅主窗口销毁时 PostQuitMessage 结束消息循环；
		// 附属窗口（多窗口）销毁只清理自身，不退出整个应用。
		if hwnd == primaryHwnd {
			procPostQuitMessage.Call(0)
		}
		return 0

	case WM_SIZE:
		width := int(lParam & 0xFFFF)
		height := int((lParam >> 16) & 0xFFFF)
		win.width = width
		win.height = height
		win.dispatcher.Dispatch(event.NewResizeEvent(width, height))
		return 0

	case WM_MOVE:
		// 窗口移动事件（可选处理，当前仅通知应用层）
		win.dispatcher.Dispatch(event.NewBaseEvent(event.TypeWindowMove))
		return 0

	case WM_PAINT:
		// 验证窗口无效区域，避免持续发送 WM_PAINT 消息
		procValidateRect.Call(hwnd, 0)
		// 通知应用层重绘
		win.dispatcher.Dispatch(event.NewBaseEvent(event.TypeWindowPaint))
		return 0

	case WM_ERASEBKGND:
		return 1

	case WM_SETCURSOR:
		// 客户区：应用 app 当前请求的光标（resize/hand/text 等），并阻止系统把光标重置为
		// 类光标（箭头）；非客户区（窗口边缘 resize 等）交系统默认（落到末尾 DefWindowProc）。
		if uint32(lParam&0xFFFF) == HTCLIENT && currentCursor != 0 {
			procSetCursor.Call(currentCursor)
			return 1
		}

	case WM_MOUSEMOVE:
		x := int16(lParam & 0xFFFF)
		y := int16((lParam >> 16) & 0xFFFF)
		ev := event.NewMouseEvent(event.TypeMouseMove, float64(x), float64(y), event.ButtonNone, getModifiers())
		win.dispatcher.Dispatch(ev)
		return 0

	case WM_LBUTTONDOWN:
		// 捕获鼠标：拖动（分隔条/滑块）或点击期间，即使光标移出元素、掠过非客户区、
		// 甚至移出窗口，也持续收到 WM_MOUSEMOVE / WM_LBUTTONUP，避免拖几像素就脱钩、
		// 或点击的 up 丢失导致 click 不触发（Win32 拖动/点击的标准要求）。
		procSetCapture.Call(hwnd)
		x := int16(lParam & 0xFFFF)
		y := int16((lParam >> 16) & 0xFFFF)
		ev := event.NewMouseEvent(event.TypeMouseDown, float64(x), float64(y), event.ButtonLeft, getModifiers())
		ev.Clicks = getClickCount(wParam)
		win.dispatcher.Dispatch(ev)
		return 0

	case WM_LBUTTONUP:
		procReleaseCapture.Call() // 释放鼠标捕获（与 WM_LBUTTONDOWN 的 SetCapture 配对）
		x := int16(lParam & 0xFFFF)
		y := int16((lParam >> 16) & 0xFFFF)
		ev := event.NewMouseEvent(event.TypeMouseUp, float64(x), float64(y), event.ButtonLeft, getModifiers())
		win.dispatcher.Dispatch(ev)
		return 0

	case WM_LBUTTONDBLCLK:
		x := int16(lParam & 0xFFFF)
		y := int16((lParam >> 16) & 0xFFFF)
		ev := event.NewMouseEvent(event.TypeMouseDoubleClick, float64(x), float64(y), event.ButtonLeft, getModifiers())
		ev.Clicks = 2
		win.dispatcher.Dispatch(ev)
		return 0

	case WM_RBUTTONDOWN:
		x := int16(lParam & 0xFFFF)
		y := int16((lParam >> 16) & 0xFFFF)
		ev := event.NewMouseEvent(event.TypeMouseDown, float64(x), float64(y), event.ButtonRight, getModifiers())
		win.dispatcher.Dispatch(ev)
		return 0

	case WM_RBUTTONUP:
		x := int16(lParam & 0xFFFF)
		y := int16((lParam >> 16) & 0xFFFF)
		// 先发送 MouseUp 事件
		upEv := event.NewMouseEvent(event.TypeMouseUp, float64(x), float64(y), event.ButtonRight, getModifiers())
		win.dispatcher.Dispatch(upEv)
		// 再发送 ContextMenu 事件（右键菜单）
		ctxEv := event.NewMouseEvent(event.TypeContextMenu, float64(x), float64(y), event.ButtonRight, getModifiers())
		win.dispatcher.Dispatch(ctxEv)
		return 0

	case WM_RBUTTONDBLCLK:
		x := int16(lParam & 0xFFFF)
		y := int16((lParam >> 16) & 0xFFFF)
		ev := event.NewMouseEvent(event.TypeMouseDoubleClick, float64(x), float64(y), event.ButtonRight, getModifiers())
		ev.Clicks = 2
		win.dispatcher.Dispatch(ev)
		return 0

	case WM_DPICHANGED:
		newDPI := float64(wParam & 0xFFFF)
		if newDPI < 96 {
			newDPI = 96
		}
		oldDPI := 96.0
		dpiRet, _, _ := procGetDpiForWindow.Call(hwnd)
		if dpiRet != 0 {
			oldDPI = float64(dpiRet)
		}
		dpiEv := event.NewDPIChangeEvent(oldDPI, newDPI)
		win.dispatcher.Dispatch(dpiEv)
		return 0

	case WM_MOUSEWHEEL:
		delta := int16(wParam >> 16)
		screenX := int16(lParam & 0xFFFF)
		screenY := int16((lParam >> 16) & 0xFFFF)
		pt := struct{ X, Y int32 }{X: int32(screenX), Y: int32(screenY)}
		procScreenToClient.Call(hwnd, uintptr(unsafe.Pointer(&pt)))
		ev := event.NewMouseEvent(event.TypeMouseWheel, float64(pt.X), float64(pt.Y), event.ButtonNone, getModifiers())
		ev.DeltaY = float64(delta) / 120.0
		win.dispatcher.Dispatch(ev)
		return 0

	case WM_KEYDOWN:
		vkCode := uint32(wParam)
		ev := event.NewKeyEvent(event.TypeKeyDown, vkCode, getKeyName(vkCode), getModifiers())
		ev.Repeat = (lParam & 0x40000000) != 0
		win.dispatcher.Dispatch(ev)
		return 0

	case WM_KEYUP:
		vkCode := uint32(wParam)
		ev := event.NewKeyEvent(event.TypeKeyUp, vkCode, getKeyName(vkCode), getModifiers())
		win.dispatcher.Dispatch(ev)
		return 0

	case WM_CHAR:
		ch := rune(wParam)
		ev := event.NewKeyEvent(event.TypeKeyChar, uint32(ch), string(ch), getModifiers())
		ev.Char = ch // 设置 Char 字段，确保 InputElement 能正确插入文字
		win.dispatcher.Dispatch(ev)
		return 0

	case WM_IME_STARTCOMPOSITION:
		// IME 开始组合。返回 0（不调用 DefWindowProc）以阻止系统绘制自己的组合
		// 窗口（拼音串）——goui 在输入框内自绘组合预览，若系统同时绘制会出现
		// “输入框下方重复的拼音预览”，且系统候选窗口跟随系统组合串末尾随输入
		// 不断右移漂移。候选词列表由 IME 独立显示，不受此返回值影响。
		// 先把候选窗口定位到缓存的光标位置，避免首帧闪现在默认位置。
		if lastCandidateX != 0 || lastCandidateY != 0 {
			setIMEWindowPos(hwnd, lastCandidateX, lastCandidateY)
		}
		return 0

	case WM_IME_COMPOSITION:
		// IME 组合过程中：获取组合字符串或结果字符串
		handleIMEComposition(win, hwnd, lParam)
		return 0

	case WM_IME_ENDCOMPOSITION:
		// IME 组合结束：发送空组合事件，清除输入框内的组合预览残留。
		win.dispatcher.Dispatch(event.NewIMECompositionEvent("", 0))
		return 0

	case WM_IME_SETCONTEXT:
		// 清除 ISC_SHOWUICOMPOSITIONWINDOW，阻止系统绘制组合窗口（拼音串自绘）；
		// 其余 IME UI（候选词列表）仍交由系统显示。
		lParam &^= ISC_SHOWUICOMPOSITIONWINDOW
		return defWindowProc(hwnd, msg, wParam, lParam)

	case WM_IME_NOTIFY:
		// IME 状态变化通知，需要时处理
		return 0

	}

	return defWindowProc(hwnd, msg, wParam, lParam)
}

func defWindowProc(hwnd uintptr, msg uint32, wParam, lParam uintptr) uintptr {
	ret, _, _ := procDefWindowProcW.Call(hwnd, uintptr(msg), wParam, lParam)
	return ret
}

func getModifiers() event.ModifierKeys {
	var mods event.ModifierKeys
	p := user32.NewProc("GetAsyncKeyState")
	if r, _, _ := p.Call(0x10); r&0x8000 != 0 {
		mods |= event.ModShift
	}
	if r, _, _ := p.Call(0x11); r&0x8000 != 0 {
		mods |= event.ModCtrl
	}
	if r, _, _ := p.Call(0x12); r&0x8000 != 0 {
		mods |= event.ModAlt
	}
	// VK_LWIN (0x5B) / VK_RWIN (0x5C) — Windows/Command 键
	if r, _, _ := p.Call(0x5B); r&0x8000 != 0 {
		mods |= event.ModSuper
	}
	if r, _, _ := p.Call(0x5C); r&0x8000 != 0 {
		mods |= event.ModSuper
	}
	return mods
}

func getClickCount(wParam uintptr) int {
	return int(wParam & 0xFFFF)
}

func getKeyName(vkCode uint32) string {
	names := map[uint32]string{
		0x08: "Backspace", 0x09: "Tab", 0x0D: "Enter",
		0x10: "Shift", 0x11: "Ctrl", 0x12: "Alt",
		0x1B: "Escape", 0x20: "Space", 0x25: "ArrowLeft",
		0x26: "ArrowUp", 0x27: "ArrowRight", 0x28: "ArrowDown",
		0x24: "Home", 0x23: "End", 0x21: "PageUp", 0x22: "PageDown",
		0x2E: "Delete", 0x2D: "Insert",
		0x30: "0", 0x31: "1", 0x32: "2", 0x33: "3", 0x34: "4",
		0x35: "5", 0x36: "6", 0x37: "7", 0x38: "8", 0x39: "9",
		0x41: "A", 0x42: "B", 0x43: "C", 0x44: "D", 0x45: "E",
		0x46: "F", 0x47: "G", 0x48: "H", 0x49: "I", 0x4A: "J",
		0x4B: "K", 0x4C: "L", 0x4D: "M", 0x4E: "N", 0x4F: "O",
		0x50: "P", 0x51: "Q", 0x52: "R", 0x53: "S", 0x54: "T",
		0x55: "U", 0x56: "V", 0x57: "W", 0x58: "X", 0x59: "Y", 0x5A: "Z",
		0x70: "F1", 0x72: "F2", 0x73: "F3", 0x74: "F4", 0x75: "F5",
		0x76: "F6", 0x77: "F7", 0x78: "F8", 0x79: "F9", 0x7A: "F10",
		0x7B: "F11", 0x7C: "F12",
	}
	if name, ok := names[vkCode]; ok {
		return name
	}
	return string(rune(vkCode))
}

// Init 初始化 Win32 平台支持
func Init() {
	window.NewWindow = NewWindow
}

// ─── IME 输入法辅助函数 ──────────────────────────────────

// handleIMEComposition 处理 IME 组合消息。
// 当 lParam 包含 GCS_RESULTSTR 时，表示输入法确认了最终字符。
// 当 lParam 包含 GCS_COMPSTR 时，表示输入法正在组合中（预览）。
func handleIMEComposition(win *Win32Window, hwnd uintptr, lParam uintptr) {
	if lParam&GCS_RESULTSTR != 0 {
		// 输入法确认了最终字符，获取结果字符串
		result := getIMECompositionString(hwnd, GCS_RESULTSTR)
		if result != "" {
			// 将结果字符串中的每个字符作为 TypeKeyChar 事件发送
			for _, ch := range result {
				ev := event.NewKeyEvent(event.TypeKeyChar, uint32(ch), string(ch), getModifiers())
				ev.Char = ch // 设置 Char 字段，确保 InputElement 能正确插入文字
				win.dispatcher.Dispatch(ev)
			}
			// 结果确认后，清除组合预览状态：发送空字符串的 IMECompositionEvent
			clearEv := event.NewIMECompositionEvent("", 0)
			win.dispatcher.Dispatch(clearEv)
		}
	}

	if lParam&GCS_COMPSTR != 0 {
		// 输入法正在组合，发送当前组合串给焦点 Element。
		// 即使组合串为空也要发送：退格把拼音删空时需要据此清除预览残留。
		compStr := getIMECompositionString(hwnd, GCS_COMPSTR)
		cursorPos := getIMECompositionCursorPos(hwnd)
		win.dispatcher.Dispatch(event.NewIMECompositionEvent(compStr, cursorPos))
	}
}

// getIMECompositionCursorPos 获取 IME 组合字符串中的光标位置
func getIMECompositionCursorPos(hwnd uintptr) int {
	himc, _, _ := procImmGetContext.Call(hwnd)
	if himc == 0 {
		return 0
	}
	defer procImmReleaseContext.Call(hwnd, himc)

	var pos uint32
	size, _, _ := procImmGetCompositionStringW.Call(himc, GCS_CURSORPOS, uintptr(unsafe.Pointer(&pos)), 4)
	if size <= 0 {
		return 0
	}
	return int(pos)
}

// SetIMECandidateWindow 设置 IME 组合/候选窗口的位置。
// x, y 为客户区坐标——CANDIDATEFORM/COMPOSITIONFORM 的 ptCurrentPos 按
// 客户区坐标解释，IME 会自行换算到屏幕，因此这里不能再做 ClientToScreen。
// 该函数对外公开，供 app 层在焦点/光标变化时调用。
func SetIMECandidateWindow(hwnd uintptr, x, y int) {
	// 缓存客户区坐标，供 WM_IME_STARTCOMPOSITION 中立即复用
	lastCandidateX = int32(x)
	lastCandidateY = int32(y)
	setIMEWindowPos(hwnd, int32(x), int32(y))
}

// setIMEWindowPos 使用客户区坐标设置 IME 组合窗口与候选窗口的位置。
// 同时设置 COMPOSITIONFORM 和 CANDIDATEFORM：不同 IME 的定位策略不同，
// 微软拼音/搜狗等多依据组合窗口定位，旧式 IME 才看候选窗口，两者都设最稳妥。
func setIMEWindowPos(hwnd uintptr, x, y int32) {
	himc, _, _ := procImmGetContext.Call(hwnd)
	if himc == 0 {
		return
	}
	defer procImmReleaseContext.Call(hwnd, himc)

	// 组合窗口位置（现代 IME 据此定位候选 UI）
	cform := COMPOSITIONFORM{
		Style: CFS_POINT,
		X:     x,
		Y:     y,
	}
	procImmSetCompositionWindow.Call(himc, uintptr(unsafe.Pointer(&cform)))

	// 候选窗口位置
	cand := CANDIDATEFORM{
		Index: 0,
		Style: CFS_POINT,
		X:     x,
		Y:     y,
	}
	procImmSetCandidateWindow.Call(himc, uintptr(unsafe.Pointer(&cand)))
}

// getIMECompositionString 从 IME 上下文中获取指定类型的组合字符串。
func getIMECompositionString(hwnd uintptr, format uint32) string {
	himc, _, _ := procImmGetContext.Call(hwnd)
	if himc == 0 {
		return ""
	}
	defer procImmReleaseContext.Call(hwnd, himc)

	// 先获取字符串大小
	size, _, _ := procImmGetCompositionStringW.Call(himc, uintptr(format), 0, 0)
	if size <= 0 {
		return ""
	}

	// 分配 UTF-16 缓冲区并获取字符串
	buf := make([]uint16, size/2+1)
	procImmGetCompositionStringW.Call(himc, uintptr(format), uintptr(unsafe.Pointer(&buf[0])), size)

	// 转换为 Go string
	return syscall.UTF16ToString(buf)
}

// clipboardReadWin32 读取系统剪贴板中的 Unicode 文本。
func clipboardReadWin32() string {
	if r, _, _ := procOpenClipboard.Call(0); r == 0 {
		return ""
	}
	defer procCloseClipboard.Call()

	h, _, _ := procGetClipboardData.Call(uintptr(CF_UNICODETEXT))
	if h == 0 {
		return ""
	}
	p, _, _ := procGlobalLock.Call(h)
	if p == 0 {
		return ""
	}
	defer procGlobalUnlock.Call(h)

	// 读取以 NUL 结尾的 UTF-16 字符串
	var u16 []uint16
	for i := uintptr(0); ; i++ {
		ch := *(*uint16)(unsafe.Pointer(p + i*2))
		if ch == 0 {
			break
		}
		u16 = append(u16, ch)
	}
	return syscall.UTF16ToString(u16)
}

// clipboardWriteWin32 将文本写入系统剪贴板（CF_UNICODETEXT）。
func clipboardWriteWin32(text string) {
	if r, _, _ := procOpenClipboard.Call(0); r == 0 {
		return
	}
	defer procCloseClipboard.Call()
	procEmptyClipboard.Call()

	u16, err := syscall.UTF16FromString(text) // 含末尾 NUL
	if err != nil {
		return
	}
	size := uintptr(len(u16) * 2)
	h, _, _ := procGlobalAlloc.Call(GMEM_MOVEABLE, size)
	if h == 0 {
		return
	}
	p, _, _ := procGlobalLock.Call(h)
	if p == 0 {
		procGlobalFree.Call(h)
		return
	}
	for i, ch := range u16 {
		*(*uint16)(unsafe.Pointer(p + uintptr(i)*2)) = ch
	}
	procGlobalUnlock.Call(h)

	// SetClipboardData 成功后，内存所有权转移给系统，不能再 GlobalFree；失败才释放。
	if r, _, _ := procSetClipboardData.Call(uintptr(CF_UNICODETEXT), h); r == 0 {
		procGlobalFree.Call(h)
	}
}

// setIMEEnabled 启用或禁用窗口的输入法（IME）。
// 禁用时解除 IME 上下文关联，窗口不再接收 IME 组合/候选，并取消进行中的组合；
// 启用时恢复默认 IME 上下文。用于仅在文本输入框聚焦时开启输入法。
func setIMEEnabled(hwnd uintptr, enabled bool) {
	if enabled {
		procImmAssociateContextEx.Call(hwnd, 0, IACE_DEFAULT)
	} else {
		procImmAssociateContext.Call(hwnd, 0)
	}
}

// openFileNameW 对应 Win32 OPENFILENAMEW 结构。64 位下 Go 编译器会在 lStructSize
// 之后自动插入 4 字节对齐填充，与 C 结构体布局一致，故 lStructSize 取 unsafe.Sizeof 即可。
type openFileNameW struct {
	lStructSize       uint32
	hwndOwner         uintptr
	hInstance         uintptr
	lpstrFilter       *uint16
	lpstrCustomFilter *uint16
	nMaxCustFilter    uint32
	nFilterIndex      uint32
	lpstrFile         *uint16
	nMaxFile          uint32
	lpstrFileTitle    *uint16
	nMaxFileTitle     uint32
	lpstrInitialDir   *uint16
	lpstrTitle        *uint16
	flags             uint32
	nFileOffset       uint16
	nFileExtension    uint16
	lpstrDefExt       *uint16
	lCustData         uintptr
	lpfnHook          uintptr
	lpTemplateName    *uint16
	pvReserved        uintptr
	dwReserved        uint32
	flagsEx           uint32
}

const (
	ofnExplorer      = 0x00080000
	ofnFileMustExist = 0x00001000
	ofnPathMustExist = 0x00000800
	ofnNoChangeDir   = 0x00000008
)

// buildFilterUTF16 把 "描述|模式" 多组（换行分隔）转换为 Win32 过滤器需要的格式：
// 各段以 NUL 分隔、整体以双 NUL 结尾的 UTF-16 序列。
func buildFilterUTF16(filter string) []uint16 {
	var out []uint16
	push := func(s string) {
		if u, err := syscall.UTF16FromString(s); err == nil {
			out = append(out, u...) // u 已含末尾 NUL，即为段分隔符
		} else {
			out = append(out, 0)
		}
	}
	if filter == "" {
		filter = "所有文件|*.*"
	}
	for _, line := range strings.Split(filter, "\n") {
		parts := strings.SplitN(line, "|", 2)
		pat := "*.*"
		if len(parts) == 2 {
			pat = parts[1]
		}
		push(parts[0])
		push(pat)
	}
	out = append(out, 0) // 终止双 NUL（前一段已补一个 NUL）
	return out
}

// openFileDialogWin32 弹出系统“打开文件”模态对话框，返回所选绝对路径；取消/出错返回空串。
func openFileDialogWin32(hwnd uintptr, title, filter string) string {
	filterU16 := buildFilterUTF16(filter)
	fileBuf := make([]uint16, 1024) // 接收所选路径（足够长）
	var titlePtr *uint16
	if title != "" {
		if t, err := syscall.UTF16PtrFromString(title); err == nil {
			titlePtr = t
		}
	}
	ofn := openFileNameW{
		hwndOwner:    hwnd,
		lpstrFilter:  &filterU16[0],
		nFilterIndex: 1,
		lpstrFile:    &fileBuf[0],
		nMaxFile:     uint32(len(fileBuf)),
		lpstrTitle:   titlePtr,
		flags:        ofnExplorer | ofnFileMustExist | ofnPathMustExist | ofnNoChangeDir,
	}
	ofn.lStructSize = uint32(unsafe.Sizeof(ofn))
	r, _, _ := procGetOpenFileNameW.Call(uintptr(unsafe.Pointer(&ofn)))
	if r == 0 {
		return "" // 用户取消或调用失败
	}
	return syscall.UTF16ToString(fileBuf)
}

// ─── 鼠标光标 ──────────────────────────────────────────────

var (
	curArrow         uintptr
	curSizeWE        uintptr
	curSizeNS        uintptr
	curHand          uintptr
	curIBeam         uintptr
	curCursorsLoaded bool
	currentCursor    uintptr // 应用当前请求的光标（WM_SETCURSOR 持续应用，防系统重置为类光标）
)

func loadCursors() {
	if curCursorsLoaded {
		return
	}
	curArrow, _, _ = procLoadCursorW.Call(0, 32512)  // IDC_ARROW
	curSizeWE, _, _ = procLoadCursorW.Call(0, 32644) // IDC_SIZEWE ↔
	curSizeNS, _, _ = procLoadCursorW.Call(0, 32645) // IDC_SIZENS ↕
	curHand, _, _ = procLoadCursorW.Call(0, 32649)   // IDC_HAND 手型
	curIBeam, _, _ = procLoadCursorW.Call(0, 32513)  // IDC_IBEAM 文本 I 形
	currentCursor = curArrow
	curCursorsLoaded = true
}

// setCursorWin32 记录并立即应用光标形状；WM_SETCURSOR 会持续维持，避免被系统重置。
func setCursorWin32(hwnd uintptr, shape int) {
	loadCursors()
	c := curArrow
	switch shape {
	case window.CursorResizeH:
		c = curSizeWE
	case window.CursorResizeV:
		c = curSizeNS
	case window.CursorHand:
		c = curHand
	case window.CursorText:
		c = curIBeam
	}
	if c == currentCursor {
		return
	}
	currentCursor = c
	procSetCursor.Call(c)
}

func init() {
	Init()
	// 注册 IME 候选窗口位置设置回调，供 app 层调用
	window.SetIMECandidatePos = func(hwnd uintptr, x, y int) {
		SetIMECandidateWindow(hwnd, x, y)
	}
	// 注册剪贴板读写实现，供 app 桥接到 widget 层
	window.ClipboardRead = clipboardReadWin32
	window.ClipboardWrite = clipboardWriteWin32
	// 注册 IME 启用/禁用，供 app 在焦点变化时调用
	window.SetIMEEnabled = setIMEEnabled
	// 注册“打开文件”对话框，供富文本插入图片等调用
	window.OpenFileDialog = openFileDialogWin32
	// 注册窗口透明度设置（整窗 alpha）
	window.SetWindowOpacity = setWindowOpacityWin32
	// 注册无边框窗口操作（自绘标题栏：拖动/最小化/最大化/还原）
	window.DragWindow = dragWindow
	window.MinimizeWindow = minimizeWindow
	window.ToggleMaximizeWindow = toggleMaximizeWindow
	window.IsWindowMaximized = isZoomedWin32
	window.SetTitleBarHitArea = setTitleBarHitArea
	window.EnableWindowShadow = enableWindowShadowRound
	window.SetWindowIcon = setWindowIconWin32
	window.SetCursor = setCursorWin32
}
