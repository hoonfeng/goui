// datepicker — 验证 batch④ 重型组件：DatePicker 日期选择器(el-date-picker 单选核心)。
// 浮层弹出需窗口实测；此处 headless 直接渲染触发器 + 展开的日历面板 preview。
//
// 运行: $env:CGO_ENABLED='1'; go run ./examples/datepicker/
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
	ch = 520
)

func main() {
	sk := canvas.NewSkiaCanvas(cw, ch)
	defer sk.Release()

	root := widget.Div(
		widget.Style{Width: cw, Height: ch, Padding: types.EdgeInsets(24), BackgroundColor: types.ColorRef(245, 247, 250), FlexDirection: "column", Gap: 16},
		widget.H3("DatePicker（点开定位今天·底部今天快捷）"),
		widget.NewDatePicker().WithPlaceholder("点开 → 定位今天"),
		widget.DatePanelPreview(""),
	)

	rootEl := widget.CreateElementFor(root)
	pipe := render.NewPipeline(cw, ch, sk)
	pipe.SetRootElement(rootEl)
	if err := pipe.Render(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if err := sk.SaveToPNG("datepicker_demo.png"); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	fmt.Println("✅ datepicker_demo.png 已保存")
}
