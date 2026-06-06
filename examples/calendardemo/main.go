// calendardemo — 验证 Calendar 独立月历（复用 DatePicker 日历工具）。
// 运行: $env:CGO_ENABLED='1'; go run ./examples/calendardemo/
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
	cw = 470
	ch = 430
)

func main() {
	sk := canvas.NewSkiaCanvas(cw, ch)
	defer sk.Release()

	root := widget.Div(
		widget.Style{Width: cw, Height: ch, Padding: types.EdgeInsets(16), BackgroundColor: types.ColorRef(245, 247, 250), FlexDirection: "column", Gap: 10},
		widget.H3("Calendar 月历（复用 DatePicker 日历工具）"),
		widget.NewCalendar().WithValue("2024-06-15"),
	)

	rootEl := widget.CreateElementFor(root)
	pipe := render.NewPipeline(cw, ch, sk)
	pipe.SetRootElement(rootEl)
	if err := pipe.Render(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if err := sk.SaveToPNG("calendar_demo.png"); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	fmt.Println("✅ calendar_demo.png 已保存")
}
