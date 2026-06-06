// backtop — 验证滚动地基应用：Backtop 回到顶部按钮。
// headless 下 scrollY=0，用 WithVisibilityHeight(-1) 强制显示按钮以验证外观；
// 真实联动(滚动超 200px 才现身、点击回顶)需窗口实测(见 showcase)。
//
// 运行: $env:CGO_ENABLED='1'; go run ./examples/backtop/
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
	cw = 440
	ch = 400
)

func main() {
	sk := canvas.NewSkiaCanvas(cw, ch)
	defer sk.Release()

	items := make([]widget.Widget, 0, 20)
	for i := 1; i <= 20; i++ {
		items = append(items, widget.Div(
			widget.Style{BackgroundColor: types.ColorRef(245, 247, 250), BorderRadius: 6, Padding: types.EdgeInsets(12)},
			widget.H4(fmt.Sprintf("列表项 %d", i)),
			widget.P("滚动这片内容，回顶按钮会在右下角浮现。"),
		))
	}
	contentArgs := []interface{}{widget.Style{FlexDirection: "column", Gap: 8}}
	for _, it := range items {
		contentArgs = append(contentArgs, it)
	}
	content := widget.Div(contentArgs...)

	scroll := widget.NewScrollView(content)
	// -1 仅为 headless 静态展示按钮；实际用默认 200(滚动超 200px 才现身)
	backtop := widget.NewBacktop(scroll).WithVisibilityHeight(-1).WithOffset(24, 24)

	viewport := widget.Div(
		widget.Style{Width: 400, Height: 320, BackgroundColor: types.ColorRef(252, 253, 255)},
		backtop,
	)

	root := widget.Div(
		widget.Style{Width: cw, Height: ch, Padding: types.EdgeInsets(20), BackgroundColor: types.ColorRef(255, 255, 255)},
		widget.H3("Backtop 回到顶部"),
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
	if err := sk.SaveToPNG("backtop_demo.png"); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	fmt.Println("✅ backtop_demo.png 已保存")
}
