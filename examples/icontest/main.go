// icontest — Lucide 矢量图标 headless 渲染验证。
//
//	$env:CGO_ENABLED='1'; go run ./examples/icontest/
//
//go:build !ignore
// +build !ignore

package main

import (
	"fmt"

	"github.com/user/goui/internal/canvas"
	"github.com/user/goui/internal/render"
	"github.com/user/goui/internal/types"
	"github.com/user/goui/internal/widget"
)

const (
	W = 560
	H = 110
)

func main() {
	sk := canvas.NewSkiaCanvas(W, H)
	defer sk.Release()

	names := []string{
		"x", "menu", "search", "settings", "folder", "file",
		"chevron-down", "chevron-right", "message-square", "terminal",
		"git-branch", "panel-left", "plus", "check",
	}
	dark := types.ColorFromRGB(40, 44, 52)
	var icons []widget.Widget
	for _, n := range names {
		icons = append(icons, widget.Div(
			widget.Style{Padding: types.EdgeInsets(8)},
			widget.Lucide(n, widget.IconSize(28), widget.IconColor(dark)),
		))
	}
	root := widget.Div(
		widget.Style{Padding: types.EdgeInsets(12), BackgroundColor: types.ColorRef(255, 255, 255)},
		widget.HBox(icons...),
	)

	rootEl := widget.CreateElementFor(root)
	pipe := render.NewPipeline(W, H, sk)
	pipe.SetRootElement(rootEl)
	if err := pipe.Render(); err != nil {
		fmt.Println("render:", err)
		return
	}
	if err := sk.SaveToPNG("icontest_output.png"); err != nil {
		fmt.Println("save:", err)
		return
	}
	fmt.Println("✅ icontest_output.png 已保存（", len(names), "个图标）")
}
