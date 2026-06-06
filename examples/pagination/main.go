// pagination — 验证 batch③ 中频组件：Pagination 分页(el-pagination 1:1)。
// 点页码/上下页切换需窗口实测；此处 headless 验证少页全显 + 多页省略号 + 当前高亮。
//
// 运行: $env:CGO_ENABLED='1'; go run ./examples/pagination/
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
	cw = 500
	ch = 240
)

func main() {
	sk := canvas.NewSkiaCanvas(cw, ch)
	defer sk.Release()

	root := widget.Div(
		widget.Style{Width: cw, Height: ch, Padding: types.EdgeInsets(24), BackgroundColor: types.ColorRef(255, 255, 255), FlexDirection: "column", Gap: 14},
		widget.H3("Pagination 分页（el-pagination 1:1）"),
		widget.P("少页（5 页全显示，当前第 2 页）:"),
		widget.NewPagination(50, 10, 2),
		widget.P("多页（20 页，当前第 6 页，带省略号）:"),
		widget.NewPagination(200, 10, 6),
	)

	rootEl := widget.CreateElementFor(root)
	pipe := render.NewPipeline(cw, ch, sk)
	pipe.SetRootElement(rootEl)
	if err := pipe.Render(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if err := sk.SaveToPNG("pagination_demo.png"); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	fmt.Println("✅ pagination_demo.png 已保存")
}
