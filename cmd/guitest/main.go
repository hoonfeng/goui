// guitest — goui 组件测试入口
//
// 本文件替代所有旧的 examples/ 目录下的示例代码，
// 使用 Web 风格组件（Div、H1、P、Button、Checkbox、Input 等）
// + 主题系统 + CSS 风格样式系统进行统一的组件测试。
//
// 运行方式:
//
//	go run ./cmd/guitest/
package main

import (
	"log"
	"runtime"

	"github.com/hoonfeng/goui/pkg/app"
	"github.com/hoonfeng/goui/pkg/types"
	"github.com/hoonfeng/goui/pkg/widget"
)

func init() {
	// ──────────────────────────────────────────────
	// 1. 自定义主题（类似 CSS 变量 / Design Tokens）
	// ──────────────────────────────────────────────
	myTheme := widget.DefaultTheme()
	myTheme.PrimaryColor = types.ColorFromRGB(66, 133, 244)   // Google Blue
	myTheme.BGColor = types.ColorFromRGB(245, 247, 250)       // 浅灰蓝背景
	myTheme.SurfaceColor = types.ColorFromRGB(255, 255, 255)  // 白色卡片
	myTheme.TextColor = types.ColorFromRGB(33, 33, 33)        // 深灰文字
	myTheme.SecondaryText = types.ColorFromRGB(120, 120, 120) // 次要文字
	myTheme.BorderColor = types.ColorFromRGB(220, 224, 228)   // 浅边框
	myTheme.SuccessColor = types.ColorFromRGB(52, 168, 83)    // 成功绿
	myTheme.WarningColor = types.ColorFromRGB(251, 188, 4)    // 警告黄
	myTheme.ErrorColor = types.ColorFromRGB(234, 67, 53)      // 错误红
	widget.SetTheme(myTheme)

	// ──────────────────────────────────────────────
	// 2. 定义 CSS 风格样式类（类似 CSS .class {}）
	// ──────────────────────────────────────────────

	// 页面容器
	widget.Define("page", widget.Style{
		Padding:         types.EdgeInsetsLTRB(32, 24, 32, 24),
		BackgroundColor: types.ColorRef(245, 247, 250),
	})

	// 卡片容器
	widget.Define("card", widget.Style{
		Padding:         types.EdgeInsets(16),
		BackgroundColor: types.ColorRef(255, 255, 255),
		BorderRadius:    8,
	})

	// 带边框的卡片
	widget.Define("card-bordered", widget.Style{
		Padding:         types.EdgeInsets(16),
		BackgroundColor: types.ColorRef(255, 255, 255),
		BorderColor:     types.ColorRef(220, 224, 228),
		BorderWidth:     1,
		BorderRadius:    8,
	})

	// 标题样式
	widget.Define("section-title", widget.Style{
		Padding: types.EdgeInsetsLTRB(0, 0, 0, 12),
	})

	// 高亮卡片（用于重要提示）
	widget.Define("highlight", widget.Style{
		Padding:         types.EdgeInsetsLTRB(16, 12, 16, 12),
		BackgroundColor: types.ColorRef(232, 240, 254),
		BorderRadius:    6,
	})

	// 成功卡片
	widget.Define("success-card", widget.Style{
		Padding:         types.EdgeInsetsLTRB(16, 12, 16, 12),
		BackgroundColor: types.ColorRef(230, 244, 234),
		BorderRadius:    6,
	})

	// 工具条（水平排列，带间距）
	widget.Define("toolbar", widget.Style{
		Padding: types.EdgeInsetsLTRB(0, 8, 0, 8),
	})
}

func main() {
	// 锁定主 goroutine 到 OS 线程（OpenGL 需要）
	runtime.LockOSThread()

	// ──────────────────────────────────────────────
	// 3. 构建 UI 界面（Web 风格组件）
	// ──────────────────────────────────────────────
	//
	// 使用 Div()、H1()~H4()、P()、Button()、Checkbox()、Input() 等
	// Web 风格组件构建界面，样式通过 Class() 引用或内联 Style{} 设置。

	root := widget.Div(
		widget.Class("page"),

		// ── 标题区域 ──
		widget.Div(widget.Class("section-title"),
			widget.H1("goui 组件测试"),
			widget.P("使用 Web 风格组件 + 主题系统 + CSS 样式系统"),
			widget.Small("2026-06-01 · 统一测试入口"),
		),

		// ── 第一行：按钮 + 复选框测试 ──
		widget.Div(
			widget.Class("card-bordered"),
			widget.H2("按钮与复选框"),
			widget.Div(
				widget.Style{Padding: types.EdgeInsetsLTRB(0, 8, 0, 4)},
				widget.NewText("按钮示例：", types.ColorFromRGB(33, 33, 33)),
			),
			widget.Div(
				widget.Style{Padding: types.EdgeInsetsLTRB(0, 4, 0, 8)},
				widget.HBox(
					widget.NewButton("确定", func() { log.Println("guitest: 确定按钮点击") }),
					widget.SpacerDiv(),
					widget.NewButton("取消", func() { log.Println("guitest: 取消按钮点击") }),
					widget.SpacerDiv(),
					widget.NewButton("危险操作", func() { log.Println("guitest: 危险按钮点击") }),
				),
			),
			widget.Separator(),
			// 复选框组
			widget.Div(
				widget.Style{Padding: types.EdgeInsetsLTRB(0, 8, 0, 4)},
				widget.NewText("复选框示例（选中/未选中状态）：", types.ColorFromRGB(33, 33, 33)),
			),
			widget.Div(
				widget.Style{Padding: types.EdgeInsetsLTRB(0, 4, 0, 4)},
				widget.NewCheckbox("显示完成状态", true, func(checked bool) {
					log.Printf("guitest: 显示完成状态 = %v", checked)
				}),
			),
			widget.Div(
				widget.Style{Padding: types.EdgeInsetsLTRB(0, 4, 0, 4)},
				widget.NewCheckbox("记住登录信息", false, func(checked bool) {
					log.Printf("guitest: 记住登录 = %v", checked)
				}),
			),
			widget.Div(
				widget.Style{Padding: types.EdgeInsetsLTRB(0, 4, 0, 4)},
				widget.NewCheckbox("同意用户协议", false, func(checked bool) {
					log.Printf("guitest: 同意协议 = %v", checked)
				}),
			),
		),

		// ── 第二行：输入框测试 ──
		widget.Div(
			widget.Class("card-bordered"),
			widget.H2("输入框"),
			widget.Div(
				widget.Style{Padding: types.EdgeInsetsLTRB(0, 8, 0, 4)},
				widget.NewText("文本输入框示例：", types.ColorFromRGB(33, 33, 33)),
			),
			widget.Div(
				widget.Style{Padding: types.EdgeInsetsLTRB(0, 4, 0, 4)},
				widget.NewInput("请输入用户名...", func(text string) {
					log.Printf("guitest: 用户名输入 = %q", text)
				}),
			),
			widget.Div(
				widget.Style{Padding: types.EdgeInsetsLTRB(0, 4, 0, 4)},
				widget.NewInput("请输入密码...", func(text string) {
					log.Printf("guitest: 密码输入 = %q", text)
				}),
			),
			widget.Separator(),
			// 带标签的输入
			widget.Div(
				widget.Style{Padding: types.EdgeInsetsLTRB(0, 8, 0, 4)},
				widget.NewText("带标签的行内输入：", types.ColorFromRGB(33, 33, 33)),
			),
			widget.Div(
				widget.Style{Padding: types.EdgeInsetsLTRB(0, 4, 0, 4)},
				widget.HBox(
					widget.NewText("邮箱：", types.ColorFromRGB(33, 33, 33)),
					widget.SpacerDiv(),
					widget.NewInput("example@mail.com", func(text string) {
						log.Printf("guitest: 邮箱输入 = %q", text)
					}),
				),
			),
		),

		// ── 第三行：排版测试 ──
		widget.Div(
			widget.Class("card-bordered"),
			widget.H2("排版系统"),
			widget.H1("标题 H1"),
			widget.H2("标题 H2"),
			widget.H3("标题 H3"),
			widget.H4("标题 H4"),
			widget.Div(
				widget.Style{Padding: types.EdgeInsetsLTRB(0, 4, 0, 4)},
				widget.P("段落文本（P）：这是 goui 的段落文本组件，支持自动换行和文本颜色设置。可以使用主题系统统一管理字体和颜色。"),
			),
			widget.Div(
				widget.Style{Padding: types.EdgeInsetsLTRB(0, 4, 0, 4)},
				widget.Small("小号文本（Small）：次要说明文字"),
			),
			widget.Div(
				widget.Style{Padding: types.EdgeInsetsLTRB(0, 4, 0, 4)},
				widget.Strong("强调文本（Strong）：使用粗体强调重要内容"),
			),
			widget.Separator(),
			widget.Div(
				widget.Style{Padding: types.EdgeInsetsLTRB(0, 8, 0, 4)},
				widget.NewText("自定义颜色文本：", types.ColorFromRGB(66, 133, 244)),
			),
		),

		// ── 第四行：样式系统测试 ──
		widget.Div(
			widget.Class("card-bordered"),
			widget.H2("CSS 风格样式系统"),
			widget.Div(
				widget.Class("highlight"),
				widget.NewText("提示：这是使用 Define('highlight', ...) 定义的高亮卡片样式", types.ColorFromRGB(13, 71, 161)),
			),
			widget.Div(
				widget.Style{Padding: types.EdgeInsetsLTRB(0, 8, 0, 4)},
				widget.NewText("内联样式示例：", types.ColorFromRGB(33, 33, 33)),
			),
			widget.Div(
				widget.Style{
					Padding:         types.EdgeInsets(8),
					BackgroundColor: types.ColorRef(230, 247, 255),
					BorderRadius:    4,
				},
				widget.P("此卡片使用内联 Style{Padding, BackgroundColor, BorderRadius} 创建"),
			),
		),

		// ── 第五行：分隔线和间距测试 ──
		widget.Div(
			widget.Class("card-bordered"),
			widget.H2("分隔线与间距"),
			widget.P("上方是一条水平分隔线（Separator）："),
			widget.Separator(),
			widget.P("下方是弹性占位空间（SpacerDiv）："),
			widget.Div(
				widget.Style{Padding: types.EdgeInsetsLTRB(0, 8, 0, 8)},
				widget.HBox(
					widget.NewText("左侧", types.ColorFromRGB(33, 33, 33)),
					widget.SpacerDiv(),
					widget.NewText("右侧", types.ColorFromRGB(33, 33, 33)),
				),
			),
		),

		// ── 第六行：成功提示测试 ──
		widget.Div(
			widget.Class("success-card"),
			widget.HBox(
				widget.Strong("✅ 所有组件加载成功"),
				widget.SpacerDiv(),
				widget.Small("goui 统一测试入口"),
			),
		),
	)

	// ──────────────────────────────────────────────
	// 4. 创建并启动应用
	// ──────────────────────────────────────────────
	application := app.NewApplication()
	application.SetRootWidget(root)

	config := app.DefaultConfig()
	config.Title = "goui 组件测试"
	config.Width = 680
	config.Height = 880
	config.Resizable = true

	log.Println("guitest: 启动 goui 组件测试应用")
	if err := application.Run(config); err != nil {
		log.Fatalf("guitest: 应用运行失败: %v", err)
	}
	log.Println("guitest: 应用正常退出")
}
