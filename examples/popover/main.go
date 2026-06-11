// popover — Popover 弹出框演示，headless 渲染到 popover_demo.png。
// 用命令式 ShowPopover 显示三个不同方位的弹出框，验证 title+内容、箭头、阴影、换行。
//
// 运行: $env:CGO_ENABLED='1'; go run ./examples/popover/
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
	cw = 720
	ch = 480
)

func main() {
	sk := canvas.NewSkiaCanvas(cw, ch)
	defer sk.Release()

	bg := widget.Div(
		widget.Style{Padding: types.EdgeInsets(20), BackgroundColor: types.ColorRef(245, 247, 250)},
		widget.H2("Popover 弹出框 · 标题 + 内容 + 箭头 + 阴影"),
		widget.NewText("三个不同方位的弹出框，内容自动换行", types.ColorFromRGB(120, 120, 120)),
	)
	host := widget.NewOverlayHost(bg)

	widget.ShowPopover("提示标题", "这是一段弹出框的说明内容，支持自动换行排版显示。",
		types.Rect{X: 130, Y: 150, Width: 90, Height: 32}, widget.PlacementBottom, 220)
	widget.ShowPopover("右侧弹出", "箭头朝左指向锚点。",
		types.Rect{X: 520, Y: 200, Width: 90, Height: 32}, widget.PlacementRight, 170)
	widget.ShowPopover("", "无标题，仅有内容，也能正常显示并排版。",
		types.Rect{X: 250, Y: 380, Width: 90, Height: 32}, widget.PlacementTop, 200)

	rootEl := widget.CreateElementFor(host)
	pipe := render.NewPipeline(cw, ch, sk)
	pipe.SetRootElement(rootEl)
	if err := pipe.Render(); err != nil {
		fmt.Fprintf(os.Stderr, "render: %v\n", err)
		os.Exit(1)
	}
	if err := sk.SaveToPNG("popover_demo.png"); err != nil {
		fmt.Fprintf(os.Stderr, "save: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("✅ popover_demo.png 已保存")
}
