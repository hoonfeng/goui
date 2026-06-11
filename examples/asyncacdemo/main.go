// asyncacdemo — 验证异步 Autocomplete：loading 态 + 结果态外观。
// 异步运行流程(防抖→fetch→回填)需窗口实测；headless 验证两态视觉。
// 运行: $env:CGO_ENABLED='1'; go run ./examples/asyncacdemo/
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
	cw = 300
	ch = 420
)

func main() {
	sk := canvas.NewSkiaCanvas(cw, ch)
	defer sk.Release()

	gray := types.ColorFromRGB(96, 98, 102)

	// 异步输入框：输入触发远程 fetch（此处同步返回模拟；真实用 go func 异步 + done 回填）
	ac := widget.NewAutocomplete().WithFetch(func(q string, done func([]string)) {
		done([]string{q + " — 远程结果 A", q + " — 远程结果 B"})
	}).WithDebounce(300)
	ac.Width = 260
	ac.Placeholder = "输入触发远程搜索…"

	loadingDD := widget.AutocompleteDropdownPreview(nil, true, 260)
	resultDD := widget.AutocompleteDropdownPreview([]string{"北京市", "北京大学", "北京路"}, false, 260)

	root := widget.Div(
		widget.Style{Width: cw, Height: ch, Padding: types.EdgeInsets(20), BackgroundColor: types.ColorRef(255, 255, 255), FlexDirection: "column", Gap: 8},
		widget.H3("异步 Autocomplete"),
		widget.NewText("远程 fetch + loading + 防抖", gray),
		widget.NewText("输入框：", gray),
		ac,
		widget.NewText("加载中态（防抖触发后）：", gray),
		loadingDD,
		widget.NewText("结果态：", gray),
		resultDD,
	)

	rootEl := widget.CreateElementFor(root)
	pipe := render.NewPipeline(cw, ch, sk)
	pipe.SetRootElement(rootEl)
	if err := pipe.Render(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if err := sk.SaveToPNG("asyncac_demo.png"); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	fmt.Println("✅ asyncac_demo.png 已保存")
}
