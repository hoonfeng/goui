// tourdemo — 验证 Tour 拼装重构：标题/内容/进度用 Text，按钮复用真 Button。
// 运行: $env:CGO_ENABLED='1'; go run ./examples/tourdemo/
package main

import (
	"fmt"
	"os"

	"github.com/user/goui/internal/canvas"
	"github.com/user/goui/internal/render"
	"github.com/user/goui/internal/widget"
)

const (
	cw = 600
	ch = 380
)

func main() {
	sk := canvas.NewSkiaCanvas(cw, ch)
	defer sk.Release()

	root := widget.TourPreview(
		widget.TourStep{Title: "欢迎使用 goui", Content: "这是漫游引导第一步。卡片的标题/正文/进度用 Text，\n上一步/下一步复用真正的 Button，不再手绘。"},
		widget.TourStep{Title: "第二步：组件", Content: "在这里浏览各个组件……"},
		widget.TourStep{Title: "完成", Content: "开始使用吧！"},
	)

	el := widget.CreateElementFor(root)
	pipe := render.NewPipeline(cw, ch, sk)
	pipe.SetRootElement(el)
	if err := pipe.Render(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if err := sk.SaveToPNG("tour_demo.png"); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	fmt.Println("✅ tour_demo.png 已保存")
}
