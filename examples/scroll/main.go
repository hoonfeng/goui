// scroll — 验证滚动地基：ScrollView(裁剪 + 滚动条)。
// 长列表放进固定高视口，超出部分被裁剪，右侧出现滚动条滑块。
// 滚动交互(滚轮)需窗口实测；此处 headless 验证裁剪 + 滚动条静态呈现。
//
// 运行: $env:CGO_ENABLED='1'; go run ./examples/scroll/
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
	cw = 440
	ch = 400
)

func main() {
	sk := canvas.NewSkiaCanvas(cw, ch)
	defer sk.Release()

	// 长内容：20 个卡片(远超视口高度)
	items := make([]widget.Widget, 0, 20)
	for i := 1; i <= 20; i++ {
		items = append(items, widget.Div(
			widget.Style{BackgroundColor: types.ColorRef(245, 247, 250), BorderRadius: 6, Padding: types.EdgeInsets(12)},
			widget.H4(fmt.Sprintf("列表项 %d", i)),
			widget.P("可滚动内容，超出视口的部分被裁剪，滚轮上下滚动。"),
		))
	}
	contentArgs := []interface{}{widget.Style{FlexDirection: "column", Gap: 8}}
	for _, it := range items {
		contentArgs = append(contentArgs, it)
	}
	content := widget.Div(contentArgs...)

	// ScrollView 装进固定高 320 的视口框 → 内容溢出 → 出现滚动条
	scroll := widget.NewScrollView(content)
	viewport := widget.Div(
		widget.Style{Width: 400, Height: 320, BackgroundColor: types.ColorRef(252, 253, 255)},
		scroll,
	)

	root := widget.Div(
		widget.Style{Width: cw, Height: ch, Padding: types.EdgeInsets(20), BackgroundColor: types.ColorRef(255, 255, 255)},
		widget.H3("ScrollView 滚动地基"),
		widget.Div(widget.Style{Height: 10}),
		viewport,
	)

	rootEl := widget.CreateElementFor(root)
	pipe := render.NewPipeline(cw, ch, sk)
	pipe.SetRootElement(rootEl)
	if err := pipe.Render(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if err := sk.SaveToPNG("scroll_demo.png"); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	fmt.Println("✅ scroll_demo.png 已保存")
}
