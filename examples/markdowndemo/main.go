// markdowndemo — 验证 MarkdownView 块级渲染（标题/列表/引用/分隔线/代码块/段落）。
// 运行: $env:CGO_ENABLED='1'; go run ./examples/markdowndemo/
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
	ch = 640
)

func main() {
	sk := canvas.NewSkiaCanvas(cw, ch)
	defer sk.Release()

	md := "# Markdown 渲染演示\n" +
		"## 二级标题\n" +
		"这是普通段落，包含 **加粗** 和 *斜体* 和 `代码` 标记。\n" +
		"\n" +
		"- 无序列表项一\n" +
		"- 无序列表项二\n" +
		"\n" +
		"1. 有序第一条\n" +
		"2. 有序第二条\n" +
		"\n" +
		"> 这是一段引用文字\n" +
		"\n" +
		"---\n" +
		"\n" +
		"```\n" +
		"func main() {\n" +
		"    println(\"hello\")\n" +
		"}\n" +
		"```"

	root := widget.Div(
		widget.Style{Width: cw, Height: ch, Padding: types.EdgeInsets(20), BackgroundColor: types.ColorRef(255, 255, 255), FlexDirection: "column", Gap: 10},
		widget.NewMarkdownView(md),
	)

	el := widget.CreateElementFor(root)
	pipe := render.NewPipeline(cw, ch, sk)
	pipe.SetRootElement(el)
	if err := pipe.Render(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if err := sk.SaveToPNG("markdown_demo.png"); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	fmt.Println("✅ markdown_demo.png 已保存")
}
