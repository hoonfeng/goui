// tabs — 验证 batch② 导航组织组件：Tabs 标签页(el-tabs 1:1)。
// 点标签切换内容 + 滑块平滑移动需窗口实测；此处 headless 验证标签栏 + 激活态 + 首个内容。
//
// 运行: $env:CGO_ENABLED='1'; go run ./examples/tabs/
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
	cw = 520
	ch = 300
)

func main() {
	sk := canvas.NewSkiaCanvas(cw, ch)
	defer sk.Release()

	tabs := widget.NewTabs(
		widget.TabPane{Label: "用户信息", Content: widget.Div(
			widget.Style{FlexDirection: "column", Gap: 8},
			widget.P("姓名：张三"),
			widget.P("邮箱：zhangsan@example.com"),
			widget.P("部门：研发中心"),
		)},
		widget.TabPane{Label: "配置管理", Content: widget.P("配置管理面板内容…")},
		widget.TabPane{Label: "角色权限", Content: widget.P("角色权限面板内容…")},
		widget.TabPane{Label: "定时任务", Content: widget.P("定时任务面板内容…")},
	)

	root := widget.Div(
		widget.Style{Width: cw, Height: ch, Padding: types.EdgeInsets(20), BackgroundColor: types.ColorRef(255, 255, 255), FlexDirection: "column", Gap: 12},
		widget.H3("Tabs 标签页（el-tabs 1:1）"),
		tabs,
	)

	rootEl := widget.CreateElementFor(root)
	pipe := render.NewPipeline(cw, ch, sk)
	pipe.SetRootElement(rootEl)
	if err := pipe.Render(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if err := sk.SaveToPNG("tabs_demo.png"); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	fmt.Println("✅ tabs_demo.png 已保存")
}
