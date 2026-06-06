// dropdown — 验证 batch③ 中频组件：Dropdown 下拉菜单(el-dropdown 1:1)。
// 点触发器弹菜单需窗口实测；此处 headless 验证触发器 + 菜单外观(hover 高亮/禁用/分隔线)。
//
// 运行: $env:CGO_ENABLED='1'; go run ./examples/dropdown/
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
	cw = 480
	ch = 400
)

func main() {
	sk := canvas.NewSkiaCanvas(cw, ch)
	defer sk.Release()

	items := []widget.DropdownItem{
		{Label: "编辑", Command: "edit"},
		{Label: "复制", Command: "copy"},
		{Label: "分享", Command: "share"},
		{Label: "删除", Command: "delete", Divided: true},
		{Label: "归档（不可用）", Command: "archive", Disabled: true},
	}
	dropdown := widget.NewDropdown(widget.NewButton("更多操作 v", nil), items...)

	root := widget.Div(
		widget.Style{Width: cw, Height: ch, Padding: types.EdgeInsets(24), BackgroundColor: types.ColorRef(255, 255, 255), FlexDirection: "column", Gap: 14},
		widget.H3("Dropdown 下拉菜单（el-dropdown 1:1）"),
		widget.P("触发器（点击弹出菜单）:"),
		dropdown,
		widget.P("菜单展开预览（禁用项灰 / 「删除」上方分隔线）:"),
		widget.DropdownMenuPreview(items),
	)

	rootEl := widget.CreateElementFor(root)
	pipe := render.NewPipeline(cw, ch, sk)
	pipe.SetRootElement(rootEl)
	if err := pipe.Render(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if err := sk.SaveToPNG("dropdown_demo.png"); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	fmt.Println("✅ dropdown_demo.png 已保存")
}
