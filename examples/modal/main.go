// modal — 验证模态弹出闭环：OverlayHost + ShowDialog(Fullscreen overlay)。
// 与 examples/dialog 不同：这里走真正的命令式弹出路径(挂到 OverlayHost 顶层)，
// 验证 Overlay 的 Fullscreen 模式铺满视口、对话框居中。
//
// 运行: $env:CGO_ENABLED='1'; go run ./examples/modal/
package main

import (
	"fmt"
	"os"
	"time"

	"github.com/user/goui/internal/animation"
	"github.com/user/goui/internal/canvas"
	"github.com/user/goui/internal/render"
	"github.com/user/goui/internal/types"
	"github.com/user/goui/internal/widget"
)

// settleOverlayAnim 把浮层进场动画推进到终态。
// headless 单帧渲染没有主循环驱动动画，不推进则捕获的是淡入起点(enterProgress=0)=透明。
func settleOverlayAnim() {
	base := time.Now()
	animation.Tick(base) // 首次仅初始化时间基准
	for i := 1; i <= 16; i++ {
		animation.Tick(base.Add(time.Duration(i) * 20 * time.Millisecond))
	}
}

const (
	cw = 620
	ch = 460
)

func main() {
	sk := canvas.NewSkiaCanvas(cw, ch)
	defer sk.Release()

	// 主内容
	bg := widget.Div(
		widget.Style{Width: cw, Height: ch, BackgroundColor: types.ColorRef(245, 247, 250), Padding: types.EdgeInsets(24)},
		widget.H2("应用主界面"),
		widget.P("通过 ShowDialog 把对话框挂到 OverlayHost 顶层弹出。"),
		widget.P("Overlay 的 Fullscreen 模式让模态铺满整个视口。"),
	)

	// OverlayHost 包裹主内容(模拟 app.Run 的顶层结构)
	host := widget.NewOverlayHost(bg)

	// 命令式弹出对话框(Fullscreen overlay)
	widget.ShowDialog(
		widget.NewDialog("提示",
			widget.P("此操作将永久删除该文件，是否继续？"),
		).WithFooter(
			widget.NewButton("取消", nil).
				WithColor(types.ColorFromRGB(244, 244, 245)).
				WithTextColor(types.ColorFromRGB(96, 98, 102)),
			widget.NewButton("确定", nil).
				WithColor(types.ColorFromRGB(64, 158, 255)),
		),
	)

	settleOverlayAnim()
	rootEl := widget.CreateElementFor(host)
	pipe := render.NewPipeline(cw, ch, sk)
	pipe.SetRootElement(rootEl)
	if err := pipe.Render(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if err := sk.SaveToPNG("modal_demo.png"); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	fmt.Println("✅ modal_demo.png 已保存")
}
