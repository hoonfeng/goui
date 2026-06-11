// notification — 验证 batch② 反馈提示组件：Notification 通知(el-notification 1:1)。
// 命令式右上角弹出/自动消失需窗口实测；此处 headless 预览四种类型卡片外观。
//
// 运行: $env:CGO_ENABLED='1'; go run ./examples/notification/
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
	cw = 420
	ch = 440
)

func main() {
	sk := canvas.NewSkiaCanvas(cw, ch)
	defer sk.Release()

	root := widget.Div(
		widget.Style{Width: cw, Height: ch, Padding: types.EdgeInsets(24), BackgroundColor: types.ColorRef(245, 247, 250), FlexDirection: "column", Gap: 12},
		widget.H3("Notification 通知（el-notification 1:1）"),
		widget.P("命令式 NotifySuccess(标题, 内容) 右上角浮出、4.5s 自动消失:"),
		widget.NotificationPreview("成功", "数据已成功保存到云端", widget.MsgSuccess),
		widget.NotificationPreview("警告", "磁盘空间即将不足", widget.MsgWarning),
		widget.NotificationPreview("通知", "您有 3 条新消息待查看", widget.MsgInfo),
		widget.NotificationPreview("错误", "连接服务器失败，请重试", widget.MsgError),
	)

	rootEl := widget.CreateElementFor(root)
	pipe := render.NewPipeline(cw, ch, sk)
	pipe.SetRootElement(rootEl)
	if err := pipe.Render(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if err := sk.SaveToPNG("notification_demo.png"); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	fmt.Println("✅ notification_demo.png 已保存")
}
