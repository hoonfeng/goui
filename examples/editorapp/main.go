// editorapp — 文本编辑组件交互窗口：多行文本框 / Markdown 编辑器 / 富文本 RichText。
// 运行(开窗口): $env:CGO_ENABLED='1'; go run ./examples/editorapp/
package main

import (
	"log"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/hoonfeng/goui/pkg/app"
	"github.com/hoonfeng/goui/pkg/types"
	"github.com/hoonfeng/goui/pkg/widget"
)

// setupGoLSP 把代码写到临时 Go 模块，返回 gopls 路径 + 工作区/文件 URI（接 gopls 用）。
// gopls 未装则返回空 server（编辑器自动回退纯词法补全）。
func setupGoLSP(src string) (server, wsURI, fileURI string) {
	gopls := "gopls"
	if p := `F:\MyGolangPrograms\bin\gopls.exe`; fileExists(p) {
		gopls = p
	} else if _, err := os.Stat(gopls); err != nil {
		// 交给 PATH 解析；若 PATH 也无，NewClient 会失败→回退词法
	}
	dir, err := os.MkdirTemp("", "gouicode")
	if err != nil {
		return "", "", ""
	}
	os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module gouicode\n\ngo 1.24\n"), 0644)
	mainGo := filepath.Join(dir, "main.go")
	os.WriteFile(mainGo, []byte(src), 0644)
	return gopls, pathToURI(dir), pathToURI(mainGo)
}

func fileExists(p string) bool { _, err := os.Stat(p); return err == nil }
func pathToURI(p string) string {
	p = filepath.ToSlash(p)
	if !strings.HasPrefix(p, "/") {
		p = "/" + p
	}
	return "file://" + p
}

func main() {
	runtime.LockOSThread()

	gray := types.ColorFromRGB(96, 98, 102)
	content := widget.Div(
		widget.Style{Padding: types.EdgeInsets(24), BackgroundColor: types.ColorRef(245, 247, 250), FlexDirection: "column", Gap: 14},
		widget.H2("goui 文本编辑组件"),

		widget.NewText("① 多行文本框 Textarea（回车换行、↑↓ 移动、横向滚动、右键菜单）：", gray),
		longTextarea(),

		widget.NewText("② Markdown 展示（只渲染显示；要编辑用上面的多行框即可）：", gray),
		widget.NewMarkdownView("# Markdown 展示\n\n这是 **渲染** 后的效果，支持 *斜体* 和 `代码`\n\n- 无序列表项\n- 列表项二\n\n1. 有序列表项\n\n> 引用一段话"),

		widget.NewText("③ 富文本 RichText（WYSIWYG：粗斜体下划删除线 / 字号 / 颜色 / 对齐 / 插图）：", gray),
		widget.NewRichText("").WithBlocks([]widget.RichBlock{
			{Type: "h2", Align: "center", Spans: []widget.RichSpan{{Text: "试试富文本编辑（标题居中示例）"}}},
			{Type: "p", Spans: []widget.RichSpan{
				{Text: "选中文字点 "},
				{Text: "B / I / U / S", Bold: true},
				{Text: " 设粗斜下划"},
				{Text: "删除线", Strike: true},
				{Text: "；点「字号」可把某几个字设"},
				{Text: "大", Size: 28},
				{Text: "或"},
				{Text: "小", Size: 12},
				{Text: "；颜色按钮是分体的——"},
				{Text: "左边 A 套用当前色", Color: types.ColorRef(245, 108, 108)},
				{Text: "、右边 ▾ 弹出拾色器；点「左/中/右」改对齐，「图片」插本地图。"},
			}},
			{Type: "p", Align: "right", Spans: []widget.RichSpan{{Text: "—— 这一段右对齐"}}},
		}).WithSize(740, 340),

		widget.NewText("④ 代码编辑器 CodeEditor（gopls 语义补全 + 诊断；打 fmt. 试试）：", gray),
		goCodeEditor(),

		widget.NewText("⑤ 结构化编辑器 StructEditor（易语言式表格：点格编辑、Tab/方向键移格、「+ 增行」、Ctrl+Enter 增行 / Ctrl+Delete 删行）：", gray),
		widget.NewStructEditor(widget.DemoFactorialProgram()).WithSize(740, 460),
	)

	application := app.NewApplication()
	application.SetRootWidget(widget.NewScrollView(content))

	config := app.DefaultConfig()
	config.Title = "goui 文本编辑组件 — Textarea / Markdown / RichText"
	config.Width = 820
	config.Height = 800
	config.Resizable = true

	log.Println("editorapp: 启动文本编辑组件窗口")
	if err := application.Run(config); err != nil {
		log.Fatalf("editorapp: %v", err)
	}
}

// goCodeEditor 创建接 gopls 的代码编辑器（内容写临时 Go 模块；gopls 未装则回退词法补全）。
func goCodeEditor() *widget.CodeEditor {
	ed := widget.NewCodeEditor("go", goSample).WithSize(740, 300)
	ed.IndentGuides = true // 缩进连线（虚实结合：当前块那条变实线）
	if server, ws, file := setupGoLSP(goSample); server != "" {
		ed = ed.WithLSP(server, ws, file)
	}
	return ed
}

// goSample 用于演示语法高亮的 Go 代码示例。
const goSample = `package main

import "fmt"

// greet 返回问候语（行注释示例）
func greet(name string) string {
	/* 块注释：
	   多行也能正确高亮 */
	prefix := ` + "`" + `Hello, ` + "`" + `
	return prefix + name + "!"
}

func main() {
	nums := []int{1, 2, 3, 42}
	total := 0
	for _, n := range nums {
		total += n // 累加
	}
	fmt.Println(greet("goui"), total)
}
`

// longTextarea 预填一行超长文本（演示横向滚动）+ 多行（纵向滚动）；右键弹默认菜单。
func longTextarea() *widget.Input {
	ta := widget.NewTextarea("在这里输入多行内容，按回车换行…", 4, nil)
	ta.Text = "这是一行很长很长的文本用来演示横向滚动——它不会自动换行，所以右边超出可视区后可以用 Shift+滚轮 或拖底部滚动条横向查看后面的内容 the quick brown fox jumps over the lazy dog 1234567890\n第二行\n第三行\n第四行\n第五行（纵向也能滚）\n第六行 右键我试试 复制/粘贴/全选 菜单"
	return ta
}
