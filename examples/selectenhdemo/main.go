// selectenhdemo — 验证 Select 增强：多选(tag) / 可清除 / 可搜索下拉。
// 运行: $env:CGO_ENABLED='1'; go run ./examples/selectenhdemo/
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
	cw = 340
	ch = 500
)

func main() {
	sk := canvas.NewSkiaCanvas(cw, ch)
	defer sk.Release()

	opts := []widget.SelectOption{
		{Label: "苹果", Value: "apple"},
		{Label: "香蕉", Value: "banana"},
		{Label: "橙子", Value: "orange"},
		{Label: "葡萄", Value: "grape"},
		{Label: "西瓜", Value: "watermelon"},
	}
	gray := types.ColorFromRGB(96, 98, 102)

	multiSel := widget.NewSelect(opts).WithMultiple(true).WithValues([]string{"apple", "orange"}).WithClearable(true).WithWidth(280)
	clearSel := widget.NewSelect(opts).WithValue("banana").WithClearable(true).WithWidth(280)
	ddPreview := widget.SelectDropdownPreviewMulti(opts, []string{"apple", "orange"}, true, 280)

	root := widget.Div(
		widget.Style{Width: cw, Height: ch, Padding: types.EdgeInsets(20), BackgroundColor: types.ColorRef(255, 255, 255), FlexDirection: "column", Gap: 8},
		widget.H3("Select 增强：多选 / 可搜索 / 可清除"),
		widget.NewText("多选（选中变 tag、点 × 删除）：", gray),
		multiSel,
		widget.NewText("可清除（hover 显示 × 清空）：", gray),
		clearSel,
		widget.NewText("下拉：多选打勾 + 顶部搜索框：", gray),
		ddPreview,
	)

	rootEl := widget.CreateElementFor(root)
	pipe := render.NewPipeline(cw, ch, sk)
	pipe.SetRootElement(rootEl)
	if err := pipe.Render(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if err := sk.SaveToPNG("selectenh_demo.png"); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	fmt.Println("✅ selectenh_demo.png 已保存")
}
