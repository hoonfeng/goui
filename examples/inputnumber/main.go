// inputnumber — 验证 batch② 表单输入组件：InputNumber 数字输入框(el-input-number 1:1)。
// 点击加减需窗口实测；此处 headless 验证各状态静态呈现。
//
// 运行: $env:CGO_ENABLED='1'; go run ./examples/inputnumber/
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
	cw = 600
	ch = 360
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

	root := widget.Div(
		widget.Style{Width: cw, Height: ch, Padding: types.EdgeInsets(24), BackgroundColor: types.ColorRef(255, 255, 255), FlexDirection: "column", Gap: 16},
		widget.H3("InputNumber 数字输入框（el-input-number 1:1）"),
		rowC(widget.NewInputNumber(5, nil), widget.P("← 基础 value=5")),
		rowC(widget.NewInputNumber(0, nil).WithRange(0, 10), widget.P("← 范围 0~10（到下界 [−] 变灰）")),
		rowC(widget.NewInputNumber(3.14, nil).WithPrecision(2).WithStep(0.1), widget.P("← 精度2 / 步长0.1")),
		rowC(widget.NewInputNumber(5, nil).WithSize("large").WithWidth(140), widget.NewInputNumber(5, nil).WithSize("small").WithWidth(120), widget.P("← large / small")),
		rowC(widget.NewInputNumber(8, nil).WithDisabled(true), widget.P("← 禁用")),
	)

	rootEl := widget.CreateElementFor(root)
	pipe := render.NewPipeline(cw, ch, sk)
	pipe.SetRootElement(rootEl)
	if err := pipe.Render(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if err := sk.SaveToPNG("inputnumber_demo.png"); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	fmt.Println("✅ inputnumber_demo.png 已保存")
}
