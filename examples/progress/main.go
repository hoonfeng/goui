// progress — 验证 batch③ 中频组件：Progress 进度条(el-progress 1:1)。
//
// 运行: $env:CGO_ENABLED='1'; go run ./examples/progress/
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
	cw = 460
	ch = 360
)

func main() {
	sk := canvas.NewSkiaCanvas(cw, ch)
	defer sk.Release()

	root := widget.Div(
		widget.Style{Width: cw, Height: ch, Padding: types.EdgeInsets(24), BackgroundColor: types.ColorRef(255, 255, 255), FlexDirection: "column", Gap: 16},
		widget.H3("Progress 进度条（el-progress 1:1）"),
		widget.NewProgress(0.3),
		widget.NewProgress(0.65),
		widget.NewProgress(0.9).WithStatus("success"),
		widget.NewProgress(0.5).WithStatus("warning"),
		widget.NewProgress(0.4).WithStatus("exception"),
		widget.NewProgress(0.7).WithHeight(14),
	)

	rootEl := widget.CreateElementFor(root)
	pipe := render.NewPipeline(cw, ch, sk)
	pipe.SetRootElement(rootEl)
	if err := pipe.Render(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if err := sk.SaveToPNG("progress_demo.png"); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	fmt.Println("✅ progress_demo.png 已保存")
}
