// loading — 验证 L3 模态组件：Loading 加载遮罩(遮罩 + spinner + 文字)。
//
// 运行: $env:CGO_ENABLED='1'; go run ./examples/loading/
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
	ch = 320
)

func main() {
	sk := canvas.NewSkiaCanvas(cw, ch)
	defer sk.Release()

	// 背景：模拟正在加载数据的内容
	bg := widget.Div(
		widget.Style{Width: cw, Height: ch, BackgroundColor: types.ColorRef(255, 255, 255), Padding: types.EdgeInsets(24)},
		widget.H2("数据报表"),
		widget.P("表格内容正在从服务器拉取……"),
		widget.P("加载期间用半透明遮罩盖住，spinner 居中转圈。"),
	)

	// 加载遮罩：spinner 转一点角度展示拖尾
	loading := widget.NewLoading("加载中...")
	loading.Angle = 0.6

	root := widget.NewStack(bg, loading)

	rootEl := widget.CreateElementFor(root)
	pipe := render.NewPipeline(cw, ch, sk)
	pipe.SetRootElement(rootEl)
	if err := pipe.Render(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if err := sk.SaveToPNG("loading_demo.png"); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	fmt.Println("✅ loading_demo.png 已保存")
}
