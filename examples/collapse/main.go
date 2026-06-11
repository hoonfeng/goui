// collapse — 验证 batch③ 中频组件：Collapse 折叠面板(el-collapse 1:1)。
// 点标题展开/收起需窗口实测；此处 headless 验证展开/收起两态 + 箭头方向。
//
// 运行: $env:CGO_ENABLED='1'; go run ./examples/collapse/
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
	cw = 480
	ch = 360
)

func main() {
	sk := canvas.NewSkiaCanvas(cw, ch)
	defer sk.Release()

	collapse := widget.NewCollapse(
		widget.CollapseItem{Title: "一致性 Consistency", Content: widget.Div(
			widget.Style{FlexDirection: "column", Gap: 6},
			widget.P("与现实生活一致：与现实生活的流程、逻辑保持一致。"),
			widget.P("在界面中一致：所有元素和结构需保持一致。"),
		)},
		widget.CollapseItem{Title: "反馈 Feedback", Content: widget.P("通过界面样式和交互动效让用户清楚操作结果。")},
		widget.CollapseItem{Title: "效率 Efficiency", Content: widget.P("设计简洁直观的操作流程。")},
	).WithExpanded(0)

	root := widget.Div(
		widget.Style{Width: cw, Height: ch, Padding: types.EdgeInsets(24), BackgroundColor: types.ColorRef(255, 255, 255), FlexDirection: "column", Gap: 10},
		widget.H3("Collapse 折叠面板（el-collapse 1:1）"),
		widget.P("点标题展开/收起 · 首项默认展开(▼)、其余收起(▶)"),
		collapse,
	)

	rootEl := widget.CreateElementFor(root)
	pipe := render.NewPipeline(cw, ch, sk)
	pipe.SetRootElement(rootEl)
	if err := pipe.Render(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if err := sk.SaveToPNG("collapse_demo.png"); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	fmt.Println("✅ collapse_demo.png 已保存")
}
