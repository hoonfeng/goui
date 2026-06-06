// textareademo — 验证多行文本框 Textarea（el-input type=textarea）。
// 编辑/换行/上下键需窗口实测；headless 验证多行渲染 + placeholder。
// 运行: $env:CGO_ENABLED='1'; go run ./examples/textareademo/
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
	ch = 400
)

func main() {
	sk := canvas.NewSkiaCanvas(cw, ch)
	defer sk.Release()

	ta := widget.NewTextarea("请输入多行内容…", 3, nil)
	ta.Text = "第一行：goui 多行文本框 textarea\n第二行：回车换行、↑↓ 移动光标\n第三行：复用 Input 的 IME / 选择\n第四行：内容超过可见行数\n第五行：右侧出现垂直滚动条\n第六行：滚轮可上下滚动\n第七行：打字时光标自动跟随"

	empty := widget.NewTextarea("请输入备注（占位提示）…", 3, nil)
	gray := types.ColorFromRGB(96, 98, 102)

	root := widget.Div(
		widget.Style{Width: cw, Height: ch, Padding: types.EdgeInsets(20), BackgroundColor: types.ColorRef(255, 255, 255), FlexDirection: "column", Gap: 10},
		widget.H3("多行文本框 Textarea"),
		widget.NewText("预填多行（可编辑）：", gray),
		ta,
		widget.NewText("空白（显示占位）：", gray),
		empty,
	)

	el := widget.CreateElementFor(root)
	pipe := render.NewPipeline(cw, ch, sk)
	pipe.SetRootElement(el)
	if err := pipe.Render(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if err := sk.SaveToPNG("textarea_demo.png"); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	fmt.Println("✅ textarea_demo.png 已保存")
}
