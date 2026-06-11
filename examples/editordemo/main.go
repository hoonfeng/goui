// editordemo — 验证 Markdown 编辑器 + 富文本 RichText（左编辑 · 右预览 · 工具栏）。
// 编辑/工具栏插入需窗口实测；headless 验证布局 + 预填内容的预览渲染。
// 运行: $env:CGO_ENABLED='1'; go run ./examples/editordemo/
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
	cw = 500
	ch = 580
)

func main() {
	sk := canvas.NewSkiaCanvas(cw, ch)
	defer sk.Release()

	mdSrc := "# Markdown 编辑器\n\n左边源码，右边预览\n\n- 实时渲染\n- 复用 CodeEditor"
	mdEditor := widget.Div( // 左 CodeEditor 编辑 · 右 MarkdownView 预览（NewMarkdown 已不存在，用现有控件拼分栏）
		widget.Style{FlexDirection: "row", Gap: 10, Height: 190},
		widget.NewCodeEditor("markdown", mdSrc).WithSize(215, 190),
		widget.Div(
			widget.Style{Width: 215, Height: 190, Padding: types.EdgeInsets(8), BackgroundColor: types.ColorRef(248, 249, 250), BorderRadius: 4},
			widget.NewMarkdownView(mdSrc),
		),
	)
	richEditor := widget.NewRichText("# 富文本\n\n点上方工具栏插入格式").WithSize(440, 160)

	root := widget.Div(
		widget.Style{Width: cw, Height: ch, Padding: types.EdgeInsets(20), BackgroundColor: types.ColorRef(255, 255, 255), FlexDirection: "column", Gap: 10},
		widget.H3("Markdown 编辑器（左编辑 · 右预览）"),
		mdEditor,
		widget.H3("富文本 RichText（工具栏 + 编辑 + 预览）"),
		richEditor,
	)

	el := widget.CreateElementFor(root)
	pipe := render.NewPipeline(cw, ch, sk)
	pipe.SetRootElement(el)
	if err := pipe.Render(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if err := sk.SaveToPNG("editor_demo.png"); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	fmt.Println("✅ editor_demo.png 已保存")
}
