// Package window 提供了跨平台的窗口抽象层。
package window

import (
	"image"

	"github.com/hoonfeng/goui/pkg/event"
)

// Window 是抽象窗口接口，各平台实现此接口。
type Window interface {
	// Title 获取/设置窗口标题
	Title() string
	SetTitle(title string)

	// Size 获取/设置窗口尺寸
	Size() (width, height int)
	SetSize(width, height int)

	// Position 获取/设置窗口位置
	Position() (x, y int)
	SetPosition(x, y int)

	// Show/Hide 显示/隐藏窗口
	Show()
	Hide()

	// Close 关闭窗口
	Close()

	// IsClosed 检查窗口是否已关闭
	IsClosed() bool

	// EventDispatcher 返回事件分发器
	EventDispatcher() *event.Dispatcher

	// NativeHandle 返回原生窗口句柄（平台相关）
	NativeHandle() uintptr

	// VSync 启用/禁用垂直同步
	SetVSync(enabled bool)

	// SwapBuffers 交换前后缓冲区
	SwapBuffers()

	// MakeCurrent 使 OpenGL 上下文当前
	MakeCurrent() error

	// ProcessEvents 处理待处理的平台消息。
	// 返回 false 表示窗口已关闭或应用应退出。
	ProcessEvents() bool

	// WaitMessage 阻塞等待下一条消息到达，用于空闲时让出 CPU。
	// 仅在没有任何待处理消息和渲染需要时调用，避免忙等待空转。
	WaitMessage()

	// PresentImage 将 RGBA 图像直接显示到窗口客户区（Raster 模式）。
	// 替代 SwapBuffers，使用 GDI BitBlt/SetDIBitsToDevice 直接将像素数据输出到屏幕。
	// img 必须是 Image() 返回的 image.RGBA，其尺寸应与窗口客户区一致。
	PresentImage(img *image.RGBA)
}

// WindowConfig 窗口创建配置
type WindowConfig struct {
	Title      string
	Width      int
	Height     int
	X          int
	Y          int
	Resizable  bool
	Fullscreen bool
	Borderless bool
	VSync      bool
	Opacity    float64 // 窗口整体不透明度 0~1（0 或 1=不透明；0<x<1=半透明，启用分层窗口）
}

// DefaultWindowConfig 返回默认窗口配置
func DefaultWindowConfig() WindowConfig {
	return WindowConfig{
		Title:      "goui Application",
		Width:      1024,
		Height:     768,
		X:          -1, // 居中
		Y:          -1,
		Resizable:  true,
		Fullscreen: false,
		Borderless: false,
		VSync:      true,
	}
}

// NewWindow 创建窗口（平台相关的工厂方法）
var NewWindow func(config WindowConfig) (Window, error)

// SetIMECandidatePos 设置 IME 候选窗口位置（平台相关，由具体平台实现设置）。
// hwnd: 窗口句柄; x, y: 屏幕坐标（通常是输入框光标位置）。
// 在调用前请确保平台层已注册此回调（如 win32 在 init 中设置）。
var SetIMECandidatePos func(hwnd uintptr, x, y int)

// ClipboardRead/ClipboardWrite 提供系统剪贴板读写，由平台层实现并在 init 中设置。
// 未实现剪贴板的平台保持为 nil。
var (
	ClipboardRead  func() string
	ClipboardWrite func(text string)
)

// SetIMEEnabled 启用/禁用窗口输入法（IME），由平台层实现。
// 仅在文本输入框聚焦时启用，焦点离开后禁用，避免失焦仍弹出候选。
var SetIMEEnabled func(hwnd uintptr, enabled bool)

// OpenFileDialog 弹出系统"打开文件"对话框（模态），由平台层实现并在 init 中设置。
// hwnd: 父窗口句柄（0 表示无属主）; title: 对话框标题; filter: 过滤器描述
// （形如 "图片|*.png;*.jpg" 的竖线分隔对，多组用换行分隔）。
// 返回用户选择的绝对路径；取消或未实现返回空串。
var OpenFileDialog func(hwnd uintptr, title, filter string) string

// OpenFolderDialog 弹出系统"选择文件夹"对话框（模态），由平台层实现并在 init 中设置。
// hwnd: 父窗口句柄（0 表示无属主）; title: 对话框标题。
// 返回用户选择的目录绝对路径；取消或未实现返回空串。
var OpenFolderDialog func(hwnd uintptr, title string) string

// SetWindowOpacity 设置窗口整体不透明度（0~1，整窗 alpha 混合），由平台层实现。
// opacity<=0 或 >=1 视为不透明（关闭分层）。
var SetWindowOpacity func(hwnd uintptr, opacity float64)

// ── 系统托盘 ──────────────────────────────────────────

// TrayMenuItem 托盘右键菜单项。Separator=true 时为分隔线（忽略其它字段）。
type TrayMenuItem struct {
	ID        int    // 菜单项 ID（>0），点击时回传给 onSelect
	Label     string // 菜单文字
	Separator bool   // 是否为分隔线
	Disabled  bool   // 是否置灰
}

// AddTrayIcon 在系统通知区添加托盘图标，由平台层实现。
// hwnd: 关联的窗口句柄（接收托盘消息）；tooltip: 悬停提示；iconPath: .ico 路径（空=用窗口默认图标）；
// onLeftClick: 左键单击回调（nil=无）。返回托盘 ID（>0，用于后续修改/删除），失败返回 0。
var AddTrayIcon func(hwnd uintptr, tooltip, iconPath string, onLeftClick func()) int

// SetTrayMenu 设置/更新托盘右键菜单，由平台层实现。右键托盘弹出，点击项回调 onSelect(id)。
var SetTrayMenu func(trayID int, items []TrayMenuItem, onSelect func(id int))

// SetTrayTooltip 更新托盘悬停提示文字。
var SetTrayTooltip func(trayID int, tooltip string)

// RemoveTrayIcon 移除托盘图标。
var RemoveTrayIcon func(trayID int)

// ShowTrayBalloon 弹出托盘气泡通知（level: 0=信息, 1=警告, 2=错误），由平台层实现。
var ShowTrayBalloon func(trayID int, title, text string, level int)

// ── 无边框窗口操作（自绘标题栏）──────────────────────

// DragWindow 发起原生窗口拖动（自绘标题栏区域按住时调用），由平台层实现。
var DragWindow func(hwnd uintptr)

// MinimizeWindow 最小化窗口。
var MinimizeWindow func(hwnd uintptr)

// ToggleMaximizeWindow 在最大化与还原之间切换。
var ToggleMaximizeWindow func(hwnd uintptr)

// IsWindowMaximized 报告窗口当前是否最大化。
var IsWindowMaximized func(hwnd uintptr) bool

// SetTitleBarHitArea 声明无边框窗口的自绘标题栏命中区（client 顶部 height 高、右侧 rightExclude 宽除外），
// 该范围由系统处理拖动 / 双击最大化 / Aero Snap。由平台层实现。
var SetTitleBarHitArea func(hwnd uintptr, height, rightExclude int)

// TitleBarHitTest 自绘标题栏内的逐点命中回调（client 坐标）：返回 true=该点是可点控件(→HTCLIENT 交 goui)、
// false=空白拖动区(→HTCAPTION 系统拖窗)。由 app 设置（命中测试 Pipeline + 沿父链找 ClickTarget），
// win32 在 WM_NCHITTEST 标题栏区调用。让菜单/按钮可放进标题栏并响应点击，空白处仍可拖动。
var TitleBarHitTest func(x, y int) bool

// EnableWindowShadow 给（无边框）窗口启用 DWM 投影阴影 + Win11 圆角，由平台层实现。
var EnableWindowShadow func(hwnd uintptr)

// 鼠标光标形状（跨平台中性常量）。
const (
	CursorArrow   = 0 // 默认箭头
	CursorResizeH = 1 // ↔ 水平双向（拖动调宽）
	CursorResizeV = 2 // ↕ 垂直双向（拖动调高）
	CursorHand    = 3 // 手型（可点击）
	CursorText    = 4 // 文本 I 形（可编辑文本）
)

// SetCursor 设置窗口当前鼠标光标形状（见 Cursor* 常量），由平台层实现。
// 平台需在 WM_SETCURSOR 等时机持续应用该形状，否则系统会把光标重置回类光标。
var SetCursor func(hwnd uintptr, shape int)

// SetWindowIcon 从 .ico 文件设置窗口图标（任务栏/Alt+Tab 大图标 + 标题栏小图标），由平台层实现。
var SetWindowIcon func(hwnd uintptr, iconPath string)
