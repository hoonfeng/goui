// watermark — 验证 L3 定位组件：Watermark 水印覆盖层。
//
// 运行: $env:CGO_ENABLED='1'; go run ./examples/watermark/
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
	cw = 540
	ch = 400
)

func main() {
	sk := canvas.NewSkiaCanvas(cw, ch)
	defer sk.Release()

	content := widget.Div(
		widget.Style{Width: 470, Height: 290, BackgroundColor: types.ColorRef(255, 255, 255), BorderRadius: 8, Padding: types.EdgeInsets(20)},
		widget.H3("内容区域"),
		widget.P("这是被水印覆盖的内容。水印半透明、旋转、平铺在内容之上，"),
		widget.P("不影响内容阅读，常用于防泄密、版权标识。"),
	)

	root := widget.Div(
		widget.Style{Padding: types.EdgeInsets(24), BackgroundColor: types.ColorRef(245, 247, 250)},
		widget.H2("L3 · Watermark 水印"),
		widget.NewWatermark("goui · 机密", content).WithColor(types.ColorFromRGBA(64, 158, 255, 36)),
	)

	rootEl := widget.CreateElementFor(root)
	pipe := render.NewPipeline(cw, ch, sk)
	pipe.SetRootElement(rootEl)
	if err := pipe.Render(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if err := sk.SaveToPNG("watermark_demo.png"); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	fmt.Println("✅ watermark_demo.png 已保存")
}
