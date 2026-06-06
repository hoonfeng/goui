// rate — 验证 batch② 表单输入组件：Rate 评分(el-rate 1:1)。
// hover 预览/点选需窗口实测；此处 headless 验证各分值静态呈现。
//
// 运行: $env:CGO_ENABLED='1'; go run ./examples/rate/
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
	cw = 560
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
	red := types.ColorFromRGB(245, 108, 108)

	root := widget.Div(
		widget.Style{Width: cw, Height: ch, Padding: types.EdgeInsets(24), BackgroundColor: types.ColorRef(255, 255, 255), FlexDirection: "column", Gap: 18},
		widget.H3("Rate 评分（el-rate 1:1）"),
		rowC(widget.NewRate(3, nil), widget.P("← value = 3")),
		rowC(widget.NewRate(3.5, nil).WithAllowHalf(true), widget.P("← 半星 value = 3.5")),
		rowC(widget.NewRate(4, nil).WithSize("large"), widget.NewRate(4, nil), widget.NewRate(4, nil).WithSize("small"), widget.P("← large / default / small")),
		rowC(widget.NewRate(4, nil).WithDisabled(true), widget.P("← 只读")),
		rowC(widget.NewRate(3, nil).WithActiveColor(red), widget.NewRate(2, nil).WithMax(6), widget.P("← 自定义色 / max=6")),
	)

	rootEl := widget.CreateElementFor(root)
	pipe := render.NewPipeline(cw, ch, sk)
	pipe.SetRootElement(rootEl)
	if err := pipe.Render(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if err := sk.SaveToPNG("rate_demo.png"); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	fmt.Println("✅ rate_demo.png 已保存")
}
