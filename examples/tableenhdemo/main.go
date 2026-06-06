// tableenhdemo — 验证 Table 固定列：姓名左固定 + 操作右固定 + 中间列横向滚动。
// DefaultScrollX 预滚动，headless 即可看到固定列贴边 + 阴影 + 中间滚动。
// 运行: $env:CGO_ENABLED='1'; go run ./examples/tableenhdemo/
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
	cw = 580
	ch = 320
)

func main() {
	sk := canvas.NewSkiaCanvas(cw, ch)
	defer sk.Release()

	columns := []widget.TableColumn{
		{Label: "姓名", Prop: "name", Width: 90, Fixed: "left"},
		{Label: "年龄", Prop: "age", Width: 70, Align: "center", Sortable: true},
		{Label: "城市", Prop: "city", Width: 90},
		{Label: "邮箱", Prop: "email", Width: 190},
		{Label: "电话", Prop: "phone", Width: 140},
		{Label: "部门", Prop: "dept", Width: 110},
		{Label: "操作", Prop: "action", Width: 80, Align: "center", Fixed: "right"},
	}
	data := []map[string]string{
		{"name": "张三", "age": "28", "city": "北京", "email": "zhangsan@example.com", "phone": "138-0000-1111", "dept": "研发部", "action": "编辑"},
		{"name": "李四", "age": "34", "city": "上海", "email": "lisi@example.com", "phone": "139-0000-2222", "dept": "市场部", "action": "编辑"},
		{"name": "王五", "age": "22", "city": "广州", "email": "wangwu@example.com", "phone": "137-0000-3333", "dept": "设计部", "action": "编辑"},
		{"name": "赵六", "age": "41", "city": "深圳", "email": "zhaoliu@example.com", "phone": "136-0000-4444", "dept": "产品部", "action": "编辑"},
	}

	tbl := widget.NewTable(columns, data).WithStripe(true)
	tbl.DefaultScrollX = 70 // 预滚动(滚掉年龄列)，验证中间列滚动 + 左右固定列阴影

	root := widget.Div(
		widget.Style{Width: cw, Height: ch, Padding: types.EdgeInsets(20), BackgroundColor: types.ColorRef(255, 255, 255), FlexDirection: "column", Gap: 10},
		widget.H3("Table 固定列：姓名左固定 · 操作右固定 · 中间横向滚动"),
		tbl,
	)

	rootEl := widget.CreateElementFor(root)
	pipe := render.NewPipeline(cw, ch, sk)
	pipe.SetRootElement(rootEl)
	if err := pipe.Render(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if err := sk.SaveToPNG("tableenh_demo.png"); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	fmt.Println("✅ tableenh_demo.png 已保存")
}
