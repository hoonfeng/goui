// tag — Tag 标签组件画廊，headless 渲染到 tag_gallery.png 做观感验证。
//
// 运行方式（需要 CGO + libSkiaSharp.dll + fonts/ 在工作目录）:
//   $env:CGO_ENABLED='1'; go run ./examples/tag/
//
// 输出: tag_gallery.png
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
	cw = 720
	ch = 560
)

// gap 返回一个固定宽度的水平间隔。
func gap() widget.Widget { return widget.Div(widget.Style{Width: 10}) }

// row 一行：左侧说明文字 + 一排标签（标签间留间隔）。
func row(label string, tags ...widget.Widget) widget.Widget {
	kids := make([]widget.Widget, 0, len(tags)*2)
	for i, t := range tags {
		if i > 0 {
			kids = append(kids, gap())
		}
		kids = append(kids, t)
	}
	return widget.Div(
		widget.Style{Padding: types.EdgeInsetsLTRB(0, 10, 0, 2)},
		widget.NewText(label, types.ColorFromRGB(120, 120, 120)),
		widget.Div(
			widget.Style{Padding: types.EdgeInsetsLTRB(0, 6, 0, 0)},
			widget.HBox(kids...),
		),
	)
}

// fiveTypes 返回某个 effect 下的五个语义类型标签。
func fiveTypes(effect widget.TagEffect) []widget.Widget {
	mk := func(s string, tp widget.TagType) widget.Widget {
		return widget.NewTag(s).WithType(tp).WithEffect(effect)
	}
	return []widget.Widget{
		mk("Primary", widget.TagPrimary),
		mk("Success", widget.TagSuccess),
		mk("Info", widget.TagInfo),
		mk("Warning", widget.TagWarning),
		mk("Danger", widget.TagDanger),
	}
}

func main() {
	sk := canvas.NewSkiaCanvas(cw, ch)
	defer sk.Release()

	root := widget.Div(
		widget.Style{Padding: types.EdgeInsets(24), BackgroundColor: types.ColorRef(255, 255, 255)},
		widget.H2("Tag 标签 · Element Plus 1:1 复刻"),

		row("effect=light（默认）", fiveTypes(widget.TagLight)...),
		row("effect=dark", fiveTypes(widget.TagDark)...),
		row("effect=plain", fiveTypes(widget.TagPlain)...),

		row("size = large / default / small",
			widget.NewTag("Large").WithSize(widget.TagSizeLarge),
			widget.NewTag("Default").WithSize(widget.TagSizeDefault),
			widget.NewTag("Small").WithSize(widget.TagSizeSmall),
		),

		row("closable / round / hit",
			widget.NewTag("可关闭").WithClosable(true).WithType(widget.TagSuccess),
			widget.NewTag("圆角胶囊").WithRound(true).WithType(widget.TagWarning),
			widget.NewTag("描边 hit").WithHit(true).WithEffect(widget.TagDark).WithType(widget.TagDanger),
		),

		row("自定义颜色",
			widget.NewTag("樱花粉").WithColor(types.ColorFromRGB(255, 107, 157)).WithEffect(widget.TagDark),
			widget.NewTag("紫罗兰").WithColor(types.ColorFromRGB(155, 89, 182)).WithEffect(widget.TagDark),
			widget.NewTag("青碧").WithColor(types.ColorFromRGB(26, 188, 156)).WithEffect(widget.TagDark),
		),
	)

	rootEl := widget.CreateElementFor(root)
	pipe := render.NewPipeline(cw, ch, sk)
	pipe.SetRootElement(rootEl)
	if err := pipe.Render(); err != nil {
		fmt.Fprintf(os.Stderr, "渲染失败: %v\n", err)
		os.Exit(1)
	}
	if err := sk.SaveToPNG("tag_gallery.png"); err != nil {
		fmt.Fprintf(os.Stderr, "保存失败: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("✅ tag_gallery.png 已保存")
}
