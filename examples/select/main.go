// select — 验证 batch② 表单输入组件：Select 下拉选择器(el-select 1:1)。
// 点击弹下拉需窗口实测；此处 headless 验证触发器各状态 + 下拉面板外观预览。
//
// 运行: $env:CGO_ENABLED='1'; go run ./examples/select/
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
	cw = 640
	ch = 480
)

func rowC(kids ...widget.Widget) widget.Widget {
	args := []interface{}{widget.Style{FlexDirection: "row", AlignItems: "center", Gap: 16}}
	for _, k := range kids {
		args = append(args, k)
	}
	return widget.Div(args...)
}

func main() {
	sk := canvas.NewSkiaCanvas(cw, ch)
	defer sk.Release()

	opts := []widget.SelectOption{
		{Label: "北京", Value: "bj"},
		{Label: "上海", Value: "sh"},
		{Label: "广州（禁用）", Value: "gz", Disabled: true},
		{Label: "深圳", Value: "sz"},
	}

	root := widget.Div(
		widget.Style{Width: cw, Height: ch, Padding: types.EdgeInsets(24), BackgroundColor: types.ColorRef(255, 255, 255), FlexDirection: "column", Gap: 14},
		widget.H3("Select 下拉选择器（el-select 1:1）"),
		rowC(widget.NewSelect(opts), widget.P("← placeholder（请选择）")),
		rowC(widget.NewSelect(opts).WithValue("sh"), widget.P("← 已选「上海」")),
		rowC(widget.NewSelect(opts).WithValue("bj").WithDisabled(true), widget.P("← 禁用")),
		rowC(widget.NewSelect(opts).WithSize("small").WithWidth(160), widget.NewSelect(opts).WithSize("large").WithWidth(200), widget.P("← small / large")),
		widget.Div(widget.Style{Height: 8}),
		widget.P("下拉展开预览（选中「上海」，含 hover 底/选中✓/禁用项）:"),
		widget.SelectDropdownPreview(opts, "sh", 220),
	)

	rootEl := widget.CreateElementFor(root)
	pipe := render.NewPipeline(cw, ch, sk)
	pipe.SetRootElement(rootEl)
	if err := pipe.Render(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if err := sk.SaveToPNG("select_demo.png"); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	fmt.Println("✅ select_demo.png 已保存")
}
