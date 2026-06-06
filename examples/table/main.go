// table — 验证 Table 列自定义渲染(slot)：单元格嵌 Tag/按钮等真组件。
// 运行: $env:CGO_ENABLED='1'; go run ./examples/table/
package main

import (
	"fmt"
	"os"

	"github.com/user/goui/internal/canvas"
	"github.com/user/goui/internal/render"
	"github.com/user/goui/internal/types"
	"github.com/user/goui/internal/widget"
)

const (
	cw = 640
	ch = 280
)

func main() {
	sk := canvas.NewSkiaCanvas(cw, ch)
	defer sk.Release()

	columns := []widget.TableColumn{
		{Label: "姓名", Prop: "name", Width: 90},
		{Label: "地址", Prop: "address"},
		{Label: "状态", Prop: "status", Width: 80, Render: func(row map[string]string) widget.Widget {
			if row["status"] == "启用" {
				return widget.NewTag("启用").WithColor(types.ColorFromRGB(103, 194, 58))
			}
			return widget.NewTag("停用").WithColor(types.ColorFromRGB(245, 108, 108))
		}},
		{Label: "操作", Width: 170, Render: func(row map[string]string) widget.Widget {
			return widget.Div(widget.Style{FlexDirection: "row", Gap: 8, AlignItems: "center"},
				widget.NewButton("编辑", func() {}).WithColor(types.ColorFromRGB(64, 158, 255)),
				widget.NewButton("删除", func() {}).WithColor(types.ColorFromRGB(245, 108, 108)),
			)
		}},
	}
	data := []map[string]string{
		{"name": "张三", "address": "北京市朝阳区建国路", "status": "启用"},
		{"name": "李四", "address": "上海市浦东新区世纪大道", "status": "停用"},
		{"name": "王五", "address": "广州市天河区珠江新城", "status": "启用"},
	}

	root := widget.Div(
		widget.Style{Width: cw, Height: ch, Padding: types.EdgeInsets(24), BackgroundColor: types.ColorRef(255, 255, 255), FlexDirection: "column", Gap: 12},
		widget.H3("Table 列自定义渲染（单元格嵌 Tag + 按钮）"),
		widget.NewTable(columns, data).WithStripe(true),
	)

	rootEl := widget.CreateElementFor(root)
	pipe := render.NewPipeline(cw, ch, sk)
	pipe.SetRootElement(rootEl)
	if err := pipe.Render(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if err := sk.SaveToPNG("table_slot_demo.png"); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	fmt.Println("✅ table_slot_demo.png 已保存")
}
