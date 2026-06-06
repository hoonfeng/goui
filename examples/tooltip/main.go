// tooltip — Tooltip 气泡演示，headless 渲染到 tooltip_demo.png。
// 用命令式 ShowTooltip 把四个方位的 dark 气泡 + 一个 light 气泡显示出来，
// 验证气泡外观(深/浅底、文字、指向锚点的箭头)与定位。
//
// 运行: $env:CGO_ENABLED='1'; go run ./examples/tooltip/
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
	cw = 720
	ch = 480
)

func main() {
	sk := canvas.NewSkiaCanvas(cw, ch)
	defer sk.Release()

	bg := widget.Div(
		widget.Style{Padding: types.EdgeInsets(20), BackgroundColor: types.ColorRef(245, 247, 250)},
		widget.H2("Tooltip 文字提示 · dark/light · 12 方位 · 箭头"),
		widget.NewText("四个深色气泡环绕中心锚点；底部一个浅色气泡", types.ColorFromRGB(120, 120, 120)),
	)
	host := widget.NewOverlayHost(bg)

	widget.ShowTooltip("上方 Top", types.Rect{X: 330, Y: 150, Width: 60, Height: 30}, widget.PlacementTop, true)
	widget.ShowTooltip("下方 Bottom", types.Rect{X: 330, Y: 250, Width: 60, Height: 30}, widget.PlacementBottom, true)
	widget.ShowTooltip("左 Left", types.Rect{X: 200, Y: 200, Width: 60, Height: 30}, widget.PlacementLeft, true)
	widget.ShowTooltip("右 Right", types.Rect{X: 460, Y: 200, Width: 60, Height: 30}, widget.PlacementRight, true)
	widget.ShowTooltip("light 浅色风格", types.Rect{X: 320, Y: 380, Width: 90, Height: 30}, widget.PlacementBottom, false)

	rootEl := widget.CreateElementFor(host)
	pipe := render.NewPipeline(cw, ch, sk)
	pipe.SetRootElement(rootEl)
	if err := pipe.Render(); err != nil {
		fmt.Fprintf(os.Stderr, "render: %v\n", err)
		os.Exit(1)
	}
	if err := sk.SaveToPNG("tooltip_demo.png"); err != nil {
		fmt.Fprintf(os.Stderr, "save: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("✅ tooltip_demo.png 已保存")
}
