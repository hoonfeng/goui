// switch — 验证 batch② 表单输入组件：Switch 开关(el-switch 1:1)。
// 滑动动画需窗口实测；此处 headless 验证各状态静态呈现。
//
// 运行: $env:CGO_ENABLED='1'; go run ./examples/switch/
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
	cw = 540
	ch = 340
)

// rowC 横向居中排列一行控件。
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
	green := types.ColorFromRGB(103, 194, 58)

	root := widget.Div(
		widget.Style{Width: cw, Height: ch, Padding: types.EdgeInsets(24), BackgroundColor: types.ColorRef(255, 255, 255), FlexDirection: "column", Gap: 18},
		widget.H3("Switch 开关（el-switch 1:1）"),
		rowC(widget.NewSwitch(false, nil), widget.NewSwitch(true, nil), widget.NewSwitch(true, nil).WithLabel("带标签")),
		rowC(widget.NewSwitch(false, nil).WithDisabled(true), widget.NewSwitch(true, nil).WithDisabled(true), widget.P("← 禁用态")),
		rowC(widget.NewSwitch(true, nil).WithSize("large"), widget.NewSwitch(true, nil), widget.NewSwitch(true, nil).WithSize("small"), widget.P("← large / default / small")),
		rowC(widget.NewSwitch(true, nil).WithTexts("开", "关"), widget.NewSwitch(false, nil).WithTexts("ON", "OFF"), widget.P("← 轨道内文字")),
		rowC(widget.NewSwitch(true, nil).WithActiveColor(green), widget.P("← 自定义色")),
	)

	rootEl := widget.CreateElementFor(root)
	pipe := render.NewPipeline(cw, ch, sk)
	pipe.SetRootElement(rootEl)
	if err := pipe.Render(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if err := sk.SaveToPNG("switch_demo.png"); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	fmt.Println("✅ switch_demo.png 已保存")
}
