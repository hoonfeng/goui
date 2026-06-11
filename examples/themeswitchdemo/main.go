// themeswitchdemo — 验证配置化"一处换肤全跟随"：同一组件树渲染两次，
// 第二次 SetPrimaryColor 切主色，主色相关组件应整体跟随变色。
// 运行: $env:CGO_ENABLED='1'; go run ./examples/themeswitchdemo/
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
	cw = 420
	ch = 340
)

// panel 每次调用新建组件树（读当时主题），便于换肤后重建跟随。
func panel(title string) widget.Widget {
	opts := []widget.SelectOption{{Label: "研发部", Value: "a"}, {Label: "设计部", Value: "b"}}
	return widget.Div(
		widget.Style{Width: cw, Height: ch, Padding: types.EdgeInsets(20), BackgroundColor: types.ColorRef(255, 255, 255), FlexDirection: "column", Gap: 14},
		widget.H3(title),
		widget.NewCheckbox("已勾选复选框", true, func(bool) {}),
		widget.Div(widget.Style{FlexDirection: "row", Gap: 8},
			widget.NewTag("Primary").WithType(widget.TagPrimary),
			widget.NewTag("Success").WithType(widget.TagSuccess),
			widget.NewTag("Danger").WithType(widget.TagDanger),
		),
		widget.NewSelect(opts).WithMultiple(true).WithValues([]string{"a", "b"}).WithWidth(300),
		widget.NewButton("主要按钮", func() {}),
	)
}

func renderPanel(title, name string) {
	sk := canvas.NewSkiaCanvas(cw, ch)
	defer sk.Release()
	el := widget.CreateElementFor(panel(title))
	pipe := render.NewPipeline(cw, ch, sk)
	pipe.SetRootElement(el)
	if err := pipe.Render(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if err := sk.SaveToPNG(name); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func main() {
	renderPanel("默认主题（el 蓝 #409EFF）", "theme_default.png")

	widget.SetPrimaryColor(types.ColorFromRGB(130, 80, 223)) // 切换品牌紫
	renderPanel("SetPrimaryColor(紫) 后", "theme_brand.png")

	fmt.Println("✅ theme_default.png + theme_brand.png 已保存")
}
