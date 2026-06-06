// compfocus — 组件级焦点态（Input 聚焦边框 / Button focus ring）headless 验证
//
// 用「两帧差异像素数」证明焦点态真正改变了渲染输出：
//   · Button：default vs focused —— focus ring 出现 → 有差异
//   · Input ：聚焦默认色(蓝) vs WithFocusColor(红) —— 聚焦色可配 → 有差异
//
//   $env:CGO_ENABLED='1'; go run ./examples/compfocus/
//
//go:build !ignore
// +build !ignore

package main

import (
	"fmt"
	"image"
	"image/png"
	"os"

	"github.com/user/goui/internal/canvas"
	"github.com/user/goui/internal/event"
	"github.com/user/goui/internal/render"
	"github.com/user/goui/internal/types"
	"github.com/user/goui/internal/widget"
)

const (
	W = 200
	H = 90
)

func findBtn(el widget.Element) *widget.ButtonElement {
	if b, ok := el.(*widget.ButtonElement); ok {
		return b
	}
	for _, c := range el.Children() {
		if r := findBtn(c); r != nil {
			return r
		}
	}
	return nil
}

func findInput(el widget.Element) *widget.InputElement {
	if b, ok := el.(*widget.InputElement); ok {
		return b
	}
	for _, c := range el.Children() {
		if r := findInput(c); r != nil {
			return r
		}
	}
	return nil
}

func findSelect(el widget.Element) *widget.SelectElement {
	if s, ok := el.(*widget.SelectElement); ok {
		return s
	}
	for _, c := range el.Children() {
		if r := findSelect(c); r != nil {
			return r
		}
	}
	return nil
}

func render2(root widget.Widget, focus func(widget.Element), path string) image.Image {
	sk := canvas.NewSkiaCanvas(W, H)
	defer sk.Release()
	rootEl := widget.CreateElementFor(root)
	pipe := render.NewPipeline(W, H, sk)
	pipe.SetRootElement(rootEl)
	_ = pipe.Render()
	if focus != nil {
		focus(rootEl)
		pipe.MarkNeedsRepaint()
		_ = pipe.Render()
	}
	_ = sk.SaveToPNG(path)
	f, err := os.Open(path)
	if err != nil {
		return nil
	}
	defer f.Close()
	img, _ := png.Decode(f)
	return img
}

func diffCount(a, b image.Image) int {
	if a == nil || b == nil {
		return -1
	}
	n := 0
	for y := 0; y < H; y++ {
		for x := 0; x < W; x++ {
			ar, ag, ab, _ := a.At(x, y).RGBA()
			br, bg, bb, _ := b.At(x, y).RGBA()
			if ar != br || ag != bg || ab != bb {
				n++
			}
		}
	}
	return n
}

func main() {
	white := types.ColorRef(255, 255, 255)
	red := types.ColorRef(245, 108, 108)

	// Button：default vs focused（focus ring 出现）
	btnRoot := func() widget.Widget {
		return widget.Div(widget.Style{Width: float64(W), Height: float64(H), BackgroundColor: white},
			widget.Div(widget.Style{Padding: types.EdgeInsets(24)},
				&widget.Button{Text: "按钮", Color: types.ColorFromRGB(64, 158, 255), MinWidth: 100, MinHeight: 38},
			),
		)
	}
	bDef := render2(btnRoot(), nil, "compfocus_btn_default.png")
	bFoc := render2(btnRoot(), func(el widget.Element) {
		if be := findBtn(el); be != nil {
			be.Focus()
		}
	}, "compfocus_btn_focus.png")
	bd := diffCount(bDef, bFoc)

	// Input：聚焦默认色(主题蓝) vs WithFocusColor(红)
	mkInput := func(focusColor *types.Color) widget.Widget {
		in := widget.NewInput("输入...", nil)
		if focusColor != nil {
			in.WithFocusColor(*focusColor)
		}
		return widget.Div(widget.Style{Width: float64(W), Height: float64(H), BackgroundColor: white},
			widget.Div(widget.Style{Padding: types.EdgeInsets(24)}, in),
		)
	}
	focusInput := func(el widget.Element) {
		if ie := findInput(el); ie != nil {
			ie.Focus()
		}
	}
	iBlue := render2(mkInput(nil), focusInput, "compfocus_input_blue.png")
	iRed := render2(mkInput(red), focusInput, "compfocus_input_red.png")
	id := diffCount(iBlue, iRed)

	// Select（触发器类）：hover 边框默认(主题蓝) vs SetHoverBorderColor(红)
	mkSelect := func(redBorder bool) widget.Widget {
		sel := &widget.Select{Placeholder: "选择", Options: []widget.SelectOption{{Label: "A", Value: "a"}}}
		if redBorder {
			sel.SetHoverBorderColor(*red)
		}
		return widget.Div(widget.Style{Width: float64(W), Height: float64(H), BackgroundColor: white},
			widget.Div(widget.Style{Padding: types.EdgeInsets(24)}, sel),
		)
	}
	hoverSelect := func(el widget.Element) {
		if se := findSelect(el); se != nil {
			se.HandleEvent(event.NewMouseEvent(event.TypeMouseEnter, 0, 0, event.ButtonLeft, 0))
		}
	}
	sBlue := render2(mkSelect(false), hoverSelect, "compfocus_select_blue.png")
	sRed := render2(mkSelect(true), hoverSelect, "compfocus_select_red.png")
	sd := diffCount(sBlue, sRed)

	fmt.Printf("Button focus ring 渲染差异像素: %d  (应>0：聚焦环出现)\n", bd)
	fmt.Printf("Input 聚焦边框 蓝 vs 红 差异像素: %d  (应>0：聚焦色可配)\n", id)
	fmt.Printf("Select 触发器 hover 边框 蓝 vs 红 差异像素: %d  (应>0：触发器伪类生效)\n", sd)
	if bd > 0 && id > 0 && sd > 0 {
		fmt.Println("✅ Button 聚焦环 + Input 聚焦色 + Select 触发器 hover 色 均生效")
	} else {
		fmt.Println("⚠️  某项无差异，请核对")
	}
}
