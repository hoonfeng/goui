// nav — 验证 batch② 导航组织组件：Steps 步骤条 + Breadcrumb 面包屑。
//
// 运行: $env:CGO_ENABLED='1'; go run ./examples/nav/
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
	cw = 560
	ch = 320
)

func main() {
	sk := canvas.NewSkiaCanvas(cw, ch)
	defer sk.Release()

	root := widget.Div(
		widget.Style{Width: cw, Height: ch, Padding: types.EdgeInsets(20), BackgroundColor: types.ColorRef(255, 255, 255), FlexDirection: "column", Gap: 14},
		widget.H3("导航组织类（el 1:1）"),

		widget.P("Breadcrumb 面包屑（前几项可点链接，末项当前）:"),
		widget.NewBreadcrumb(
			widget.BreadcrumbItem{Label: "首页"},
			widget.BreadcrumbItem{Label: "组件"},
			widget.BreadcrumbItem{Label: "导航"},
			widget.BreadcrumbItem{Label: "面包屑"},
		),

		widget.Div(widget.Style{Height: 6}),
		widget.P("Steps 步骤条（active=1：已完成✓ / 进行中 / 等待）:"),
		widget.NewSteps(
			widget.Step{Title: "填写信息", Description: "第一步"},
			widget.Step{Title: "核对数据", Description: "第二步"},
			widget.Step{Title: "提交完成", Description: "第三步"},
		).WithActive(1),
	)

	rootEl := widget.CreateElementFor(root)
	pipe := render.NewPipeline(cw, ch, sk)
	pipe.SetRootElement(rootEl)
	if err := pipe.Render(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if err := sk.SaveToPNG("nav_demo.png"); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	fmt.Println("✅ nav_demo.png 已保存")
}
