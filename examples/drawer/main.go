// drawer — 验证 L3 模态组件：Drawer 抽屉(遮罩 + 右侧贴边面板)。
//
// 运行: $env:CGO_ENABLED='1'; go run ./examples/drawer/
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
	cw = 720
	ch = 460
)

func main() {
	sk := canvas.NewSkiaCanvas(cw, ch)
	defer sk.Release()

	// 背景：模拟应用内容
	bg := widget.Div(
		widget.Style{Width: cw, Height: ch, BackgroundColor: types.ColorRef(245, 247, 250), Padding: types.EdgeInsets(24)},
		widget.H2("应用主界面"),
		widget.P("点击按钮，从右侧滑出设置抽屉。遮罩压暗背景，"),
		widget.P("抽屉贴右边缘、占满整列高度。"),
	)

	// 抽屉：标题 + 内容 + 取消/保存按钮，从右侧滑出
	drawer := widget.NewDrawer("设置面板",
		widget.Div(
			widget.Style{FlexDirection: "column", Gap: 10},
			widget.P("在这里调整应用的偏好设置。"),
			widget.P("抽屉常用于侧边配置、详情查看。"),
			widget.Small("· 通知提醒"),
			widget.Small("· 隐私与安全"),
			widget.Small("· 外观主题"),
		),
	).WithSize(300).WithFooter(
		widget.NewButton("取消", nil).
			WithColor(types.ColorFromRGB(244, 244, 245)).
			WithTextColor(types.ColorFromRGB(96, 98, 102)),
		widget.NewButton("保存", nil).
			WithColor(types.ColorFromRGB(64, 158, 255)),
	)

	root := widget.NewStack(bg, drawer)

	rootEl := widget.CreateElementFor(root)
	pipe := render.NewPipeline(cw, ch, sk)
	pipe.SetRootElement(rootEl)
	if err := pipe.Render(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if err := sk.SaveToPNG("drawer_demo.png"); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	fmt.Println("✅ drawer_demo.png 已保存")
}
