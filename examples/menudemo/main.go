// menudemo — 验证 Menu 级联：有 Children 的项显示 ▶（子菜单 hover 弹出需窗口实测）。
// 运行: $env:CGO_ENABLED='1'; go run ./examples/menudemo/
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
	cw = 240
	ch = 210
)

func main() {
	sk := canvas.NewSkiaCanvas(cw, ch)
	defer sk.Release()

	menu := widget.NewMenu(
		widget.MenuItem{Label: "新建", Icon: "📄", Enabled: true},
		widget.MenuItem{Label: "打开", Enabled: true},
		widget.MenuItem{Label: "导出为", Enabled: true, Children: []widget.MenuItem{
			{Label: "PDF", Enabled: true},
			{Label: "PNG 图片", Enabled: true},
		}},
		widget.MenuItem{Label: "删除", Enabled: true},
	)

	root := widget.Div(
		widget.Style{Width: cw, Height: ch, Padding: types.EdgeInsets(16), BackgroundColor: types.ColorRef(245, 247, 250)},
		menu,
	)

	el := widget.CreateElementFor(root)
	pipe := render.NewPipeline(cw, ch, sk)
	pipe.SetRootElement(el)
	if err := pipe.Render(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if err := sk.SaveToPNG("menu_demo.png"); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	fmt.Println("✅ menu_demo.png 已保存")
}
