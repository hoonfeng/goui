// windowfeatures — 窗口能力演示：透明窗口 / 系统托盘 / 托盘菜单 / 多窗口
//
// 运行（Windows，需 libSkiaSharp.dll + fonts/ 在 cwd）：
//
//	$env:CGO_ENABLED='1'; go run ./examples/windowfeatures/
//
//   · 主窗口以 95% 不透明度显示（半透明）
//   · 系统通知区出现托盘图标，右键弹出菜单
//   · 点「打开副窗口」可开任意多个独立渲染的附属窗口
//
//go:build windows

package main

import (
	"fmt"

	"github.com/user/goui/internal/app"
	"github.com/user/goui/internal/types"
	"github.com/user/goui/internal/widget"
	"github.com/user/goui/internal/window"

	_ "github.com/user/goui/internal/platform"
)

var (
	application *app.Application
	trayID      int
	opaque      = false // 启动即 95%，视为半透明态
	subCount    int
)

func main() {
	application = app.NewApplication()
	application.SetRootWidget(mainUI())

	// Ready：首帧后注册系统托盘 + 托盘菜单（须在窗口创建后）
	application.Ready = func() {
		trayID = application.AddTray("goui 窗口能力示例", "", func() {
			widget.MessageInfo("托盘左键单击 👋")
		})
		application.SetTrayMenu(trayID, []app.TrayMenuItem{
			{ID: 1, Label: "打开副窗口"},
			{ID: 2, Label: "切换主窗口透明度"},
			{ID: 3, Label: "弹个托盘气泡通知"},
			{Separator: true},
			{ID: 9, Label: "退出应用"},
		}, onTrayMenu)

		// 声明自绘标题栏命中区：40 高，右侧 138(=3×46 按钮)除外 → 系统处理拖动/双击最大化/Aero Snap
		application.SetTitleBar(40, 138)
		// DWM 投影阴影 + Win11 圆角（无边框窗口补回系统阴影）
		application.EnableWindowEffects()
	}

	cfg := app.Config{
		Title:      "goui 窗口能力 Demo",
		Width:      640,
		Height:     440,
		Resizable:  true,
		Borderless: true, // 无边框：隐藏系统标题栏，用下面 titleBar() 自绘
	}
	if err := application.Run(cfg); err != nil {
		fmt.Println("运行失败:", err)
	}
}

func onTrayMenu(id int) {
	switch id {
	case 1:
		openSubWindow()
	case 2:
		toggleOpacity()
	case 3:
		application.ShowTrayBalloon(trayID, "goui 通知", "这是一条托盘气泡通知 🎉", 0)
	case 9:
		application.Close()
	}
}

func toggleOpacity() {
	opaque = !opaque
	if opaque {
		application.SetOpacity(1)
		widget.MessageSuccess("主窗口：不透明")
	} else {
		application.SetOpacity(0.7)
		widget.MessageSuccess("主窗口：70% 半透明")
	}
}

// openSubWindow 打开一个独立渲染管线的附属窗口（多窗口）。
func openSubWindow() {
	subCount++
	n := subCount
	var sub *app.SubWindow
	subBtn := func(label string, onClick func()) widget.Widget {
		return &widget.Button{Text: label, OnClick: onClick,
			Color: types.ColorFromRGB(60, 64, 78), TextColor: types.ColorFromRGB(235, 238, 245),
			MinWidth: 44, MinHeight: 36}
	}
	root := widget.VBox(
		// 副窗口也是无边框 + 自绘标题栏
		widget.Div(
			widget.Style{Height: 36, BackgroundColor: types.ColorRef(60, 64, 78)},
			widget.HBox(
				&widget.Expanded{
					SingleChildWidget: widget.SingleChildWidget{Child: widget.Div(
						widget.Style{Height: 36, Padding: types.EdgeInsetsLTRB(12, 0, 0, 0)},
						widget.NewText(fmt.Sprintf("副窗口 #%d", n), types.ColorFromRGB(235, 238, 245)),
					)},
					Flex: 1,
				},
				subBtn("—", func() {
					if sub != nil {
						sub.Minimize()
					}
				}),
				subBtn("☐", func() {
					if sub != nil {
						sub.ToggleMaximize()
					}
				}),
				subBtn("✕", func() {
					if sub != nil {
						sub.Close()
					}
				}),
			),
		),
		widget.Div(
			widget.Style{Padding: types.EdgeInsets(20), BackgroundColor: types.ColorRef(255, 255, 255)},
			widget.VBox(
				widget.H2(fmt.Sprintf("副窗口 #%d", n)),
				widget.P("我也是无边框 + 自绘标题栏 + 圆角阴影。"),
				widget.P("拖标题栏移动、双击最大化、— ☐ ✕ 都好使。"),
			),
		),
	)
	cfg := window.WindowConfig{
		Title:      fmt.Sprintf("goui 副窗口 #%d", n),
		Width:      400,
		Height:     300,
		Resizable:  true,
		Borderless: true,
	}
	s, err := application.OpenWindow(cfg, root)
	if err != nil {
		widget.MessageError("打开副窗口失败: " + err.Error())
		return
	}
	sub = s
	sub.SetTitleBar(36, 3*44) // 标题栏 36 高，右侧 3 按钮(44 宽)除外
	sub.EnableEffects()       // 圆角 + 阴影
}

func mainUI() widget.Widget {
	return widget.VBox(
		titleBar(), // 自绘标题栏（无边框窗口）
		widget.Div(
			widget.Style{Padding: types.EdgeInsets(24), BackgroundColor: types.ColorRef(245, 247, 250)},
			widget.VBox(
				widget.H1("goui 窗口能力演示"),
				widget.P("· 本窗口无系统标题栏，顶部深色条是自绘的"),
				widget.P("· 按住标题栏空白处可拖动窗口；右上角 — ☐ ✕ 最小化/最大化/关闭"),
				widget.P("· 系统通知区有托盘图标，右键试试菜单"),
				widget.P("· 多窗口：每个副窗口独立渲染"),
				widget.Div(widget.Style{Height: 12}),
				widget.HBox(
					widget.NewButton("打开副窗口", openSubWindow),
					widget.Div(widget.Style{Width: 12}),
					widget.NewButton("透明 70%", func() { application.SetOpacity(0.7) }),
					widget.Div(widget.Style{Width: 12}),
					widget.NewButton("不透明", func() { application.SetOpacity(1) }),
				),
			),
		),
	)
}

// titleBar 自绘标题栏：左侧拖动区（图标+标题），右侧最小化/最大化/关闭按钮。
func titleBar() widget.Widget {
	return widget.Div(
		widget.Style{Height: 40, BackgroundColor: types.ColorRef(48, 52, 64)}, // 深色标题栏
		widget.HBox(
			&widget.Expanded{
				// 标题区不用 WindowDragHandle —— 改由 SetTitleBar 声明命中区，系统处理拖动 + 双击最大化
				SingleChildWidget: widget.SingleChildWidget{Child: widget.Div(
					widget.Style{Height: 40, Padding: types.EdgeInsetsLTRB(14, 0, 0, 0)},
					widget.NewText("🪟  goui 窗口能力 Demo", types.ColorFromRGB(235, 238, 245)),
				)},
				Flex: 1,
			},
			titleButton("—", func() { application.Minimize() }),
			titleButton("☐", func() { application.ToggleMaximize() }),
			titleButton("✕", func() { application.Close() }),
		),
	)
}

func titleButton(label string, onClick func()) widget.Widget {
	return &widget.Button{
		Text:      label,
		OnClick:   onClick,
		Color:     types.ColorFromRGB(48, 52, 64), // 与标题栏同色，hover 自动变暗
		TextColor: types.ColorFromRGB(235, 238, 245),
		MinWidth:  46,
		MinHeight: 40,
	}
}
