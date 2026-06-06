// overlay — Overlay 浮层定位系统演示，headless 渲染到 overlay_demo.png。
//
// 运行: $env:CGO_ENABLED='1'; go run ./examples/overlay/
// 输出: overlay_demo.png
//
// 验证点：四个深色浮层应环绕中心锚点(320,250)呈十字；贴顶锚点的 top 浮层
// 自动翻转到下方；贴右锚点的浮层被平移回视口内不越界。
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
	ch = 560
)

func tip(text string, tp widget.TagType, pl widget.Placement, anchor types.Rect) {
	widget.ShowOverlay(&widget.OverlayEntry{
		AnchorRect: anchor,
		Content:    widget.NewTag(text).WithEffect(widget.TagDark).WithType(tp),
		Placement:  pl,
	})
}

func main() {
	sk := canvas.NewSkiaCanvas(cw, ch)
	defer sk.Release()

	bg := widget.Div(
		widget.Style{Padding: types.EdgeInsets(20), BackgroundColor: types.ColorRef(245, 247, 250)},
		widget.H2("Overlay 浮层定位 · 12 方位 + 边缘翻转 + 视口纠偏"),
		widget.NewText("中心锚点 (320,250)：四个浮层环绕成十字", types.ColorFromRGB(120, 120, 120)),
	)
	host := widget.NewOverlayHost(bg)

	// 中心锚点：上下左右四方位
	center := types.Rect{X: 320, Y: 250, Width: 80, Height: 36}
	tip("◆ 锚点", widget.TagInfo, widget.PlacementBottom, types.Rect{X: 320, Y: 250, Width: 80, Height: 0}) // 贴锚点上沿当标记
	tip("Top 上方", widget.TagPrimary, widget.PlacementTop, center)
	tip("Bottom 下方", widget.TagPrimary, widget.PlacementBottom, center)
	tip("Left 左", widget.TagSuccess, widget.PlacementLeft, center)
	tip("Right 右", widget.TagSuccess, widget.PlacementRight, center)

	// 贴顶锚点 → top 放不下，自动翻转到 bottom
	tip("贴顶→自动翻到下方", widget.TagWarning, widget.PlacementTop,
		types.Rect{X: 60, Y: 8, Width: 120, Height: 30})

	// 贴右边锚点 → bottom 居中会越界，平移回视口
	tip("贴右→平移回视口内", widget.TagDanger, widget.PlacementBottom,
		types.Rect{X: 600, Y: 470, Width: 110, Height: 30})

	rootEl := widget.CreateElementFor(host)
	pipe := render.NewPipeline(cw, ch, sk)
	pipe.SetRootElement(rootEl)
	if err := pipe.Render(); err != nil {
		fmt.Fprintf(os.Stderr, "render: %v\n", err)
		os.Exit(1)
	}
	if err := sk.SaveToPNG("overlay_demo.png"); err != nil {
		fmt.Fprintf(os.Stderr, "save: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("✅ overlay_demo.png 已保存")
}
