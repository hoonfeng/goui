package app

import (
	"github.com/hoonfeng/goui/pkg/widget"
	"github.com/hoonfeng/goui/pkg/window"
)

// ── 窗口能力 API：透明窗口 / 系统托盘 / 托盘菜单 ──────────────
// 这些能力由平台层（win32）实现并在 init 中注册；未实现的平台调用为安全空操作。

// SetOpacity 运行时设置主窗口整体不透明度（0~1，整窗 alpha 混合）。
// 0 或 1 视为不透明（关闭分层）。
func (app *Application) SetOpacity(opacity float64) {
	if window.SetWindowOpacity != nil && app.Window != nil {
		window.SetWindowOpacity(app.Window.NativeHandle(), opacity)
	}
}

// TrayMenuItem 托盘菜单项（re-export，调用方无需直接 import window）。
type TrayMenuItem = window.TrayMenuItem

// AddTray 在系统通知区添加托盘图标，返回托盘 ID（>0；0=失败或平台不支持）。
// tooltip 悬停提示；iconPath .ico 文件路径（空=系统默认图标）；onLeftClick 左键单击回调（可 nil）。
func (app *Application) AddTray(tooltip, iconPath string, onLeftClick func()) int {
	if window.AddTrayIcon == nil || app.Window == nil {
		return 0
	}
	return window.AddTrayIcon(app.Window.NativeHandle(), tooltip, iconPath, onLeftClick)
}

// SetTrayMenu 设置/更新托盘右键菜单与点击回调。右键托盘弹出，选中项回调 onSelect(item.ID)。
func (app *Application) SetTrayMenu(trayID int, items []TrayMenuItem, onSelect func(id int)) {
	if window.SetTrayMenu != nil {
		window.SetTrayMenu(trayID, items, onSelect)
	}
}

// SetTrayTooltip 更新托盘悬停提示文字。
func (app *Application) SetTrayTooltip(trayID int, tooltip string) {
	if window.SetTrayTooltip != nil {
		window.SetTrayTooltip(trayID, tooltip)
	}
}

// RemoveTray 移除托盘图标。
func (app *Application) RemoveTray(trayID int) {
	if window.RemoveTrayIcon != nil {
		window.RemoveTrayIcon(trayID)
	}
}

// ShowTrayBalloon 弹出托盘气泡通知（level: 0=信息, 1=警告, 2=错误）。
func (app *Application) ShowTrayBalloon(trayID int, title, text string, level int) {
	if window.ShowTrayBalloon != nil {
		window.ShowTrayBalloon(trayID, title, text, level)
	}
}

// ── 无边框窗口 / 自绘标题栏操作 ──────────────────────
// 配合 app.Config.Borderless=true（隐藏系统标题栏）使用：自绘标题栏的按钮调用这些方法。

// DragWindow 发起主窗口拖动（自绘标题栏拖动区按下时调用）。
func (app *Application) DragWindow() {
	if window.DragWindow != nil && app.Window != nil {
		window.DragWindow(app.Window.NativeHandle())
	}
}

// Minimize 最小化主窗口。
func (app *Application) Minimize() {
	if window.MinimizeWindow != nil && app.Window != nil {
		window.MinimizeWindow(app.Window.NativeHandle())
	}
}

// ToggleMaximize 主窗口在最大化与还原之间切换。
func (app *Application) ToggleMaximize() {
	if window.ToggleMaximizeWindow != nil && app.Window != nil {
		window.ToggleMaximizeWindow(app.Window.NativeHandle())
	}
}

// IsMaximized 报告主窗口当前是否最大化（自绘标题栏据此切换最大化/还原图标）。
func (app *Application) IsMaximized() bool {
	if window.IsWindowMaximized != nil && app.Window != nil {
		return window.IsWindowMaximized(app.Window.NativeHandle())
	}
	return false
}

// Close 关闭主窗口（等同点自绘标题栏的关闭按钮）：请求主循环退出，
// 由 Run 的 defer 统一 Window.Close() 清理——避免窗口销毁后主循环再渲染一帧、
// 对着已失效的 DC 调 MakeCurrent 而报 wglCreateContext/MakeCurrent failed。
func (app *Application) Close() {
	app.Running = false
}

// SetTitleBar 声明自绘标题栏的命中区域：顶部 height 像素高、右侧 rightExclude 像素宽（按钮区）除外，
// 由系统接管拖动 / 双击最大化 / Aero Snap（比 WindowDragHandle 更完整，自带双击最大化）。
// 配合 Config.Borderless=true 使用，在 Ready 回调中按标题栏实际布局调用一次。
func (app *Application) SetTitleBar(height, rightExclude int) {
	if window.SetTitleBarHitArea != nil && app.Window != nil {
		window.SetTitleBarHitArea(app.Window.NativeHandle(), height, rightExclude)
	}
	// 逐点命中：标题栏里命中可点控件(ClickTarget：Button/Clickable/Dropdown…)→ 可点(HTCLIENT)，
	// 否则空白→ 系统拖动(HTCAPTION)。让菜单/按钮可嵌入标题栏并响应点击。
	window.TitleBarHitTest = func(x, y int) bool {
		if app.Pipeline == nil {
			return false
		}
		for el := app.Pipeline.HitTest(float64(x), float64(y)); el != nil; el = el.Parent() {
			if ct, ok := el.(widget.ClickTarget); ok && ct.IsClickTarget() {
				return true
			}
		}
		return false
	}
}

// EnableWindowEffects 给主窗口启用 DWM 投影阴影 + Win11 圆角（无边框窗口尤其需要——
// 没了系统标题栏就没了系统阴影；Win11 圆角在旧系统自动忽略）。在 Ready 中调用。
func (app *Application) EnableWindowEffects() {
	if window.EnableWindowShadow != nil && app.Window != nil {
		window.EnableWindowShadow(app.Window.NativeHandle())
	}
}

// SetIcon 设置主窗口图标（任务栏 / Alt+Tab / 标题栏），iconPath 为 .ico 文件路径。
func (app *Application) SetIcon(iconPath string) {
	if window.SetWindowIcon != nil && app.Window != nil {
		window.SetWindowIcon(app.Window.NativeHandle(), iconPath)
	}
}
