// slider — 验证 batch③ 中频组件：Slider 滑块(el-slider 1:1)。
// 拖拽选值需窗口实测；此处 headless 验证各值/步长/自定义色/禁用的静态呈现。
//
// 运行: $env:CGO_ENABLED='1'; go run ./examples/slider/
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
	ch = 320
)

func main() {
	sk := canvas.NewSkiaCanvas(cw, ch)
	defer sk.Release()
	green := types.ColorFromRGB(103, 194, 58)

	root := widget.Div(
		widget.Style{Width: cw, Height: ch, Padding: types.EdgeInsets(24), BackgroundColor: types.ColorRef(255, 255, 255), FlexDirection: "column", Gap: 12},
		widget.H3("Slider 滑块（el-slider 1:1）"),
		widget.NewSlider(30, nil).WithLabel(true),
		widget.NewSlider(60, nil),
		widget.NewSlider(45, nil).WithStep(5).WithLabel(true),
		widget.NewSlider(70, nil).WithColor(green),
		widget.NewSlider(40, nil).WithDisabled(true),
	)

	rootEl := widget.CreateElementFor(root)
	pipe := render.NewPipeline(cw, ch, sk)
	pipe.SetRootElement(rootEl)
	if err := pipe.Render(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if err := sk.SaveToPNG("slider_demo.png"); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	fmt.Println("✅ slider_demo.png 已保存")
}
