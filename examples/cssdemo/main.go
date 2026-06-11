// cssdemo — 验证扩展 CSS 属性：渐变 / 阴影 / 透明度 / 字重，全部用声明式 Style 配置。
//
// 运行: $env:CGO_ENABLED='1'; go run ./examples/cssdemo/
package main

import (
	"fmt"
	"os"

	"github.com/hoonfeng/goui/pkg/canvas"
	"github.com/hoonfeng/goui/pkg/paint"
	"github.com/hoonfeng/goui/pkg/render"
	"github.com/hoonfeng/goui/pkg/types"
	"github.com/hoonfeng/goui/pkg/widget"
)

const (
	cw = 720
	ch = 520
)

func card(style widget.Style, label string, textColor types.Color) widget.Widget {
	return widget.Div(
		widget.Merge(style, widget.Style{Margin: types.EdgeInsetsLTRB(0, 14, 0, 0)}),
		widget.NewText(label, textColor),
	)
}

func main() {
	sk := canvas.NewSkiaCanvas(cw, ch)
	defer sk.Release()

	grad := &paint.Gradient{
		Type:  paint.GradientLinear,
		Start: types.Point{X: 0, Y: 0}, End: types.Point{X: 1, Y: 1},
		Stops: []paint.ColorStop{
			{Offset: 0, Color: types.ColorFromRGB(64, 158, 255)},
			{Offset: 1, Color: types.ColorFromRGB(155, 89, 182)},
		},
	}
	shadow := &paint.Shadow{Offset: types.Point{X: 0, Y: 5}, Blur: 10, Color: types.ColorFromRGBA(0, 0, 0, 45)}
	half := 0.45

	root := widget.Div(
		widget.Style{Padding: types.EdgeInsets(24), BackgroundColor: types.ColorRef(245, 247, 250)},
		widget.H2("CSS 扩展属性 · 渐变 / 阴影 / 透明度 / 字重（纯 Style 配置）"),

		card(widget.Style{Width: 340, Height: 72, BorderRadius: 12, Gradient: grad, Padding: types.EdgeInsets(18)},
			"渐变背景 LinearGradient（蓝→紫）", types.ColorWhite),

		card(widget.Style{Width: 340, Height: 64, BorderRadius: 10, BackgroundColor: types.ColorRef(255, 255, 255), Shadow: shadow, Padding: types.EdgeInsets(18)},
			"盒阴影 BoxShadow", types.ColorFromRGB(33, 33, 33)),

		card(widget.Style{Width: 340, Height: 56, BorderRadius: 10, BackgroundColor: types.ColorRef(245, 108, 108), Opacity: &half, Padding: types.EdgeInsets(16)},
			"半透明 Opacity 0.45", types.ColorWhite),

		widget.Div(widget.Style{Margin: types.EdgeInsetsLTRB(0, 18, 0, 0)},
			widget.Styled(widget.NewText("常规字重 Normal 12345", types.ColorFromRGB(33, 33, 33)),
				widget.Style{FontSize: 18}),
		),
		widget.Div(widget.Style{Margin: types.EdgeInsetsLTRB(0, 4, 0, 0)},
			widget.Styled(widget.NewText("粗体 FontWeight Bold 12345", types.ColorFromRGB(33, 33, 33)),
				widget.Style{FontSize: 18, FontWeight: canvas.FontWeightBold}),
		),
	)

	rootEl := widget.CreateElementFor(root)
	pipe := render.NewPipeline(cw, ch, sk)
	pipe.SetRootElement(rootEl)
	if err := pipe.Render(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if err := sk.SaveToPNG("css_demo.png"); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	fmt.Println("✅ css_demo.png 已保存")
}
