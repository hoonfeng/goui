// hradmin — 员工管理后台（goui 进阶组件业务验收）
//
// 一个贴近真实业务的 HR 后台页面，把进阶细分四大块用在实战里：
//   • Table 四大增强：多选批量 + 列排序 + 固定列(姓名左/操作右) + 可展开行(员工详情)
//                      + slot(状态彩色 Tag、操作 Link)
//   • Select 三件套：部门多选(变 tag) + 状态可清除 + 下拉
//   • 异步 Autocomplete：顶部搜索员工(远程 fetch + loading + 防抖)
//   • Form 联动校验：新增员工(工号格式/邮箱/密码 MinLen + 确认密码联动 + 必填)
//
// 运行(开窗口交互): $env:CGO_ENABLED='1'; go run ./examples/hradmin/
// 渲染静态预览PNG:  $env:CGO_ENABLED='1'; $env:HRADMIN_PNG='1'; go run ./examples/hradmin/
package main

import (
	"fmt"
	"log"
	"os"
	"runtime"
	"strings"

	"github.com/user/goui/internal/app"
	"github.com/user/goui/internal/canvas"
	"github.com/user/goui/internal/render"
	"github.com/user/goui/internal/types"
	"github.com/user/goui/internal/widget"
)

const (
	winW = 1120
	winH = 720
)

var (
	colDark  = types.ColorFromRGB(48, 49, 51)
	colText  = types.ColorFromRGB(96, 98, 102)
	colMuted = types.ColorFromRGB(144, 147, 153)
)

// 员工数据（真实字段：工号/姓名/部门/职位/邮箱/电话/入职/状态 + 详情）
var employees = []map[string]string{
	{"empno": "E1001", "name": "张伟", "dept": "研发部", "title": "架构师", "email": "zhangwei@corp.com", "phone": "138-0011-2233", "joinDate": "2019-03-15", "status": "在职", "addr": "北京市朝阳区望京 SOHO", "emergency": "李娜 139-8800-1122"},
	{"empno": "E1002", "name": "李娜", "dept": "市场部", "title": "市场总监", "email": "lina@corp.com", "phone": "139-0022-3344", "joinDate": "2020-07-01", "status": "在职", "addr": "上海市浦东新区世纪大道", "emergency": "王强 137-6600-2233"},
	{"empno": "E1003", "name": "王芳", "dept": "设计部", "title": "UI 设计师", "email": "wangfang@corp.com", "phone": "137-0033-4455", "joinDate": "2021-11-20", "status": "试用", "addr": "广州市天河区珠江新城", "emergency": "赵敏 136-5500-3344"},
	{"empno": "E1004", "name": "刘洋", "dept": "产品部", "title": "产品经理", "email": "liuyang@corp.com", "phone": "136-0044-5566", "joinDate": "2018-05-09", "status": "在职", "addr": "深圳市南山区科技园", "emergency": "孙丽 135-4400-4455"},
	{"empno": "E1005", "name": "陈静", "dept": "研发部", "title": "前端工程师", "email": "chenjing@corp.com", "phone": "135-0055-6677", "joinDate": "2022-02-14", "status": "试用", "addr": "杭州市西湖区文三路", "emergency": "周杰 134-3300-5566"},
	{"empno": "E1006", "name": "赵敏", "dept": "运营部", "title": "运营专员", "email": "zhaomin@corp.com", "phone": "134-0066-7788", "joinDate": "2017-09-30", "status": "离职", "addr": "成都市武侯区天府大道", "emergency": "吴昊 133-2200-6677"},
	{"empno": "E1007", "name": "孙丽", "dept": "设计部", "title": "视觉设计师", "email": "sunli@corp.com", "phone": "133-0077-8899", "joinDate": "2020-12-05", "status": "在职", "addr": "南京市鼓楼区中山路", "emergency": "郑爽 132-1100-7788"},
	{"empno": "E1008", "name": "周杰", "dept": "研发部", "title": "后端工程师", "email": "zhoujie@corp.com", "phone": "132-0088-9900", "joinDate": "2021-06-18", "status": "在职", "addr": "武汉市洪山区光谷大道", "emergency": "张伟 138-0011-2233"},
}

var hrForm *widget.Form // 包级引用，供 PNG 预览触发校验

func cardStyle() widget.Style {
	return widget.Style{
		Padding:         types.EdgeInsets(16),
		BackgroundColor: types.ColorRef(255, 255, 255),
		BorderColor:     types.ColorRef(228, 231, 237),
		BorderWidth:     1,
		BorderRadius:    8,
		FlexDirection:   "column",
		Gap:             12,
	}
}

func buildHRAdmin() widget.Widget {
	return widget.Div(
		widget.Style{Width: winW, Height: winH, Padding: types.EdgeInsets(24),
			BackgroundColor: types.ColorRef(245, 247, 250), FlexDirection: "column", Gap: 16},
		titleBar(),
		widget.Div(
			widget.Style{FlexDirection: "row", Gap: 20},
			widget.Div(
				widget.Style{Width: 740, FlexDirection: "column", Gap: 16},
				filterCard(),
				tableCard(),
			),
			rightFormCard(),
		),
	)
}

func titleBar() widget.Widget {
	return widget.Div(
		widget.Style{FlexDirection: "row", Gap: 12},
		widget.Div(
			widget.Style{FlexDirection: "column", Gap: 2},
			widget.H2("员工管理系统"),
			widget.NewText("goui 进阶组件业务验收 · Table / Select / Autocomplete / Form", colMuted),
		),
		widget.SpacerDiv(),
		widget.NewButton("+ 新增员工", func() { widget.MessageInfo("请在右侧表单填写后提交") }),
	)
}

func filterCard() widget.Widget {
	// 异步搜索：从员工库远程 fetch（此处同步模拟）
	search := widget.NewAutocomplete().WithFetch(func(q string, done func([]string)) {
		var out []string
		for _, e := range employees {
			if strings.Contains(e["name"], q) || strings.Contains(e["empno"], strings.ToUpper(q)) {
				out = append(out, e["name"]+"（"+e["empno"]+"）")
			}
		}
		done(out)
	}).WithDebounce(250)
	search.Width = 230
	search.Placeholder = "搜索姓名 / 工号…"

	deptSel := widget.NewSelect([]widget.SelectOption{
		{Label: "研发部", Value: "研发部"}, {Label: "市场部", Value: "市场部"},
		{Label: "设计部", Value: "设计部"}, {Label: "产品部", Value: "产品部"}, {Label: "运营部", Value: "运营部"},
	}).WithMultiple(true).WithFilterable(true).WithValues([]string{"研发部", "设计部"}).
		WithClearable(true).WithPlaceholder("部门筛选").WithWidth(230)

	statusSel := widget.NewSelect([]widget.SelectOption{
		{Label: "在职", Value: "在职"}, {Label: "试用", Value: "试用"}, {Label: "离职", Value: "离职"},
	}).WithValue("在职").WithClearable(true).WithPlaceholder("状态").WithWidth(120)

	return widget.Div(
		cardStyle(),
		widget.Div(
			widget.Style{FlexDirection: "row", Gap: 12},
			search, deptSel, statusSel,
			widget.SpacerDiv(),
			widget.NewText(fmt.Sprintf("共 %d 名员工", len(employees)), colText),
		),
	)
}

func tableCard() widget.Widget {
	columns := []widget.TableColumn{
		{Label: "姓名", Prop: "name", Width: 80, Fixed: "left"},
		{Label: "工号", Prop: "empno", Width: 80, Sortable: true},
		{Label: "部门", Prop: "dept", Width: 90, Sortable: true},
		{Label: "状态", Prop: "status", Width: 84, Align: "center", Render: statusTag},
		{Label: "职位", Prop: "title", Width: 110},
		{Label: "邮箱", Prop: "email", Width: 180},
		{Label: "电话", Prop: "phone", Width: 130},
		{Label: "入职日期", Prop: "joinDate", Width: 110, Sortable: true},
		{Label: "操作", Prop: "ops", Width: 120, Align: "center", Fixed: "right", Render: rowOps},
	}

	tbl := widget.NewTable(columns, employees).
		WithStripe(true).
		WithSelectable(true).
		WithExpand(expandDetail).
		WithDefaultExpanded(1).
		WithHeight(392) // 表格自带固定高度：表头固定 + body 内部竖滚
	tbl.DefaultScrollX = 0 // 初始从第一列自然开始；列多时拖底部横滚条看后续列
	tbl.OnSelectChange = func(rows []int) {
		widget.MessageInfo(fmt.Sprintf("已选中 %d 名员工", len(rows)))
	}
	tbl.OnSortChange = func(prop string, asc bool) {
		dir := "升序"
		if !asc {
			dir = "降序"
		}
		widget.MessageInfo("按「" + prop + "」" + dir)
	}

	return widget.Div(
		cardStyle(),
		widget.Div(widget.Style{FlexDirection: "row", Gap: 8},
			widget.H3("员工列表"),
			widget.SpacerDiv(),
			widget.NewText("勾选批量 · 点列头排序 · ▶ 展开 · 表内可上下/左右滚动", colMuted),
		),
		tbl, // 表格自带固定高度 + 内部竖滚，无需外层滚动容器
	)
}

func statusTag(row map[string]string) widget.Widget {
	tp := widget.TagInfo
	switch row["status"] {
	case "在职":
		tp = widget.TagSuccess
	case "试用":
		tp = widget.TagWarning
	case "离职":
		tp = widget.TagInfo
	}
	return widget.NewTag(row["status"]).WithType(tp)
}

func rowOps(row map[string]string) widget.Widget {
	name := row["name"]
	return widget.Div(
		widget.Style{FlexDirection: "row", Gap: 10},
		widget.NewLink("编辑", func() { widget.MessageInfo("编辑 " + name) }),
		widget.NewLink("删除", func() { widget.MessageWarning("删除 " + name) }),
	)
}

func expandDetail(row map[string]string) widget.Widget {
	return widget.Div(
		widget.Style{FlexDirection: "column", Gap: 4},
		widget.NewText("📋 "+row["name"]+" · "+row["title"]+"（"+row["dept"]+"）", colDark),
		widget.NewText("入职日期："+row["joinDate"]+"      联系电话："+row["phone"], colText),
		widget.NewText("住址："+row["addr"], colText),
		widget.NewText("紧急联系人："+row["emergency"], colText),
	)
}

func rightFormCard() widget.Widget {
	hrForm = widget.NewForm(
		&widget.FormItem{Label: "工号", Prop: "empno", Placeholder: "如 E1009", Required: true,
			Rules: []widget.FormRule{{Pattern: `^E\d{4}$`, Message: "工号格式应为 E+4位数字"}}},
		&widget.FormItem{Label: "姓名", Prop: "name", Placeholder: "请输入姓名", Required: true},
		&widget.FormItem{Label: "邮箱", Prop: "email", Placeholder: "name@corp.com",
			Rules: []widget.FormRule{{Required: true, Pattern: `^[^@]+@[^@]+\.[^@]+$`, Message: "邮箱格式不正确"}}},
		&widget.FormItem{Label: "密码", Prop: "pwd", Placeholder: "至少 6 位", Required: true,
			Rules: []widget.FormRule{{MinLen: 6}}, Triggers: []string{"pwd2"}},
		&widget.FormItem{Label: "确认密码", Prop: "pwd2", Placeholder: "再次输入密码", Required: true,
			Rules: []widget.FormRule{{Validator: func(v string) string {
				if v != hrForm.FieldValue("pwd") {
					return "两次密码不一致"
				}
				return ""
			}}}},
	).WithLabelWidth(72)

	return widget.Div(
		widget.Style{Width: 336, FlexDirection: "column"},
		widget.Div(
			cardStyle(),
			widget.H3("新增员工"),
			widget.NewText("提交时校验，密码改动联动确认框", colMuted),
			hrForm,
			widget.Div(
				widget.Style{FlexDirection: "row", Gap: 10},
				widget.SpacerDiv(),
				widget.NewButton("重置", func() { widget.MessageInfo("已重置") }),
				widget.NewButton("提交新增", func() {
					if hrForm.Validate() {
						widget.MessageSuccess("员工添加成功！")
					} else {
						widget.MessageError("请修正表单中的错误")
					}
				}),
			),
		),
	)
}

func main() {
	root := buildHRAdmin()

	if os.Getenv("HRADMIN_PNG") != "" {
		renderPNG(root)
		return
	}

	runtime.LockOSThread()
	application := app.NewApplication()
	application.SetRootWidget(root)
	config := app.DefaultConfig()
	config.Title = "员工管理系统 — goui 进阶组件验收"
	config.Width = winW
	config.Height = winH
	config.Resizable = true
	log.Println("hradmin: 启动员工管理后台")
	if err := application.Run(config); err != nil {
		log.Fatalf("hradmin: %v", err)
	}
}

// renderPNG 无窗口渲染静态预览（供验证布局；触发表单校验以展示错误态）。
func renderPNG(root widget.Widget) {
	sk := canvas.NewSkiaCanvas(winW, winH)
	defer sk.Release()
	el := widget.CreateElementFor(root)
	pipe := render.NewPipeline(winW, winH, sk)
	pipe.SetRootElement(el)
	pipe.Render()
	if hrForm != nil {
		hrForm.Validate()
		pipe.MarkNeedsLayout()
		pipe.Render()
	}
	if err := sk.SaveToPNG("hradmin.png"); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	fmt.Println("✅ hradmin.png 已保存")
}
