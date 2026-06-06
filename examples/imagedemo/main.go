// imagedemo — 验证图片加载：SVG 字符串光栅化 + 内存 image.Image 直接给(不依赖文件)。
// 运行: $env:CGO_ENABLED='1'; go run ./examples/imagedemo/
package main

import (
	"fmt"
	"image"
	"image/color"
	"os"

	"github.com/user/goui/internal/canvas"
	"github.com/user/goui/internal/render"
	"github.com/user/goui/internal/types"
	"github.com/user/goui/internal/widget"
)

const (
	cw = 420
	ch = 180
)

func main() {
	sk := canvas.NewSkiaCanvas(cw, ch)
	defer sk.Release()

	// 1) SVG 字符串 → 光栅化（数据直接给，非文件）
	svgContent := `<svg width="80" height="80" viewBox="0 0 80 80" xmlns="http://www.w3.org/2000/svg">` +
		`<rect x="6" y="6" width="68" height="68" rx="14" fill="#409EFF"/>` +
		`<polyline points="24,42 35,53 56,28" stroke="#ffffff" stroke-width="7" fill="none"/></svg>`
	svgImg, err := widget.NewImageFromSVG(svgContent, 0, 0)
	if err != nil {
		fmt.Println("svg err:", err)
		os.Exit(1)
	}

	// 2) 程序生成内存 image.RGBA（数据直接给）：对角渐变
	gen := image.NewRGBA(image.Rect(0, 0, 80, 80))
	for y := 0; y < 80; y++ {
		for x := 0; x < 80; x++ {
			gen.Set(x, y, color.RGBA{R: uint8(x * 3), G: uint8(y * 3), B: 200, A: 255})
		}
	}

	root := widget.Div(
		widget.Style{Width: cw, Height: ch, Padding: types.EdgeInsets(24), BackgroundColor: types.ColorRef(255, 255, 255), FlexDirection: "column", Gap: 14},
		widget.H3("Image：SVG 光栅化 + 内存数据直接给"),
		widget.Div(widget.Style{FlexDirection: "row", Gap: 28, AlignItems: "center"},
			svgImg.WithSize(80, 80).WithFit(widget.ImageFitContain),
			widget.NewImage(gen).WithSize(80, 80),
			widget.NewText("← SVG 字符串 · 内存 RGBA →", types.ColorFromRGB(96, 98, 102)),
		),
	)

	rootEl := widget.CreateElementFor(root)
	pipe := render.NewPipeline(cw, ch, sk)
	pipe.SetRootElement(rootEl)
	if err := pipe.Render(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if err := sk.SaveToPNG("image_demo.png"); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	fmt.Println("✅ image_demo.png 已保存")
}
