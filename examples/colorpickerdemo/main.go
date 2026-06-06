// colorpickerdemo — 验证 ColorPicker（SV 渐变方块 + Hue 彩虹条 + 预览/hex）。
// 拖动选色需窗口实测；此处 headless 验证 HSV 渐变渲染。
// 运行: $env:CGO_ENABLED='1'; go run ./examples/colorpickerdemo/
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
	cw = 300
	ch = 320
)

func main() {
	sk := canvas.NewSkiaCanvas(cw, ch)
	defer sk.Release()

	root := widget.Div(
		widget.Style{Width: cw, Height: ch, Padding: types.EdgeInsets(24), BackgroundColor: types.ColorRef(255, 255, 255), FlexDirection: "column", Gap: 12},
		widget.H3("ColorPicker 取色器"),
		widget.NewColorPicker(),
	)

	rootEl := widget.CreateElementFor(root)
	pipe := render.NewPipeline(cw, ch, sk)
	pipe.SetRootElement(rootEl)
	if err := pipe.Render(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if err := sk.SaveToPNG("colorpicker_demo.png"); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	fmt.Println("✅ colorpicker_demo.png 已保存")
}
