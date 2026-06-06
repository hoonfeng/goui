// showcase — 批次① 全组件可交互验收 Demo（真实窗口）。
//
// 运行（从 repo root，需 CGO + libSkiaSharp.dll + fonts/）:
//   $env:CGO_ENABLED='1'; go run ./examples/showcase/
//
// 体验点：
//   · hover「悬停我」按钮 → 弹出 Tooltip
//   · 点「点我」按钮 → 弹出 Popover，点别处关闭
//   · hover 各 Link → 变色 + 下划线；hover 可关闭 Tag 的 ×
//   · 点 Alert 右侧 × → 关闭
package main

import (
	"bytes"
	"image"
	"image/color"
	"image/gif"
	"runtime"
	"time"

	"github.com/user/goui/internal/app"
	"github.com/user/goui/internal/types"
	"github.com/user/goui/internal/widget"
)

func gap(w float64) widget.Widget { return widget.Div(widget.Style{Width: w}) }

// carouselSlide 一张轮播色块（撑满 Carousel 给的尺寸 + 居中文字）。
func carouselSlide(text string, col types.Color) widget.Widget {
	c := col
	return widget.Div(widget.Style{BackgroundColor: &c, AlignItems: "center", JustifyContent: "center"},
		widget.NewText(text, types.ColorFromRGB(255, 255, 255)),
	)
}

// makeDemoGIF 生成一段方块左右往返的动画 GIF（演示 GIF 逐帧播放）。
func makeDemoGIF() []byte {
	const n, sz = 12, 64
	pal := color.Palette{
		color.RGBA{245, 247, 250, 255}, // 背景
		color.RGBA{64, 158, 255, 255},  // 蓝
		color.RGBA{103, 194, 58, 255},  // 绿
	}
	g := &gif.GIF{}
	for i := 0; i < n; i++ {
		fr := image.NewPaletted(image.Rect(0, 0, sz, sz), pal)
		t := i
		if t >= n/2 {
			t = n - 1 - t // 往返
		}
		x0 := t * (sz - 18) / (n/2 - 1)
		ci := uint8(1 + i%2)
		for y := 24; y < 40; y++ {
			for x := x0; x < x0+18 && x < sz; x++ {
				fr.SetColorIndex(x, y, ci)
			}
		}
		g.Image = append(g.Image, fr)
		g.Delay = append(g.Delay, 9) // 90ms/帧
	}
	var buf bytes.Buffer
	_ = gif.EncodeAll(&buf, g)
	return buf.Bytes()
}

func section(label string) widget.Widget {
	return widget.Div(
		widget.Style{Padding: types.EdgeInsetsLTRB(0, 16, 0, 6)},
		widget.NewText(label, types.ColorFromRGB(120, 120, 120)),
	)
}

// rowC 横向居中排列一行控件。
func rowC(kids ...widget.Widget) widget.Widget {
	args := []interface{}{widget.Style{FlexDirection: "row", AlignItems: "center", Gap: 18}}
	for _, k := range kids {
		args = append(args, k)
	}
	return widget.Div(args...)
}

func buildRoot() widget.Widget {
	blue := types.ColorFromRGB(64, 158, 255)
	green := types.ColorFromRGB(103, 194, 58)

	loginForm := widget.NewForm(
		&widget.FormItem{Label: "用户名", Placeholder: "请输入用户名", Required: true},
		&widget.FormItem{Label: "邮箱", Placeholder: "请输入邮箱", Rules: []widget.FormRule{
			{Required: true, Pattern: `^[\w.\-]+@[\w.\-]+\.\w+$`, Message: "邮箱格式不正确"},
		}},
		&widget.FormItem{Label: "密码", Placeholder: "请输入密码", Rules: []widget.FormRule{
			{Required: true, MinLen: 6, Message: "密码至少 6 位"},
		}},
		&widget.FormItem{Label: "角色", Field: widget.NewSelect([]widget.SelectOption{
			{Label: "管理员", Value: "admin", Icon: "★"},
			{Label: "普通用户", Value: "user", Icon: "☆"},
		}).WithPlaceholder("FormItem.Field 嵌任意控件").WithWidth(260)},
	).WithLabelWidth(70)

	gifImg, _ := widget.NewImageFromBytes(makeDemoGIF())
	svgIcon, _ := widget.NewImageFromSVG(`<svg width="48" height="48" viewBox="0 0 48 48"><rect x="4" y="4" width="40" height="40" rx="8" fill="#409EFF"/><polyline points="14,24 21,31 34,16" stroke="#ffffff" stroke-width="4" fill="none"/></svg>`, 0, 0)

	content := widget.Div(
		widget.Style{Padding: types.EdgeInsets(24), BackgroundColor: types.ColorRef(255, 255, 255), FlexDirection: "column", Gap: 10},
		// Affix 吸顶条：向下滚动时固定在视口顶部
		widget.NewAffix(widget.Div(
			widget.Style{BackgroundColor: types.ColorRef(64, 158, 255), Padding: types.EdgeInsets(10), BorderRadius: 6},
			widget.NewText("📌 滚动地基实测：向下滚动此条吸顶 · 右下角现回顶按钮",
				types.ColorFromRGB(255, 255, 255)),
		)).WithOffset(8),
		widget.Div(widget.Style{Height: 12}),
		widget.H2("goui × Element Plus · 批次① 验收（可交互）"),
		widget.NewText("hover「悬停我」出提示 · 点「点我」出弹框 · hover 链接看反馈 · 点 Alert 的 × 关闭",
			types.ColorFromRGB(150, 150, 150)),

		section("Tag 标签"),
		widget.HBox(
			widget.NewTag("Primary"),
			gap(8), widget.NewTag("Success").WithType(widget.TagSuccess),
			gap(8), widget.NewTag("Plain").WithEffect(widget.TagPlain).WithType(widget.TagDanger),
			gap(8), widget.NewTag("可关闭").WithClosable(true).WithType(widget.TagWarning),
			gap(8), widget.NewTag("圆角").WithRound(true).WithEffect(widget.TagDark).WithType(widget.TagInfo),
		),

		section("Badge 徽章 / Avatar 头像"),
		widget.HBox(
			widget.NewBadge("5", widget.NewAvatar("张").WithBgColor(blue)),
			gap(24), widget.NewBadge("100", widget.NewAvatar("李").WithBgColor(green)).WithMax(99),
			gap(24), widget.NewDotBadge(widget.NewAvatar("王")),
			gap(24), widget.NewAvatar("方").WithShape(widget.AvatarSquare).WithBgColor(blue),
		),

		section("Link 链接（hover 看下划线 / 变色）"),
		widget.HBox(
			widget.NewLink("默认", func() {}),
			gap(16), widget.NewLink("主要", func() {}).WithType(widget.LinkPrimary),
			gap(16), widget.NewLink("成功", func() {}).WithType(widget.LinkSuccess),
			gap(16), widget.NewLink("危险", func() {}).WithType(widget.LinkDanger),
			gap(16), widget.NewLink("禁用", func() {}).WithDisabled(true),
		),

		section("Tooltip 提示 / Popover 弹出框"),
		widget.HBox(
			widget.NewTooltip("我是一条文字提示 🎈", widget.NewButton("悬停我", func() {})),
			gap(16),
			widget.NewPopover(widget.NewButton("点我", func() {})).
				WithTitle("弹出框标题").
				WithContent("这是弹出框里的内容，点击别处即可关闭。"),
		),

		section("Alert 警告（点 × 关闭）"),
		widget.NewAlert("成功提示", widget.AlertSuccess).
			WithDescription("可以点右侧 × 关闭我").WithClosable(true),
		widget.Div(widget.Style{Padding: types.EdgeInsetsLTRB(0, 6, 0, 0)},
			widget.NewAlert("危险提示（dark）", widget.AlertError).WithDark(true).WithClosable(true)),

		section("Empty 空状态"),
		widget.NewEmpty("暂无数据"),

		section("表单输入类（batch② · 拖开关 / 点星星 / 点加减 / 开下拉 实测）"),
		rowC(
			widget.NewSwitch(false, nil),
			widget.NewSwitch(true, nil).WithLabel("通知"),
			widget.NewSwitch(true, nil).WithTexts("开", "关"),
			widget.NewSwitch(true, nil).WithDisabled(true),
		),
		rowC(
			widget.NewRate(3, nil),
			widget.NewRate(2.5, nil).WithAllowHalf(true),
			widget.NewRate(4, nil).WithSize("large"),
		),
		rowC(
			widget.NewInputNumber(5, nil),
			widget.NewInputNumber(0, nil).WithRange(0, 10).WithStep(2),
			widget.NewInputNumber(3.14, nil).WithPrecision(2).WithStep(0.1).WithWidth(160),
		),
		rowC(
			widget.NewSelect([]widget.SelectOption{
				{Label: "北京", Value: "bj"},
				{Label: "上海", Value: "sh"},
				{Label: "广州（禁用）", Value: "gz", Disabled: true},
				{Label: "深圳", Value: "sz"},
			}).WithValue("sh"),
			widget.P("← 点击弹下拉 / 选项 hover / 点选切换"),
		),

		section("导航组织类（batch② · Tabs 点切换看滑块 / Breadcrumb hover 链接 / Steps 流程）"),
		widget.NewBreadcrumb(
			widget.BreadcrumbItem{Label: "首页", OnClick: func() {}},
			widget.BreadcrumbItem{Label: "组件", OnClick: func() {}},
			widget.BreadcrumbItem{Label: "导航", OnClick: func() {}},
			widget.BreadcrumbItem{Label: "面包屑"},
		),
		widget.Div(widget.Style{Height: 8}),
		widget.NewSteps(
			widget.Step{Title: "填写信息", Description: "第一步"},
			widget.Step{Title: "核对数据", Description: "第二步"},
			widget.Step{Title: "提交完成", Description: "第三步"},
		).WithActive(1),
		widget.Div(widget.Style{Height: 8}),
		widget.NewTabs(
			widget.TabPane{Label: "用户信息", Content: widget.Div(
				widget.Style{FlexDirection: "column", Gap: 6},
				widget.P("姓名：张三"),
				widget.P("邮箱：zhangsan@example.com"),
			)},
			widget.TabPane{Label: "配置管理", Content: widget.P("配置管理面板内容…")},
			widget.TabPane{Label: "角色权限", Content: widget.P("角色权限面板内容…")},
		),

		section("中频组件（batch③ · 拖滑块 / 展开折叠 / 切分页 / 弹下拉菜单）"),
		widget.NewProgress(0.6),
		widget.NewProgress(0.85).WithStatus("success"),
		widget.NewSlider(40, nil).WithLabel(true),
		widget.NewCollapse(
			widget.CollapseItem{Title: "折叠项 A：一致性", Content: widget.P("与现实生活的流程、逻辑保持一致。")},
			widget.CollapseItem{Title: "折叠项 B：反馈", Content: widget.P("通过界面样式和交互动效让用户清楚操作结果。")},
		).WithExpanded(0),
		widget.NewPagination(120, 10, 3),
		widget.NewDropdown(widget.NewButton("操作菜单 v", nil),
			widget.DropdownItem{Label: "编辑", Command: "edit"},
			widget.DropdownItem{Label: "复制", Command: "copy"},
			widget.DropdownItem{Label: "删除", Command: "delete", Divided: true},
			widget.DropdownItem{Label: "归档（禁用）", Command: "archive", Disabled: true},
		),

		section("L3 模态（点按钮弹出 · 点 ✕/遮罩/按钮关闭）"),
		widget.HBox(
			widget.NewButton("打开对话框", func() {
				var id int
				closeFn := func() { widget.HideOverlay(id) }
				id = widget.ShowDialog(
					widget.NewDialog("提示",
						widget.P("此操作将永久删除该文件，是否继续？"),
					).WithFooter(
						widget.NewButton("取消", closeFn).
							WithColor(types.ColorFromRGB(244, 244, 245)).
							WithTextColor(types.ColorFromRGB(96, 98, 102)),
						widget.NewButton("确定", closeFn).WithColor(blue),
					),
				)
			}),
			gap(12),
			widget.NewButton("打开抽屉", func() {
				var id int
				closeFn := func() { widget.HideOverlay(id) }
				id = widget.ShowDrawer(
					widget.NewDrawer("设置面板",
						widget.Div(widget.Style{FlexDirection: "column", Gap: 8},
							widget.P("在这里调整应用偏好设置。"),
							widget.Small("· 通知提醒"),
							widget.Small("· 隐私与安全"),
							widget.Small("· 外观主题"),
						),
					).WithSize(300).WithFooter(
						widget.NewButton("取消", closeFn).
							WithColor(types.ColorFromRGB(244, 244, 245)).
							WithTextColor(types.ColorFromRGB(96, 98, 102)),
						widget.NewButton("保存", closeFn).WithColor(blue),
					),
				)
			}),
			gap(12),
			widget.NewButton("显示加载", func() {
				l := widget.NewLoading("加载中... 点我关闭")
				var lid int
				l.OnClose = func() { widget.HideOverlay(lid) }
				lid = widget.ShowLoading(l)
			}),
		),

		section("反馈提示类（batch② · Message 顶部消息 / Notification 右上角通知，自动消失）"),
		rowC(
			widget.NewButton("成功", func() { widget.MessageSuccess("操作成功完成！") }).WithColor(types.ColorFromRGB(103, 194, 58)),
			widget.NewButton("警告", func() { widget.MessageWarning("请注意检查输入") }).WithColor(types.ColorFromRGB(230, 162, 60)),
			widget.NewButton("信息", func() { widget.MessageInfo("这是一条提示信息") }).WithColor(types.ColorFromRGB(144, 147, 153)),
			widget.NewButton("错误", func() { widget.MessageError("网络请求失败") }).WithColor(types.ColorFromRGB(245, 108, 108)),
		),
		rowC(
			widget.NewButton("通知·成功", func() { widget.NotifySuccess("成功", "数据已保存到云端") }).WithColor(types.ColorFromRGB(103, 194, 58)),
			widget.NewButton("通知·错误", func() { widget.NotifyError("错误", "连接服务器失败，请重试") }).WithColor(types.ColorFromRGB(245, 108, 108)),
			widget.NewButton("通知·信息", func() { widget.NotifyInfo("提示", "您有 3 条新消息待查看") }).WithColor(types.ColorFromRGB(144, 147, 153)),
		),
		rowC(
			widget.NewButton("确认框", func() {
				widget.ShowConfirm("提示", "此操作将永久删除该文件，是否继续？", widget.MsgWarning, nil, nil)
			}).WithColor(types.ColorFromRGB(230, 162, 60)),
			widget.NewButton("告知框", func() {
				widget.ShowAlert("成功", "操作已成功完成！", widget.MsgSuccess, nil)
			}).WithColor(types.ColorFromRGB(103, 194, 58)),
			widget.P("← MessageBox 居中模态确认/告知框"),
		),
		rowC(
			widget.NewButton("通知·左上", func() {
				widget.ShowNotificationWith("位置可配", "我在左上角 top-left", widget.MsgInfo, widget.ToastConfig{Position: "top-left"})
			}).WithColor(types.ColorFromRGB(144, 147, 153)),
			widget.NewButton("通知·左下", func() {
				widget.ShowNotificationWith("位置可配", "我在左下角 bottom-left", widget.MsgSuccess, widget.ToastConfig{Position: "bottom-left"})
			}).WithColor(green),
			widget.NewButton("通知·右下", func() {
				widget.ShowNotificationWith("位置可配", "我在右下角 bottom-right", widget.MsgWarning, widget.ToastConfig{Position: "bottom-right"})
			}).WithColor(types.ColorFromRGB(230, 162, 60)),
			widget.NewButton("消息·底部", func() {
				widget.ShowMessageWith("我在底部居中弹出", widget.MsgInfo, widget.ToastConfig{Position: "bottom"})
			}),
			widget.NewButton("消息·自定义紫", func() {
				purple := types.ColorFromRGB(130, 80, 220)
				white := types.ColorFromRGB(255, 255, 255)
				widget.ShowMessageWith("自定义紫底白字消息", widget.MsgInfo, widget.ToastConfig{
					BgColor: &purple, BorderColor: &purple, IconColor: &white, TextColor: &white,
				})
			}).WithColor(types.ColorFromRGB(130, 80, 220)),
		),
		widget.NewText("↑ 通知四角任放 · 消息可底部 · 可自定义配色（ToastConfig 参数）", types.ColorFromRGB(144, 147, 153)),

		section("重型组件类（batch④ · Table hover 行 / DatePicker 日历今天 / Cascader 级联展开）"),
		widget.NewTable([]widget.TableColumn{
			{Label: "日期", Prop: "date", Width: 120},
			{Label: "姓名", Prop: "name", Width: 90},
			{Label: "地址", Prop: "address"},
			{Label: "金额", Prop: "amount", Width: 100, Align: "right"},
		}, []map[string]string{
			{"date": "2024-01-01", "name": "张三", "address": "北京市朝阳区建国路 88 号", "amount": "¥1,200"},
			{"date": "2024-01-02", "name": "李四", "address": "上海市浦东新区世纪大道", "amount": "¥3,500"},
			{"date": "2024-01-03", "name": "王五", "address": "广州市天河区珠江新城", "amount": "¥890"},
		}).WithStripe(true),
		rowC(
			widget.NewText("选择日期：", types.ColorFromRGB(96, 98, 102)),
			widget.NewDatePicker().WithPlaceholder("点开 → 定位今天").WithOnChange(func(d string) {
				widget.MessageSuccess("已选择日期：" + d)
			}),
			widget.NewText("（弹出即停在当前月·今天蓝字·底部「今天」一键选当天）", types.ColorFromRGB(144, 147, 153)),
		),
		rowC(
			widget.NewText("级联选择：", types.ColorFromRGB(96, 98, 102)),
			widget.NewCascader([]widget.CascaderOption{
				{Label: "浙江", Value: "zj", Children: []widget.CascaderOption{
					{Label: "杭州", Value: "hz", Children: []widget.CascaderOption{
						{Label: "西湖区", Value: "xh"}, {Label: "余杭区", Value: "yh"},
					}},
					{Label: "宁波", Value: "nb", Children: []widget.CascaderOption{
						{Label: "海曙区", Value: "hs"},
					}},
				}},
				{Label: "江苏", Value: "js", Children: []widget.CascaderOption{
					{Label: "南京", Value: "nj", Children: []widget.CascaderOption{
						{Label: "玄武区", Value: "xw"},
					}},
				}},
			}).WithPlaceholder("请选择省/市/区").WithWidth(320).WithOnChange(func(vals []string) {
				widget.MessageSuccess("已选择级联路径！")
			}),
			widget.NewText("（点列展开下一级·选到区级回填整条路径）", types.ColorFromRGB(144, 147, 153)),
		),

		section("数据组件 slot（表格嵌按钮/Tag · 下拉带图标 · 不框定 el 风格）"),
		widget.NewTable([]widget.TableColumn{
			{Label: "姓名", Prop: "name", Width: 100},
			{Label: "状态", Prop: "status", Width: 90, Render: func(row map[string]string) widget.Widget {
				if row["status"] == "在线" {
					return widget.NewTag("在线").WithColor(green)
				}
				return widget.NewTag("离线").WithColor(types.ColorFromRGB(144, 147, 153))
			}},
			{Label: "操作", Width: 180, Render: func(row map[string]string) widget.Widget {
				name := row["name"]
				return widget.Div(widget.Style{FlexDirection: "row", Gap: 8, AlignItems: "center"},
					widget.NewButton("编辑", func() { widget.MessageInfo("编辑 " + name) }).WithColor(blue),
					widget.NewButton("删除", func() { widget.MessageWarning("删除 " + name) }).WithColor(types.ColorFromRGB(245, 108, 108)),
				)
			}},
		}, []map[string]string{
			{"name": "张三", "status": "在线"},
			{"name": "李四", "status": "离线"},
			{"name": "王五", "status": "在线"},
		}).WithStripe(true),
		rowC(
			widget.NewText("带图标下拉：", types.ColorFromRGB(96, 98, 102)),
			widget.NewSelect([]widget.SelectOption{
				{Label: "文档", Value: "doc", Icon: "📄"},
				{Label: "图片", Value: "img", Icon: "🖼"},
				{Label: "音乐", Value: "music", Icon: "🎵"},
			}).WithPlaceholder("选择类型").WithWidth(180),
			widget.NewDropdown(widget.NewButton("操作菜单 ▾", nil).WithColor(blue),
				widget.DropdownItem{Label: "编辑", Command: "edit", Icon: "✎"},
				widget.DropdownItem{Label: "复制", Command: "copy", Icon: "📋"},
				widget.DropdownItem{Label: "删除", Command: "delete", Icon: "✗", Divided: true},
			).WithOnCommand(func(cmd string) { widget.MessageInfo("命令：" + cmd) }),
		),
		section("图片资源（SVG 矢量 · GIF 动画 · 数据直接给，可塞进任意组件）"),
		widget.HBox(
			svgIcon.WithSize(48, 48),
			gap(24),
			gifImg.WithSize(64, 64),
			gap(20),
			widget.NewText("← SVG 矢量图标 · GIF 动画（方块往返播放）", types.ColorFromRGB(96, 98, 102)),
		),

		section("主线收官（ColorPicker 拖色 / Carousel 轮播 / Transfer 穿梭 / Popconfirm 气泡确认 / Calendar 月历）"),
		rowC(
			widget.NewStatistic("总销售额", "126,560").WithSuffix("元"),
			widget.NewStatistic("订单数", "1,280"),
			widget.NewPopconfirm(
				widget.NewButton("删除", nil).WithColor(types.ColorFromRGB(245, 108, 108)),
				"确定删除这条记录吗？",
			).WithOnConfirm(func() { widget.MessageSuccess("已删除") }).WithOnCancel(func() { widget.MessageInfo("已取消") }),
			widget.NewText("← Statistic 统计 · Popconfirm 点删除弹气泡", types.ColorFromRGB(144, 147, 153)),
		),
		widget.HBox(
			widget.NewColorPicker().WithOnChange(func(hex string) { widget.MessageInfo("取色：" + hex) }),
			gap(24),
			widget.NewCalendar().WithValue("2024-06-15"),
		),
		widget.NewCarousel(
			carouselSlide("第一张 · 蓝", types.ColorFromRGB(64, 158, 255)),
			carouselSlide("第二张 · 绿", types.ColorFromRGB(103, 194, 58)),
			carouselSlide("第三张 · 橙", types.ColorFromRGB(230, 162, 60)),
		).WithHeight(160).WithInterval(2500 * time.Millisecond),
		widget.NewTransfer(
			widget.TransferItem{Key: "1", Label: "选项一"},
			widget.TransferItem{Key: "2", Label: "选项二"},
			widget.TransferItem{Key: "3", Label: "选项三"},
			widget.TransferItem{Key: "4", Label: "选项四"},
		).WithTitles("待选", "已选"),

		section("表单校验类（batch④ 收官 · 输入实时校验 / 点提交全量校验 · 失败红边框+红字）"),
		loginForm,
		widget.NewButton("提交校验", func() {
			if loginForm.Validate() {
				widget.MessageSuccess("校验通过，提交成功！")
			} else {
				widget.MessageError("请修正表单中的错误")
			}
		}).WithColor(blue),

		widget.Div(widget.Style{Height: 240}), // 底部留白，给滚动充足空间
	)

	// 整个验收页包进 ScrollView：窗口可滚动看全部组件；
	// 顶部 Affix 吸顶、右下 Backtop 回顶——一并实测滚动地基。
	return widget.NewBacktop(widget.NewScrollView(content)).
		WithVisibilityHeight(100).WithOffset(30, 30)
}

func main() {
	runtime.LockOSThread()
	application := app.NewApplication()
	application.SetRootWidget(buildRoot())
	cfg := app.DefaultConfig()
	cfg.Title = "goui — 批次① 验收"
	application.Run(cfg)
}
