// dialog — 验证 L3 模态组件：Dialog 对话框(遮罩 + 居中面板)。
//
// 运行: $env:CGO_ENABLED='1'; go run ./examples/dialog/
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
	cw = 620
	ch = 460
)

func main() {
	sk := canvas.NewSkiaCanvas(cw, ch)
	defer sk.Release()

	// 背景：模拟应用内容(半透明遮罩会把它压暗)
	bg := widget.Div(
		widget.Style{Width: cw, Height: ch, BackgroundColor: types.ColorRef(245, 247, 250), Padding: types.EdgeInsets(24)},
		widget.H2("应用主界面"),
		widget.P("这是对话框背后的页面内容，半透明遮罩会把它压暗，"),
		widget.P("让视线聚焦到弹出的对话框上。"),
	)

	// 对话框：标题 + 内容 + 取消/确定按钮
	dialog := widget.NewDialog("提示",
		widget.P("此操作将永久删除该文件，是否继续？"),
	).WithFooter(
		widget.NewButton("取消", nil).
			WithColor(types.ColorFromRGB(244, 244, 245)).
			WithTextColor(types.ColorFromRGB(96, 98, 102)),
		widget.NewButton("确定", nil).
			WithColor(types.ColorFromRGB(64, 158, 255)),
	)

	root := widget.NewStack(bg, dialog)

	rootEl := widget.CreateElementFor(root)
	pipe := render.NewPipeline(cw, ch, sk)
	pipe.SetRootElement(rootEl)
	if err := pipe.Render(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if err := sk.SaveToPNG("dialog_demo.png"); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	fmt.Println("✅ dialog_demo.png 已保存")
}
