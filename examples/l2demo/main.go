// l2demo — L2 Flex 全套：direction / justify / align / gap / flex-grow，纯 Style 配置。
//
// 运行: $env:CGO_ENABLED='1'; go run ./examples/l2demo/
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
	ch = 640
)

func box(bg *types.Color, label string, h float64) widget.Widget {
	return widget.Div(
		widget.Style{Width: 84, Height: h, BackgroundColor: bg, BorderRadius: 6, Padding: types.EdgeInsets(8)},
		widget.NewText(label, types.ColorWhite),
	)
}

func expanded(child widget.Widget) widget.Widget {
	return &widget.Expanded{SingleChildWidget: widget.SingleChildWidget{Child: child}, Flex: 1}
}

func row(label, just, align string, gap float64, kids ...widget.Widget) widget.Widget {
	inner := []interface{}{
		widget.Style{
			FlexDirection: "row", JustifyContent: just, AlignItems: align, Gap: gap,
			BackgroundColor: types.ColorRef(235, 238, 242), BorderRadius: 6, Padding: types.EdgeInsets(8),
		},
	}
	for _, k := range kids {
		inner = append(inner, k)
	}
	return widget.Div(
		widget.Style{Margin: types.EdgeInsetsLTRB(0, 10, 0, 0)},
		widget.NewText(label, types.ColorFromRGB(120, 120, 120)),
		widget.Div(inner...),
	)
}

func main() {
	sk := canvas.NewSkiaCanvas(cw, ch)
	defer sk.Release()
	blue := types.ColorRef(64, 158, 255)
	green := types.ColorRef(103, 194, 58)
	orange := types.ColorRef(230, 162, 60)

	root := widget.Div(
		widget.Style{Padding: types.EdgeInsets(24), BackgroundColor: types.ColorRef(255, 255, 255)},
		widget.H2("L2 · Flex：direction / justify / align / gap / flex-grow"),
		row("row + gap 10", "start", "start", 10, box(blue, "1", 44), box(green, "2", 44), box(orange, "3", 44)),
		row("justify-content: center", "center", "start", 8, box(blue, "A", 44), box(green, "B", 44)),
		row("justify-content: space-between", "space-between", "start", 0, box(blue, "X", 44), box(green, "Y", 44), box(orange, "Z", 44)),
		row("align-items: center（不同高垂直居中）", "start", "center", 8, box(blue, "44", 44), box(green, "68", 68), box(orange, "32", 32)),
		row("flex-grow: Expanded 撑满中间", "start", "start", 8, box(blue, "固定", 44), expanded(box(green, "Expanded 填充", 44)), box(orange, "固定", 44)),
		row("Spacer: 推到两端", "start", "start", 0, box(blue, "左", 44), &widget.Spacer{}, box(orange, "右", 44)),
	)

	rootEl := widget.CreateElementFor(root)
	pipe := render.NewPipeline(cw, ch, sk)
	pipe.SetRootElement(rootEl)
	if err := pipe.Render(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if err := sk.SaveToPNG("l2_demo.png"); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	fmt.Println("✅ l2_demo.png 已保存")
}
