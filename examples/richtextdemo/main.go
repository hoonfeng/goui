// richtextdemo — 验证 WYSIWYG 富文本渲染：混排粗体/斜体/下划线 + 标题/列表/引用。
// 编辑/选中加格式/工具栏需窗口实测；headless 验证带样式文档的渲染。
// 运行: $env:CGO_ENABLED='1'; go run ./examples/richtextdemo/
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
	cw = 620
	ch = 460
)

func main() {
	sk := canvas.NewSkiaCanvas(cw, ch)
	defer sk.Release()

	red := types.ColorFromRGB(245, 108, 108)
	green := types.ColorFromRGB(103, 194, 58)
	blue := types.ColorFromRGB(64, 158, 255)
	rt := widget.NewRichText("").WithBlocks([]widget.RichBlock{
		{Type: "h1", Spans: []widget.RichSpan{{Text: "富文本编辑器（WYSIWYG）"}}},
		{Type: "p", Spans: []widget.RichSpan{
			{Text: "这一行混排了 "},
			{Text: "粗体", Bold: true},
			{Text: "、"},
			{Text: "斜体", Italic: true},
			{Text: "、"},
			{Text: "下划线", Underline: true},
			{Text: "，还有 "},
			{Text: "粗斜体", Bold: true, Italic: true},
			{Text: " 组合。"},
		}},
		{Type: "h2", Spans: []widget.RichSpan{{Text: "二级标题"}}},
		{Type: "ul", Spans: []widget.RichSpan{{Text: "无序列表项"}}},
		{Type: "ol", Spans: []widget.RichSpan{{Text: "有序列表项"}}},
		{Type: "quote", Spans: []widget.RichSpan{{Text: "这是一段引用文字"}}},
		{Type: "p", Spans: []widget.RichSpan{
			{Text: "字体颜色："},
			{Text: "红色", Color: &red},
			{Text: " "},
			{Text: "绿色", Color: &green},
			{Text: " "},
			{Text: "蓝色加粗", Bold: true, Color: &blue},
		}},
	}).WithSize(580, 430)

	root := widget.Div(
		widget.Style{Width: cw, Height: ch, Padding: types.EdgeInsets(20), BackgroundColor: types.ColorRef(255, 255, 255)},
		rt,
	)

	el := widget.CreateElementFor(root)
	pipe := render.NewPipeline(cw, ch, sk)
	pipe.SetRootElement(el)
	if err := pipe.Render(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if err := sk.SaveToPNG("richtext_demo.png"); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	fmt.Println("✅ richtext_demo.png 已保存")
}
