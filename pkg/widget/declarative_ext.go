package widget

import (
	"fmt"

	"github.com/hoonfeng/goui/internal/layout"
)

// 声明式扩展：把更多组件接入 BuildFromSpec/LoadConfig，使声明式(JSON 配置)能用全部组件。
// 内置基础组件在 declarative.go 注册；本文件按批补齐其余组件，最终达到声明式全覆盖。

// ── props 取值 helper ──

// propStr 取对象的字符串字段（无则 ""）。
func propStr(m map[string]any, key string) string {
	if v, ok := m[key]; ok {
		return fmt.Sprintf("%v", v)
	}
	return ""
}

// propsStr 取 spec.Props 的字符串字段。
func propsStr(props map[string]any, key string) string { return propStr(props, key) }

// orStr a 非空用 a，否则 b（用于 props 与 spec.Text 二选一）。
func orStr(a, b string) string {
	if a != "" {
		return a
	}
	return b
}

// propsArray 把 props[key] 解析为对象数组（每元素 map[string]any）；支持 item 列表型组件。
func propsArray(props map[string]any, key string) []map[string]any {
	raw, ok := props[key]
	if !ok {
		return nil
	}
	arr, ok := raw.([]any)
	if !ok {
		return nil
	}
	out := make([]map[string]any, 0, len(arr))
	for _, e := range arr {
		if mm, ok := e.(map[string]any); ok {
			out = append(out, mm)
		}
	}
	return out
}

// propsStrArray 把 props[key] 解析为字符串数组（如 tags/suggestions）。
func propsStrArray(props map[string]any, key string) []string {
	raw, ok := props[key]
	if !ok {
		return nil
	}
	arr, ok := raw.([]any)
	if !ok {
		return nil
	}
	out := make([]string, 0, len(arr))
	for _, e := range arr {
		out = append(out, fmt.Sprintf("%v", e))
	}
	return out
}

func init() { registerExtComponents() }

func registerExtComponents() {
	// ── 展示/布局（packed_widgets）──

	RegisterComponent("Space", func(ctx DeclarativeContext) Widget {
		s := &Space{Children: ctx.Children, Size: parseFloat(ctx.Spec.Props["size"], 8)}
		s.Vertical = parseBool(ctx.Spec.Props["vertical"], false)
		return s
	})

	RegisterComponent("Statistic", func(ctx DeclarativeContext) Widget {
		return &Statistic{
			Title:  orStr(propsStr(ctx.Spec.Props, "title"), ctx.Spec.Text),
			Value:  propsStr(ctx.Spec.Props, "value"),
			Suffix: propsStr(ctx.Spec.Props, "suffix"),
		}
	})

	RegisterComponent("Result", func(ctx DeclarativeContext) Widget {
		r := &Result{
			Status:   propsStr(ctx.Spec.Props, "status"),
			Title:    orStr(propsStr(ctx.Spec.Props, "title"), ctx.Spec.Text),
			SubTitle: propsStr(ctx.Spec.Props, "subTitle"),
		}
		if len(ctx.Children) > 0 {
			r.Extra = ctx.Children[0] // 子组件作操作区
		}
		return r
	})

	RegisterComponent("Descriptions", func(ctx DeclarativeContext) Widget {
		d := &Descriptions{Title: propsStr(ctx.Spec.Props, "title")}
		for _, m := range propsArray(ctx.Spec.Props, "items") {
			d.Items = append(d.Items, DescItem{Label: propStr(m, "label"), Value: propStr(m, "value")})
		}
		return d
	})

	RegisterComponent("Timeline", func(ctx DeclarativeContext) Widget {
		t := &Timeline{}
		for _, m := range propsArray(ctx.Spec.Props, "items") {
			it := TimelineItem{Time: propStr(m, "time"), Content: propStr(m, "content")}
			if c := parseColor(propStr(m, "color")); c != nil {
				it.Color = *c
			}
			t.Items = append(t.Items, it)
		}
		return t
	})

	RegisterComponent("Skeleton", func(ctx DeclarativeContext) Widget {
		return &Skeleton{Rows: int(parseFloat(ctx.Spec.Props["rows"], 3))}
	})

	// ── 容器/布局（child / 多 child 映射）──

	RegisterComponent("Loading", func(ctx DeclarativeContext) Widget {
		l := NewLoading(orStr(propsStr(ctx.Spec.Props, "text"), ctx.Spec.Text))
		if c := parseColor(propsStr(ctx.Spec.Props, "color")); c != nil {
			l.Color = *c
		}
		return l
	})

	RegisterComponent("ScrollView", func(ctx DeclarativeContext) Widget {
		var child Widget
		if len(ctx.Children) > 0 {
			child = ctx.Children[0]
		}
		sv := NewScrollView(child)
		if parseBool(ctx.Spec.Props["horizontal"], false) {
			sv.ScrollDirection = layout.FlexRow
		}
		sv.ShowBar = parseBool(ctx.Spec.Props["showBar"], sv.ShowBar)
		return sv
	})

	RegisterComponent("Splitter", func(ctx DeclarativeContext) Widget {
		var l, r Widget
		if len(ctx.Children) > 0 {
			l = ctx.Children[0]
		}
		if len(ctx.Children) > 1 {
			r = ctx.Children[1]
		}
		s := NewSplitter(l, r)
		s.Height = parseFloat(ctx.Spec.Props["height"], s.Height)
		return s
	})

	RegisterComponent("Watermark", func(ctx DeclarativeContext) Widget {
		var child Widget
		if len(ctx.Children) > 0 {
			child = ctx.Children[0]
		}
		w := NewWatermark(orStr(propsStr(ctx.Spec.Props, "content"), ctx.Spec.Text), child)
		w.FontSize = parseFloat(ctx.Spec.Props["fontSize"], w.FontSize)
		w.Rotate = parseFloat(ctx.Spec.Props["rotate"], w.Rotate)
		return w
	})

	RegisterComponent("PageHeader", func(ctx DeclarativeContext) Widget {
		p := NewPageHeader(orStr(propsStr(ctx.Spec.Props, "title"), ctx.Spec.Text))
		p.SubTitle = propsStr(ctx.Spec.Props, "subTitle")
		if len(ctx.Children) > 0 {
			p.Extra = ctx.Children[0]
		}
		if h, ok := ctx.Spec.Events["back"]; ok {
			if fn, ok := ctx.Handlers[h]; ok {
				p.OnBack = func() { fn(EventContext{Name: "back", Target: p}) }
			}
		}
		return p
	})

	RegisterComponent("Stack", func(ctx DeclarativeContext) Widget {
		return NewStack(ctx.Children...)
	})

	RegisterComponent("Positioned", func(ctx DeclarativeContext) Widget {
		var child Widget
		if len(ctx.Children) > 0 {
			child = ctx.Children[0]
		}
		p := NewPositioned(child)
		if v, ok := ctx.Spec.Props["left"]; ok {
			p.Left, p.HasLeft = parseFloat(v, 0), true
		}
		if v, ok := ctx.Spec.Props["top"]; ok {
			p.Top, p.HasTop = parseFloat(v, 0), true
		}
		if v, ok := ctx.Spec.Props["right"]; ok {
			p.Right, p.HasRight = parseFloat(v, 0), true
		}
		if v, ok := ctx.Spec.Props["bottom"]; ok {
			p.Bottom, p.HasBottom = parseFloat(v, 0), true
		}
		p.ZIndex = int(parseFloat(ctx.Spec.Props["zIndex"], 0))
		return p
	})

	RegisterComponent("Carousel", func(ctx DeclarativeContext) Widget {
		c := NewCarousel(ctx.Children...)
		c.Height = parseFloat(ctx.Spec.Props["height"], c.Height)
		return c
	})

	RegisterComponent("Dialog", func(ctx DeclarativeContext) Widget {
		var body Widget
		if len(ctx.Children) > 0 {
			body = ctx.Children[0]
		}
		d := NewDialog(orStr(propsStr(ctx.Spec.Props, "title"), ctx.Spec.Text), body)
		d.Width = parseFloat(ctx.Spec.Props["width"], d.Width)
		d.Center = parseBool(ctx.Spec.Props["center"], d.Center)
		if len(ctx.Children) > 1 {
			d.Footer = ctx.Children[1:] // 额外子组件作底部按钮组
		}
		return d
	})

	RegisterComponent("Drawer", func(ctx DeclarativeContext) Widget {
		var body Widget
		if len(ctx.Children) > 0 {
			body = ctx.Children[0]
		}
		d := NewDrawer(orStr(propsStr(ctx.Spec.Props, "title"), ctx.Spec.Text), body)
		d.Direction = orStr(propsStr(ctx.Spec.Props, "direction"), d.Direction)
		d.Size = parseFloat(ctx.Spec.Props["size"], d.Size)
		if len(ctx.Children) > 1 {
			d.Footer = ctx.Children[1:]
		}
		return d
	})

	// ── item-list 组件（labels+children 配对 / props 数组解析）──

	RegisterComponent("Tabs", func(ctx DeclarativeContext) Widget {
		labels := propsStrArray(ctx.Spec.Props, "labels")
		var panes []TabPane
		for i := range ctx.Children {
			label := ""
			if i < len(labels) {
				label = labels[i]
			}
			panes = append(panes, TabPane{Label: label, Content: ctx.Children[i]})
		}
		t := NewTabs(panes...)
		t.ActiveIndex = int(parseFloat(ctx.Spec.Props["active"], 0))
		if h, ok := ctx.Spec.Events["change"]; ok {
			if fn, ok := ctx.Handlers[h]; ok {
				t.OnChange = func(i int) { fn(EventContext{Name: "change", Target: t, Data: i}) }
			}
		}
		return t
	})

	RegisterComponent("Collapse", func(ctx DeclarativeContext) Widget {
		titles := propsStrArray(ctx.Spec.Props, "titles")
		var items []CollapseItem
		for i := range ctx.Children {
			title := ""
			if i < len(titles) {
				title = titles[i]
			}
			items = append(items, CollapseItem{Title: title, Content: ctx.Children[i]})
		}
		c := NewCollapse(items...)
		c.Accordion = parseBool(ctx.Spec.Props["accordion"], false)
		return c
	})

	RegisterComponent("Steps", func(ctx DeclarativeContext) Widget {
		var steps []Step
		for _, m := range propsArray(ctx.Spec.Props, "items") {
			steps = append(steps, Step{Title: propStr(m, "title"), Description: propStr(m, "description")})
		}
		s := NewSteps(steps...)
		s.Active = int(parseFloat(ctx.Spec.Props["active"], 0))
		return s
	})

	RegisterComponent("Breadcrumb", func(ctx DeclarativeContext) Widget {
		var items []BreadcrumbItem
		for _, m := range propsArray(ctx.Spec.Props, "items") {
			items = append(items, BreadcrumbItem{Label: propStr(m, "label")})
		}
		return NewBreadcrumb(items...)
	})

	RegisterComponent("Dropdown", func(ctx DeclarativeContext) Widget {
		var trigger Widget
		if len(ctx.Children) > 0 {
			trigger = ctx.Children[0]
		}
		var items []DropdownItem
		for _, m := range propsArray(ctx.Spec.Props, "items") {
			items = append(items, DropdownItem{
				Label: propStr(m, "label"), Command: propStr(m, "command"),
				Disabled: parseBool(m["disabled"], false), Divided: parseBool(m["divided"], false), Icon: propStr(m, "icon"),
			})
		}
		return NewDropdown(trigger, items...)
	})

	RegisterComponent("Anchor", func(ctx DeclarativeContext) Widget {
		var links []AnchorLink
		for _, m := range propsArray(ctx.Spec.Props, "links") {
			links = append(links, AnchorLink{Title: propStr(m, "title"), Href: propStr(m, "href")})
		}
		return NewAnchor(links...)
	})

	RegisterComponent("Menu", func(ctx DeclarativeContext) Widget {
		return NewMenu(parseMenuItems(propsArray(ctx.Spec.Props, "items"))...)
	})

	RegisterComponent("Table", func(ctx DeclarativeContext) Widget {
		var cols []TableColumn
		for _, m := range propsArray(ctx.Spec.Props, "columns") {
			cols = append(cols, TableColumn{
				Label: propStr(m, "label"), Prop: propStr(m, "prop"), Width: parseFloat(m["width"], 0),
				Align: propStr(m, "align"), Sortable: parseBool(m["sortable"], false), Fixed: propStr(m, "fixed"),
			})
		}
		var data []map[string]string
		for _, m := range propsArray(ctx.Spec.Props, "data") {
			row := make(map[string]string, len(m))
			for k, v := range m {
				row[k] = fmt.Sprintf("%v", v)
			}
			data = append(data, row)
		}
		return NewTable(cols, data)
	})

	RegisterComponent("Transfer", func(ctx DeclarativeContext) Widget {
		var items []TransferItem
		for _, m := range propsArray(ctx.Spec.Props, "items") {
			items = append(items, TransferItem{Key: propStr(m, "key"), Label: propStr(m, "label")})
		}
		return NewTransfer(items...)
	})

	// ── 表单/输入类 ──

	RegisterComponent("DatePicker", func(ctx DeclarativeContext) Widget {
		d := NewDatePicker()
		d.Value = propsStr(ctx.Spec.Props, "value")
		d.Placeholder = orStr(propsStr(ctx.Spec.Props, "placeholder"), ctx.Spec.Text)
		d.Width = parseFloat(ctx.Spec.Props["width"], d.Width)
		d.Disabled = parseBool(ctx.Spec.Props["disabled"], false)
		bindStrChange(ctx, &d.OnChange)
		return d
	})

	RegisterComponent("TimePicker", func(ctx DeclarativeContext) Widget {
		t := NewTimePicker()
		t.Value = propsStr(ctx.Spec.Props, "value")
		if p := propsStr(ctx.Spec.Props, "placeholder"); p != "" {
			t.Placeholder = p
		}
		t.Width = parseFloat(ctx.Spec.Props["width"], t.Width)
		bindStrChange(ctx, &t.OnChange)
		return t
	})

	RegisterComponent("TimeSelect", func(ctx DeclarativeContext) Widget {
		t := NewTimeSelect()
		t.Start = orStr(propsStr(ctx.Spec.Props, "start"), t.Start)
		t.End = orStr(propsStr(ctx.Spec.Props, "end"), t.End)
		t.Step = orStr(propsStr(ctx.Spec.Props, "step"), t.Step)
		t.Value = propsStr(ctx.Spec.Props, "value")
		bindStrChange(ctx, &t.OnChange)
		return t
	})

	RegisterComponent("ColorPicker", func(ctx DeclarativeContext) Widget {
		c := NewColorPicker()
		bindStrChange(ctx, &c.OnChange)
		return c
	})

	RegisterComponent("Autocomplete", func(ctx DeclarativeContext) Widget {
		a := NewAutocomplete(propsStrArray(ctx.Spec.Props, "suggestions")...)
		a.Placeholder = orStr(propsStr(ctx.Spec.Props, "placeholder"), ctx.Spec.Text)
		bindStrChange(ctx, &a.OnChange)
		return a
	})

	RegisterComponent("Mention", func(ctx DeclarativeContext) Widget {
		m := NewMention(propsStrArray(ctx.Spec.Props, "users")...)
		m.Placeholder = orStr(propsStr(ctx.Spec.Props, "placeholder"), ctx.Spec.Text)
		m.Width = parseFloat(ctx.Spec.Props["width"], m.Width)
		bindStrChange(ctx, &m.OnChange)
		return m
	})

	RegisterComponent("InputTag", func(ctx DeclarativeContext) Widget {
		it := NewInputTag(propsStrArray(ctx.Spec.Props, "tags")...)
		if p := propsStr(ctx.Spec.Props, "placeholder"); p != "" {
			it.Placeholder = p
		}
		return it
	})

	RegisterComponent("Upload", func(ctx DeclarativeContext) Widget {
		u := NewUpload()
		u.Tip = orStr(propsStr(ctx.Spec.Props, "tip"), u.Tip)
		u.Files = propsStrArray(ctx.Spec.Props, "files")
		if h, ok := ctx.Spec.Events["upload"]; ok {
			if fn, ok := ctx.Handlers[h]; ok {
				u.OnUpload = func() { fn(EventContext{Name: "upload", Target: u}) }
			}
		}
		return u
	})

	RegisterComponent("Cascader", func(ctx DeclarativeContext) Widget {
		c := NewCascader(parseCascaderOptions(propsArray(ctx.Spec.Props, "options")))
		c.Placeholder = orStr(propsStr(ctx.Spec.Props, "placeholder"), ctx.Spec.Text)
		c.Value = propsStrArray(ctx.Spec.Props, "value")
		c.Width = parseFloat(ctx.Spec.Props["width"], c.Width)
		return c
	})

	RegisterComponent("SelectV2", func(ctx DeclarativeContext) Widget {
		s := NewSelectV2(parseSelectOptions(propsArray(ctx.Spec.Props, "options")))
		s.Placeholder = orStr(propsStr(ctx.Spec.Props, "placeholder"), ctx.Spec.Text)
		s.Value = propsStr(ctx.Spec.Props, "value")
		s.Width = parseFloat(ctx.Spec.Props["width"], s.Width)
		bindStrChange(ctx, &s.OnChange)
		return s
	})

	RegisterComponent("TreeSelect", func(ctx DeclarativeContext) Widget {
		t := NewTreeSelect(parseTreeNodes(propsArray(ctx.Spec.Props, "nodes"))...)
		t.Placeholder = orStr(propsStr(ctx.Spec.Props, "placeholder"), ctx.Spec.Text)
		t.Value = propsStr(ctx.Spec.Props, "value")
		t.Width = parseFloat(ctx.Spec.Props["width"], t.Width)
		bindStrChange(ctx, &t.OnChange)
		return t
	})

	RegisterComponent("TreeV2", func(ctx DeclarativeContext) Widget {
		return NewTreeV2(parseTreeNodes(propsArray(ctx.Spec.Props, "nodes"))...)
	})

	RegisterComponent("Form", func(ctx DeclarativeContext) Widget {
		var items []*FormItem
		for _, m := range propsArray(ctx.Spec.Props, "items") {
			items = append(items, &FormItem{
				Label: propStr(m, "label"), Placeholder: propStr(m, "placeholder"),
				Prop: propStr(m, "prop"), Required: parseBool(m["required"], false),
			})
		}
		f := NewForm(items...)
		f.LabelWidth = parseFloat(ctx.Spec.Props["labelWidth"], f.LabelWidth)
		return f
	})

	RegisterComponent("Textarea", func(ctx DeclarativeContext) Widget {
		rows := int(parseFloat(ctx.Spec.Props["rows"], 3))
		in := NewTextarea(orStr(propsStr(ctx.Spec.Props, "placeholder"), ctx.Spec.Text), rows, nil)
		if h, ok := ctx.Spec.Events["change"]; ok {
			if fn, ok := ctx.Handlers[h]; ok {
				in.OnTextChanged = func(s string) { fn(EventContext{Name: "change", Target: in, Data: s}) }
			}
		}
		return in
	})

	RegisterComponent("Markdown", func(ctx DeclarativeContext) Widget {
		return NewMarkdownView(orStr(propsStr(ctx.Spec.Props, "source"), ctx.Spec.Text), MarkdownStyle{})
	})

	// ── 收尾：日期时间 / 日历 / 容器 / 编辑器 ──

	RegisterComponent("DateTimePicker", func(ctx DeclarativeContext) Widget {
		d := NewDateTimePicker()
		d.Value = propsStr(ctx.Spec.Props, "value")
		bindStrChange(ctx, &d.OnChange)
		return d
	})

	RegisterComponent("Calendar", func(ctx DeclarativeContext) Widget {
		c := NewCalendar()
		c.Value = propsStr(ctx.Spec.Props, "value")
		if h, ok := ctx.Spec.Events["select"]; ok {
			if fn, ok := ctx.Handlers[h]; ok {
				c.OnSelect = func(s string) { fn(EventContext{Name: "select", Target: c, Data: s}) }
			}
		}
		return c
	})

	RegisterComponent("ConfigProvider", func(ctx DeclarativeContext) Widget {
		var child Widget
		if len(ctx.Children) > 0 {
			child = ctx.Children[0]
		}
		cp := NewConfigProvider(child)
		cp.PrimaryColor = parseColor(propsStr(ctx.Spec.Props, "primaryColor"))
		cp.TextColor = parseColor(propsStr(ctx.Spec.Props, "textColor"))
		return cp
	})

	RegisterComponent("Affix", func(ctx DeclarativeContext) Widget {
		var child Widget
		if len(ctx.Children) > 0 {
			child = ctx.Children[0]
		}
		a := NewAffix(child)
		a.Offset = parseFloat(ctx.Spec.Props["offset"], 0)
		return a
	})

	RegisterComponent("RichText", func(ctx DeclarativeContext) Widget {
		r := NewRichText(orStr(propsStr(ctx.Spec.Props, "text"), ctx.Spec.Text))
		r.Width = parseFloat(ctx.Spec.Props["width"], r.Width)
		r.Height = parseFloat(ctx.Spec.Props["height"], r.Height)
		return r
	})

	RegisterComponent("CodeEditor", func(ctx DeclarativeContext) Widget {
		return NewCodeEditor(orStr(propsStr(ctx.Spec.Props, "language"), "go"), orStr(propsStr(ctx.Spec.Props, "value"), ctx.Spec.Text))
	})

	// 代码表格化编辑器：内容即源码文本，故能声明式——CodeWorkbench 直接吃源码，
	// StructEditor 经 provider.Parse(源码)→表格(C9 的去语言化让它跟语言无关)。

	RegisterComponent("CodeWorkbench", func(ctx DeclarativeContext) Widget {
		wb := NewCodeWorkbench(orStr(propsStr(ctx.Spec.Props, "source"), ctx.Spec.Text))
		if lang := propsStr(ctx.Spec.Props, "language"); lang != "" {
			wb.WithLang(lang)
		}
		return wb
	})

	RegisterComponent("StructEditor", func(ctx DeclarativeContext) Widget {
		lang := orStr(propsStr(ctx.Spec.Props, "language"), "ey")
		src := orStr(propsStr(ctx.Spec.Props, "source"), ctx.Spec.Text)
		prog := &SEProgram{}
		if p := providerFor(lang); p != nil && src != "" { // 源码文本 → 表格模型
			if parsed, err := p.Parse(src); err == nil && parsed != nil {
				prog = parsed
			}
		}
		return NewStructEditor(prog).WithLang(lang)
	})
}

// bindStrChange 把 "change" 事件绑定到 func(string) 型回调（DatePicker/Select 等通用）。
func bindStrChange(ctx DeclarativeContext, fn *func(string)) {
	if h, ok := ctx.Spec.Events["change"]; ok {
		if handler, ok := ctx.Handlers[h]; ok {
			*fn = func(s string) { handler(EventContext{Name: "change", Data: s}) }
		}
	}
}

// parseSelectOptions 解析下拉选项数组。
func parseSelectOptions(arr []map[string]any) []SelectOption {
	out := make([]SelectOption, 0, len(arr))
	for _, m := range arr {
		out = append(out, SelectOption{
			Label: propStr(m, "label"), Value: propStr(m, "value"),
			Disabled: parseBool(m["disabled"], false), Icon: propStr(m, "icon"),
		})
	}
	return out
}

// parseCascaderOptions 递归解析级联选项。
func parseCascaderOptions(arr []map[string]any) []CascaderOption {
	out := make([]CascaderOption, 0, len(arr))
	for _, m := range arr {
		out = append(out, CascaderOption{
			Label: propStr(m, "label"), Value: propStr(m, "value"), Icon: propStr(m, "icon"),
			Children: parseCascaderOptions(propsArray(m, "children")),
		})
	}
	return out
}

// parseTreeNodes 递归解析树节点。
func parseTreeNodes(arr []map[string]any) []*TreeNode {
	out := make([]*TreeNode, 0, len(arr))
	for _, m := range arr {
		out = append(out, &TreeNode{
			Label: propStr(m, "label"), Expanded: parseBool(m["expanded"], false), Selected: parseBool(m["selected"], false),
			Children: parseTreeNodes(propsArray(m, "children")),
		})
	}
	return out
}

// parseMenuItems 递归解析菜单项数组（支持 children 子菜单）。
func parseMenuItems(arr []map[string]any) []MenuItem {
	var items []MenuItem
	for _, m := range arr {
		it := MenuItem{
			Label:    propStr(m, "label"),
			Icon:     propStr(m, "icon"),
			Enabled:  parseBool(m["enabled"], true),
			Children: parseMenuItems(propsArray(m, "children")),
		}
		items = append(items, it)
	}
	return items
}
