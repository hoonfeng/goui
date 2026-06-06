// basics — Element Plus 简单件画廊：Badge / Avatar / Link / Alert / Empty。
// headless 渲染到 basics_demo.png。
//
// 运行: $env:CGO_ENABLED='1'; go run ./examples/basics/
package main

import (
	"fmt"
	"os"

	"github.com/user/goui/internal/canvas"
	"github.com/user/goui/internal/render"
	"github.com/user/goui/internal/types"
	"github.com/user/goui/internal/widget"
)

const (
	cw = 760
	ch = 640
)

func gap(w float64) widget.Widget { return widget.Div(widget.Style{Width: w}) }

func row(label string, body widget.Widget) widget.Widget {
	return widget.Div(
		widget.Style{Padding: types.EdgeInsetsLTRB(0, 12, 0, 2)},
		widget.NewText(label, types.ColorFromRGB(120, 120, 120)),
		widget.Div(widget.Style{Padding: types.EdgeInsetsLTRB(0, 8, 0, 0)}, body),
	)
}

func main() {
	sk := canvas.NewSkiaCanvas(cw, ch)
	defer sk.Release()

	blue := types.ColorFromRGB(64, 158, 255)
	green := types.ColorFromRGB(103, 194, 58)

	root := widget.Div(
		widget.Style{Padding: types.EdgeInsets(24), BackgroundColor: types.ColorRef(255, 255, 255)},
		widget.H2("Element Plus 简单件 · Badge / Avatar / Link / Alert / Empty"),

		row("Badge 徽章", widget.HBox(
			widget.NewBadge("5", widget.NewAvatar("张").WithBgColor(blue)),
			gap(28),
			widget.NewBadge("100", widget.NewAvatar("李").WithBgColor(green)).WithMax(99),
			gap(28),
			widget.NewDotBadge(widget.NewAvatar("王")),
		)),

		row("Avatar 头像", widget.HBox(
			widget.NewAvatar("A").WithBgColor(blue),
			gap(12), widget.NewAvatar("方").WithShape(widget.AvatarSquare).WithBgColor(green),
			gap(12), widget.NewAvatar("大").WithSize(56).WithBgColor(types.ColorFromRGB(230, 162, 60)),
		)),

		row("Link 链接", widget.HBox(
			widget.NewLink("默认", nil),
			gap(16), widget.NewLink("主要", nil).WithType(widget.LinkPrimary),
			gap(16), widget.NewLink("成功", nil).WithType(widget.LinkSuccess),
			gap(16), widget.NewLink("警告", nil).WithType(widget.LinkWarning),
			gap(16), widget.NewLink("危险", nil).WithType(widget.LinkDanger),
			gap(16), widget.NewLink("禁用", nil).WithDisabled(true),
		)),

		widget.Div(
			widget.Style{Padding: types.EdgeInsetsLTRB(0, 12, 0, 2)},
			widget.NewText("Alert 警告", types.ColorFromRGB(120, 120, 120)),
			widget.Div(widget.Style{Padding: types.EdgeInsetsLTRB(0, 8, 0, 6)},
				widget.NewAlert("成功提示", widget.AlertSuccess).WithDescription("这是一条成功的描述文字").WithClosable(true)),
			widget.Div(widget.Style{Padding: types.EdgeInsetsLTRB(0, 0, 0, 6)},
				widget.NewAlert("警告提示", widget.AlertWarning).WithClosable(true)),
			widget.Div(widget.Style{Padding: types.EdgeInsetsLTRB(0, 0, 0, 0)},
				widget.NewAlert("危险提示（dark）", widget.AlertError).WithDark(true).WithDescription("实色风格")),
		),

		row("Empty 空状态", widget.NewEmpty("暂无数据")),
	)

	rootEl := widget.CreateElementFor(root)
	pipe := render.NewPipeline(cw, ch, sk)
	pipe.SetRootElement(rootEl)
	if err := pipe.Render(); err != nil {
		fmt.Fprintf(os.Stderr, "render: %v\n", err)
		os.Exit(1)
	}
	if err := sk.SaveToPNG("basics_demo.png"); err != nil {
		fmt.Fprintf(os.Stderr, "save: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("✅ basics_demo.png 已保存")
}
