// virtualdemo — 验证 VirtualList 虚拟滚动（10000 项只渲染可视区）。
// 滚动需窗口；headless 渲染初始可视区 + 滚动条（很短 = 反映总量大）。
// 运行: $env:CGO_ENABLED='1'; go run ./examples/virtualdemo/
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
	cw = 360
	ch = 360
)

func main() {
	sk := canvas.NewSkiaCanvas(cw, ch)
	defer sk.Release()

	list := widget.NewVirtualList(10000, 32, func(i int) widget.Widget {
		bg := types.ColorRef(255, 255, 255)
		if i%2 == 1 {
			bg = types.ColorRef(250, 250, 252)
		}
		return widget.Div(widget.Style{Height: 32, Padding: types.EdgeInsetsLTRB(14, 0, 14, 0), FlexDirection: "row", AlignItems: "center", BackgroundColor: bg},
			widget.NewText(fmt.Sprintf("第 %d 项 / 共 10000", i+1), types.ColorFromRGB(96, 98, 102)),
		)
	}).WithHeight(280).WithWidth(300)

	root := widget.Div(
		widget.Style{Width: cw, Height: ch, Padding: types.EdgeInsets(24), BackgroundColor: types.ColorRef(255, 255, 255), FlexDirection: "column", Gap: 10},
		widget.H3("VirtualList 虚拟滚动（10000 项）"),
		list,
	)

	rootEl := widget.CreateElementFor(root)
	pipe := render.NewPipeline(cw, ch, sk)
	pipe.SetRootElement(rootEl)
	if err := pipe.Render(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if err := sk.SaveToPNG("virtual_demo.png"); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	fmt.Println("✅ virtual_demo.png 已保存")
}
