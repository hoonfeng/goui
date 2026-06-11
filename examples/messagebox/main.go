// messagebox — 验证 batch② 反馈提示组件：MessageBox 消息弹框(el-messagebox 1:1)。
// 复用 Dialog 模态体系；此处 headless 渲染 ShowConfirm 的居中确认框外观。
//
// 运行: $env:CGO_ENABLED='1'; go run ./examples/messagebox/
package main

import (
	"fmt"
	"os"
	"time"

	"github.com/hoonfeng/goui/pkg/animation"
	"github.com/hoonfeng/goui/pkg/canvas"
	"github.com/hoonfeng/goui/pkg/render"
	"github.com/hoonfeng/goui/pkg/types"
	"github.com/hoonfeng/goui/pkg/widget"
)

// settleOverlayAnim 把浮层进场动画推进到终态(headless 单帧无主循环驱动)。
func settleOverlayAnim() {
	base := time.Now()
	animation.Tick(base)
	for i := 1; i <= 16; i++ {
		animation.Tick(base.Add(time.Duration(i) * 20 * time.Millisecond))
	}
}

const (
	cw = 620
	ch = 400
)

func main() {
	sk := canvas.NewSkiaCanvas(cw, ch)
	defer sk.Release()

	bg := widget.Div(
		widget.Style{Width: cw, Height: ch, BackgroundColor: types.ColorRef(245, 247, 250), Padding: types.EdgeInsets(24)},
		widget.H2("应用主界面"),
		widget.P("命令式 ShowConfirm 弹出居中确认框，复用 Dialog 模态体系 + 类型图标。"),
	)

	host := widget.NewOverlayHost(bg)
	widget.ShowConfirm("提示", "此操作将永久删除该文件，是否继续？", widget.MsgWarning, nil, nil)

	settleOverlayAnim()
	rootEl := widget.CreateElementFor(host)
	pipe := render.NewPipeline(cw, ch, sk)
	pipe.SetRootElement(rootEl)
	if err := pipe.Render(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if err := sk.SaveToPNG("messagebox_demo.png"); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	fmt.Println("✅ messagebox_demo.png 已保存")
}
