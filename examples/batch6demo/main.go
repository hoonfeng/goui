// batch6demo — 验证补全第二批：TimePicker(触发器) / InputTag。
// 浮层/输入需窗口实测；此处 headless 验证布局。
// 运行: $env:CGO_ENABLED='1'; go run ./examples/batch6demo/
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
	cw = 460
	ch = 200
)

func main() {
	sk := canvas.NewSkiaCanvas(cw, ch)
	defer sk.Release()

	root := widget.Div(
		widget.Style{Width: cw, Height: ch, Padding: types.EdgeInsets(24), BackgroundColor: types.ColorRef(255, 255, 255), FlexDirection: "column", Gap: 18},
		widget.H3("补全第二批：TimePicker / InputTag"),
		widget.Div(widget.Style{FlexDirection: "row", AlignItems: "center", Gap: 12},
			widget.NewText("时间：", types.ColorFromRGB(96, 98, 102)),
			widget.NewTimePicker().WithValue("14:30"),
		),
		widget.Div(widget.Style{FlexDirection: "row", AlignItems: "center", Gap: 12},
			widget.NewText("标签：", types.ColorFromRGB(96, 98, 102)),
			widget.NewInputTag("Vue", "React", "Go"),
		),
	)

	rootEl := widget.CreateElementFor(root)
	pipe := render.NewPipeline(cw, ch, sk)
	pipe.SetRootElement(rootEl)
	if err := pipe.Render(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if err := sk.SaveToPNG("batch6_demo.png"); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	fmt.Println("✅ batch6_demo.png 已保存")
}
