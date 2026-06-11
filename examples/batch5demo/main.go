// batch5demo — 验证补全第一批：PageHeader / Splitter / TimeSelect / TreeSelect / DateTimePicker。
// 浮层/拖动需窗口实测；此处 headless 验证布局 + 触发器外观。
// 运行: $env:CGO_ENABLED='1'; go run ./examples/batch5demo/
package main

import (
	"fmt"
	"os"

	"github.com/hoonfeng/goui/pkg/canvas"
	"github.com/hoonfeng/goui/pkg/render"
	"github.com/hoonfeng/goui/pkg/types"
	"github.com/hoonfeng/goui/pkg/widget"
)

const (
	cw = 600
	ch = 460
)

func main() {
	sk := canvas.NewSkiaCanvas(cw, ch)
	defer sk.Release()

	blue := types.ColorFromRGB(64, 158, 255)
	treeNodes := []*widget.TreeNode{
		{Label: "一级 A", Children: []*widget.TreeNode{{Label: "二级 A1"}, {Label: "二级 A2"}}},
		{Label: "一级 B"},
	}

	root := widget.Div(
		widget.Style{Width: cw, Height: ch, Padding: types.EdgeInsets(20), BackgroundColor: types.ColorRef(245, 247, 250), FlexDirection: "column", Gap: 14},
		widget.H3("补全第一批"),
		widget.NewPageHeader("用户详情").WithSubTitle("ID 12345").WithOnBack(func() {}).
			WithExtra(widget.NewButton("编辑", nil).WithColor(blue)),
		widget.NewSplitter(
			widget.Div(widget.Style{BackgroundColor: types.ColorRef(236, 245, 255), AlignItems: "center", JustifyContent: "center"}, widget.NewText("左面板", types.ColorFromRGB(64, 158, 255))),
			widget.Div(widget.Style{BackgroundColor: types.ColorRef(240, 249, 235), AlignItems: "center", JustifyContent: "center"}, widget.NewText("右面板", types.ColorFromRGB(103, 194, 58))),
		).WithHeight(110),
		widget.Div(widget.Style{FlexDirection: "row", Gap: 14, AlignItems: "center"},
			widget.NewTimeSelect().WithValue("09:30"),
			widget.NewTreeSelect(treeNodes...).WithValue("二级 A1"),
		),
		widget.NewDateTimePicker().WithValue("2024-06-15 09:00"),
	)

	rootEl := widget.CreateElementFor(root)
	pipe := render.NewPipeline(cw, ch, sk)
	pipe.SetRootElement(rootEl)
	if err := pipe.Render(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if err := sk.SaveToPNG("batch5_demo.png"); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	fmt.Println("✅ batch5_demo.png 已保存")
}
