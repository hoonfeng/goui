// l3demo — 验证 L3 绝对定位的**纯 Style 声明式闭环**：
// Style{Position:"relative"} 容器 + Style{Position:"absolute", Top/Left/Right/Bottom, ZIndex} 子，
// 不写一行 Stack/Positioned，全由样式驱动。
//
// 运行: $env:CGO_ENABLED='1'; go run ./examples/l3demo/
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
	ch = 360
)

func main() {
	sk := canvas.NewSkiaCanvas(cw, ch)
	defer sk.Release()
	blue := types.ColorRef(64, 158, 255)
	green := types.ColorRef(103, 194, 58)
	orange := types.ColorRef(230, 162, 60)
	red := types.ColorRef(245, 108, 108)
	white := types.ColorWhite
	p := widget.Px

	root := widget.Div(
		widget.Style{Padding: types.EdgeInsets(24), BackgroundColor: types.ColorRef(245, 247, 250)},
		widget.H2("L3 · 纯 Style 绝对定位 + z-index"),
		widget.Div(
			widget.Style{Position: "relative", Width: 420, Height: 240, BackgroundColor: types.ColorRef(228, 231, 237), BorderRadius: 8},
			widget.Div(widget.Style{Position: "absolute", Left: p(0), Top: p(0), Width: 120, Height: 46, BackgroundColor: blue, BorderRadius: 6, Padding: types.EdgeInsets(8)},
				widget.NewText("left/top 0", white)),
			widget.Div(widget.Style{Position: "absolute", Right: p(0), Bottom: p(0), Width: 140, Height: 46, BackgroundColor: green, BorderRadius: 6, Padding: types.EdgeInsets(8)},
				widget.NewText("right/bottom 0", white)),
			widget.Div(widget.Style{Position: "absolute", Left: p(150), Top: p(86), Width: 130, Height: 70, BackgroundColor: orange, BorderRadius: 6, Padding: types.EdgeInsets(8), ZIndex: 1},
				widget.NewText("z-index 1", white)),
			widget.Div(widget.Style{Position: "absolute", Left: p(190), Top: p(110), Width: 130, Height: 70, BackgroundColor: red, BorderRadius: 6, Padding: types.EdgeInsets(8), ZIndex: 2},
				widget.NewText("z-index 2 盖住", white)),
		),
	)

	rootEl := widget.CreateElementFor(root)
	pipe := render.NewPipeline(cw, ch, sk)
	pipe.SetRootElement(rootEl)
	if err := pipe.Render(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if err := sk.SaveToPNG("l3_demo.png"); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	fmt.Println("✅ l3_demo.png 已保存")
}
