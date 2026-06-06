// skiawidget — 完整 Widget 管线 + SkiaCanvas 渲染测试
//
// 本程序测试从 Widget 树 → Element → Layout → Paint(Canvas) → SkiaCanvas
// 的完整渲染管线，不依赖 OpenGL 窗口，直接输出 PNG 验证。
//
// 运行方式（需要设置 CGO_ENABLED=1）:
//   $env:CGO_ENABLED='1'; go run ./examples/skiawidget/
//
// 输出: skiawidget_output.png

//go:build !ignore
// +build !ignore

package main

import (
	"fmt"
	"os"

	"github.com/user/goui/internal/canvas"
	"github.com/user/goui/internal/render"
	"github.com/user/goui/internal/types"
	"github.com/user/goui/internal/widget"
)

func init() {
	// ── 配置主题 ──
	theme := widget.DefaultTheme()
	theme.PrimaryColor = types.ColorFromRGB(66, 133, 244)    // Google Blue
	theme.BGColor = types.ColorFromRGB(245, 247, 250)        // 浅灰蓝背景
	theme.SurfaceColor = types.ColorFromRGB(255, 255, 255)   // 白色卡片
	theme.TextColor = types.ColorFromRGB(33, 33, 33)         // 深灰文字
	theme.SecondaryText = types.ColorFromRGB(120, 120, 120)  // 次要文字
	theme.BorderColor = types.ColorFromRGB(220, 224, 228)    // 浅边框
	theme.SuccessColor = types.ColorFromRGB(52, 168, 83)     // 成功绿
	theme.WarningColor = types.ColorFromRGB(251, 188, 4)     // 警告黄
	theme.ErrorColor = types.ColorFromRGB(234, 67, 53)       // 错误红
	widget.SetTheme(theme)

	// ── 定义 CSS 风格样式类 ──
	widget.Define("page", widget.Style{
		Padding:         types.EdgeInsetsLTRB(24, 20, 24, 20),
		BackgroundColor: types.ColorRef(245, 247, 250),
	})

	widget.Define("card", widget.Style{
		Padding:         types.EdgeInsets(16),
		BackgroundColor: types.ColorRef(255, 255, 255),
		BorderRadius:    8,
	})

	widget.Define("card-bordered", widget.Style{
		Padding:         types.EdgeInsets(16),
		BackgroundColor: types.ColorRef(255, 255, 255),
		BorderColor:     types.ColorRef(220, 224, 228),
		BorderWidth:     1,
		BorderRadius:    8,
		Margin:          types.EdgeInsetsLTRB(0, 0, 0, 12),
	})

	widget.Define("toolbar", widget.Style{
		Padding: types.EdgeInsetsLTRB(0, 8, 0, 8),
	})
}

const (
	canvasW = 800
	canvasH = 1000
)

func main() {
	// ============================================================
	// 1. 创建 SkiaCanvas（作为最终渲染目标）
	// ============================================================
	sk := canvas.NewSkiaCanvas(canvasW, canvasH)
	defer sk.Release()

	// ============================================================
	// 2. 构建 Widget 树
	// ============================================================
	root := widget.Div(
		widget.Class("page"),

		// ── 标题区域 ──
		widget.Div(widget.Class("card"),
			widget.H1("goui + Skia 引擎集成测试"),
			widget.P("完整 Widget 管线渲染测试，使用 SkiaCanvas 硬件加速后端"),
			widget.Small("测试日期: 2026-06-02 · 800x1000 画布"),
		),

		// ── 第一行：按钮测试 ──
		widget.Div(widget.Class("card-bordered"),
			widget.H2("按钮组件"),
			widget.Div(
				widget.Style{Padding: types.EdgeInsetsLTRB(0, 8, 0, 8)},
				widget.HBox(
					widget.NewButton("主按钮", func() { fmt.Println("主按钮点击") }),
					widget.SpacerDiv(),
					widget.NewButton("次按钮", func() { fmt.Println("次按钮点击") }),
					widget.SpacerDiv(),
					widget.NewButton("危险", func() { fmt.Println("危险按钮点击") }),
				),
			),
		),

		// ── 第二行：复选框测试 ──
		widget.Div(widget.Class("card-bordered"),
			widget.H2("复选框组件"),
			widget.Div(
				widget.Style{Padding: types.EdgeInsetsLTRB(0, 4, 0, 4)},
				widget.NewCheckbox("启用功能 A", true, func(checked bool) {
					fmt.Printf("功能 A = %v\n", checked)
				}),
			),
			widget.Div(
				widget.Style{Padding: types.EdgeInsetsLTRB(0, 4, 0, 4)},
				widget.NewCheckbox("启用功能 B", false, func(checked bool) {
					fmt.Printf("功能 B = %v\n", checked)
				}),
			),
			widget.Div(
				widget.Style{Padding: types.EdgeInsetsLTRB(0, 4, 0, 4)},
				widget.NewCheckbox("启用功能 C", false, func(checked bool) {
					fmt.Printf("功能 C = %v\n", checked)
				}),
			),
		),

		// ── 第三行：输入框测试 ──
		widget.Div(widget.Class("card-bordered"),
			widget.H2("输入框组件"),
			widget.Div(
				widget.Style{Padding: types.EdgeInsetsLTRB(0, 4, 0, 4)},
				widget.NewInput("请输入用户名...", func(text string) {
					fmt.Printf("用户名输入: %q\n", text)
				}),
			),
			widget.Div(
				widget.Style{Padding: types.EdgeInsetsLTRB(0, 4, 0, 4)},
				widget.NewInput("请输入密码...", func(text string) {
					fmt.Printf("密码输入: %q\n", text)
				}),
			),
		),

		// ── 第四行：排版系统测试 ──
		widget.Div(widget.Class("card-bordered"),
			widget.H2("排版系统"),
			widget.H3("标题 H3 — 中文字体测试"),
			widget.H4("标题 H4 — SkiaCanvas 渲染"),
			widget.P("段落文本：这是使用 Skia 硬件加速引擎渲染的中文文本。阿里普惠体字体，支持中文、英文 12345 和标点符号。"),
			widget.NewText("自定义文本颜色", types.ColorFromRGB(66, 133, 244)),
		),

		// ── 第五行：分隔线和进度条 ──
		widget.Div(widget.Class("card-bordered"),
			widget.H2("分隔线与进度条"),
			widget.Separator(),
			widget.Div(
				widget.Style{Padding: types.EdgeInsetsLTRB(0, 8, 0, 8)},
				&widget.ProgressBar{Value: 0.75},
			),
			widget.Separator(),
			widget.Div(
				widget.Style{Padding: types.EdgeInsetsLTRB(0, 8, 0, 8)},
				&widget.ProgressBar{Value: 0.35},
			),
		),

		// ── 状态栏信息 ──
		widget.Div(widget.Class("card"),
			widget.Style{
				BackgroundColor: types.ColorRef(33, 33, 33),
				BorderRadius:    6,
			},
			widget.HBox(
				widget.NewText("状态: 渲染成功 ✓", types.ColorFromRGB(255, 255, 255)),
				widget.SpacerDiv(),
				widget.NewText("SkiaCanvas", types.ColorFromRGB(144, 164, 174)),
			),
		),
	)

	// ============================================================
	// 3. 通过渲染管线执行完整渲染
	// ============================================================
	//
	// 使用 Pipeline 处理：Build → Layout → Paint(Canvas)
	// 最终输出到 SkiaCanvas

	// 创建 Element
	rootEl := widget.CreateElementFor(root)

	// 创建并配置渲染管线
	pipe := render.NewPipeline(canvasW, canvasH, sk)
	pipe.SetRootElement(rootEl)

	// 执行完整渲染（Layout + Paint）
	if err := pipe.Render(); err != nil {
		fmt.Fprintf(os.Stderr, "渲染失败: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("✅ 渲染管线执行成功")

	// ============================================================
	// 4. 保存为 PNG
	// ============================================================
	if err := sk.SaveToPNG("skiawidget_output.png"); err != nil {
		fmt.Fprintf(os.Stderr, "保存 PNG 失败: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("✅ skiawidget_output.png 已保存 — 完整 Widget→SkiaCanvas 管线测试完成")

	// ============================================================
	// 5. 验证：查找 Element 树中的组件
	// ============================================================
	fmt.Println("\n📋 渲染 Element 树统计:")
	countElements(rootEl, 0)
}

// countElements 递归统计 Element 数量
func countElements(el widget.Element, depth int) {
	prefix := ""
	for i := 0; i < depth; i++ {
		prefix += "  "
	}
	wt := fmt.Sprintf("%T", el)
	off := el.Offset()
	sz := el.Size()
	fmt.Printf("%s• %s [%.0f,%.0f %.0fx%.0f]\n", prefix, wt, off.X, off.Y, sz.Width, sz.Height)
	for _, child := range el.Children() {
		countElements(child, depth+1)
	}
}
