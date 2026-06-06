// uploaddemo — 验证 Upload 拖拽区 + 文件列表（hover/点击需窗口实测）。
// 运行: $env:CGO_ENABLED='1'; go run ./examples/uploaddemo/
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
	ch = 340
)

func main() {
	sk := canvas.NewSkiaCanvas(cw, ch)
	defer sk.Release()

	root := widget.Div(
		widget.Style{Width: cw, Height: ch, Padding: types.EdgeInsets(20), BackgroundColor: types.ColorRef(255, 255, 255), FlexDirection: "column", Gap: 12},
		widget.H3("Upload（文件列表拼装 + 删除）"),
		widget.NewUpload().WithFiles("季度报告.pdf", "销售数据.xlsx", "产品图.png").WithOnRemove(func(i int) {}),
	)

	rootEl := widget.CreateElementFor(root)
	pipe := render.NewPipeline(cw, ch, sk)
	pipe.SetRootElement(rootEl)
	if err := pipe.Render(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if err := sk.SaveToPNG("upload_demo.png"); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	fmt.Println("✅ upload_demo.png 已保存")
}
