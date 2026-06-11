// transferdemo — 验证 Transfer 穿梭框（双面板 + checkbox + 移动按钮）。
// 勾选/移动需窗口实测；此处 headless 验证布局 + 文字居中。
// 运行: $env:CGO_ENABLED='1'; go run ./examples/transferdemo/
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
	cw = 550
	ch = 400
)

func main() {
	sk := canvas.NewSkiaCanvas(cw, ch)
	defer sk.Release()

	root := widget.Div(
		widget.Style{Width: cw, Height: ch, Padding: types.EdgeInsets(20), BackgroundColor: types.ColorRef(245, 247, 250), FlexDirection: "column", Gap: 12},
		widget.H3("Transfer 穿梭框"),
		widget.NewTransfer(
			widget.TransferItem{Key: "1", Label: "选项一"},
			widget.TransferItem{Key: "2", Label: "选项二"},
			widget.TransferItem{Key: "3", Label: "选项三"},
			widget.TransferItem{Key: "4", Label: "选项四"},
			widget.TransferItem{Key: "5", Label: "选项五"},
			widget.TransferItem{Key: "6", Label: "选项六"},
		).WithTitles("待选列表", "已选列表"),
	)

	rootEl := widget.CreateElementFor(root)
	pipe := render.NewPipeline(cw, ch, sk)
	pipe.SetRootElement(rootEl)
	if err := pipe.Render(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if err := sk.SaveToPNG("transfer_demo.png"); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	fmt.Println("✅ transfer_demo.png 已保存")
}
