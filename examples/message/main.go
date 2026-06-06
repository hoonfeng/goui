// message — 验证 batch② 反馈提示组件：Message 全局消息(el-message 1:1)。
// 命令式弹出/自动消失需窗口实测；此处 headless 预览四种类型消息条外观。
//
// 运行: $env:CGO_ENABLED='1'; go run ./examples/message/
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
	ch = 320
)

func main() {
	sk := canvas.NewSkiaCanvas(cw, ch)
	defer sk.Release()

	root := widget.Div(
		widget.Style{Width: cw, Height: ch, Padding: types.EdgeInsets(24), BackgroundColor: types.ColorRef(255, 255, 255), FlexDirection: "column", Gap: 12},
		widget.H3("Message 全局消息（el-message 1:1）"),
		widget.P("命令式 MessageSuccess(\"...\") 顶部居中浮出、3s 自动消失。四种类型外观:"),
		widget.MessageBarPreview("这是一条成功消息", widget.MsgSuccess),
		widget.MessageBarPreview("这是一条警告消息", widget.MsgWarning),
		widget.MessageBarPreview("这是一条信息消息", widget.MsgInfo),
		widget.MessageBarPreview("这是一条错误消息", widget.MsgError),
	)

	rootEl := widget.CreateElementFor(root)
	pipe := render.NewPipeline(cw, ch, sk)
	pipe.SetRootElement(rootEl)
	if err := pipe.Render(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if err := sk.SaveToPNG("message_demo.png"); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	fmt.Println("✅ message_demo.png 已保存")
}
