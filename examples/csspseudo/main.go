// csspseudo — CSS 伪类 :hover / :focus / :active headless 验证
//
// 一个填满画布的 Div，设三档伪类样式；分别在默认 / 悬停 / 聚焦 / 按下状态下渲染，
// 程序内解码每张 PNG 读中心像素 RGB 自证伪类生效（背景色按状态切换）。
//
//   $env:CGO_ENABLED='1'; go run ./examples/csspseudo/
//
//go:build !ignore
// +build !ignore

package main

import (
	"fmt"
	"image/png"
	"os"

	"github.com/hoonfeng/goui/pkg/canvas"
	"github.com/hoonfeng/goui/pkg/event"
	"github.com/hoonfeng/goui/pkg/render"
	"github.com/hoonfeng/goui/pkg/types"
	"github.com/hoonfeng/goui/pkg/widget"
)

const (
	W = 360
	H = 240
)

// box 一个填满画布、设了三档伪类的 Div。
func box() widget.Widget {
	base := types.ColorRef(235, 238, 245) // 浅灰蓝
	hov := types.ColorRef(64, 158, 255)   // el 主色蓝
	foc := types.ColorRef(103, 194, 58)   // 成功绿
	act := types.ColorRef(245, 108, 108)  // 危险红
	ring := types.ColorRef(48, 120, 200)
	return widget.Div(widget.Style{
		Width:           float64(W),
		Height:          float64(H),
		BackgroundColor: base,
		BorderRadius:    14,
		Hover:           &widget.Style{BackgroundColor: hov, BorderColor: ring, BorderWidth: 5},
		Focus:           &widget.Style{BackgroundColor: foc, BorderColor: ring, BorderWidth: 5},
		Active:          &widget.Style{BackgroundColor: act},
	})
}

func findCE(el widget.Element) *widget.ContainerElement {
	if ce, ok := el.(*widget.ContainerElement); ok {
		return ce
	}
	for _, c := range el.Children() {
		if r := findCE(c); r != nil {
			return r
		}
	}
	return nil
}

func renderState(name string, apply func(*widget.ContainerElement)) (uint32, uint32, uint32) {
	sk := canvas.NewSkiaCanvas(W, H)
	defer sk.Release()
	rootEl := widget.CreateElementFor(box())
	pipe := render.NewPipeline(W, H, sk)
	pipe.SetRootElement(rootEl)
	if err := pipe.Render(); err != nil { // 首次布局 + 绘制（默认态）
		fmt.Fprintln(os.Stderr, "render:", err)
		os.Exit(1)
	}
	if apply != nil {
		if ce := findCE(rootEl); ce != nil {
			apply(ce)
		}
		pipe.MarkNeedsRepaint()
		_ = pipe.Render() // 重绘该状态
	}
	path := "csspseudo_" + name + ".png"
	if err := sk.SaveToPNG(path); err != nil {
		fmt.Fprintln(os.Stderr, "save:", err)
		os.Exit(1)
	}
	return centerRGB(path)
}

// centerRGB 解码 PNG，读中心像素 0~255 RGB。
func centerRGB(path string) (uint32, uint32, uint32) {
	f, err := os.Open(path)
	if err != nil {
		return 0, 0, 0
	}
	defer f.Close()
	img, err := png.Decode(f)
	if err != nil {
		return 0, 0, 0
	}
	r, g, b, _ := img.At(W/2, H/2).RGBA()
	return r >> 8, g >> 8, b >> 8
}

func main() {
	ev := func(t event.Type) event.Event { return event.NewMouseEvent(t, 0, 0, event.ButtonLeft, 0) }

	dr, dg, db := renderState("default", nil)
	hr, hg, hb := renderState("hover", func(ce *widget.ContainerElement) {
		ce.HandleEvent(ev(event.TypeMouseEnter))
	})
	fr, fg, fb := renderState("focus", func(ce *widget.ContainerElement) {
		ce.Focus()
	})
	ar, ag, ab := renderState("active", func(ce *widget.ContainerElement) {
		ce.HandleEvent(ev(event.TypeMouseEnter))
		ce.HandleEvent(ev(event.TypeMouseDown))
	})

	fmt.Println("中心像素 RGB（期望：default 浅灰 / hover 蓝 / focus 绿 / active 红）:")
	fmt.Printf("  default = (%3d,%3d,%3d)  期望≈(235,238,245)\n", dr, dg, db)
	fmt.Printf("  hover   = (%3d,%3d,%3d)  期望≈( 64,158,255)\n", hr, hg, hb)
	fmt.Printf("  focus   = (%3d,%3d,%3d)  期望≈(103,194, 58)\n", fr, fg, fb)
	fmt.Printf("  active  = (%3d,%3d,%3d)  期望≈(245,108,108)\n", ar, ag, ab)

	ok := hb > 200 && hr < 120 && // hover 偏蓝
		fg > 150 && fr < 150 && fb < 120 && // focus 偏绿
		ar > 200 && ag < 160 && ab < 160 // active 偏红
	if ok {
		fmt.Println("✅ 四态背景色均按 CSS 伪类正确切换")
	} else {
		fmt.Println("⚠️  某些状态颜色不符预期，请核对")
	}
}
