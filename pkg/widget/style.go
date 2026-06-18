package widget

import (
	"reflect"

	"github.com/hoonfeng/goui/internal/layout"
	"github.com/hoonfeng/goui/pkg/canvas"
	"github.com/hoonfeng/goui/pkg/paint"
	"github.com/hoonfeng/goui/pkg/types"
)

// ──────────────────────────────────────────────
// Style — CSS 风格的样式声明系统
// ──────────────────────────────────────────────

// Style 定义 CSS 风格的样式声明。
// 支持类似 CSS 的盒模型、颜色、排版等属性，可全局命名复用。
//
// 基本用法:
//
//	// 1. 定义可复用的样式类（类似 CSS class）
//	widget.Define("card", widget.Style{
//	    Padding: types.EdgeInsets(16),
//	    BackgroundColor: types.ColorRef(255, 255, 255),
//	})
//
//	// 2. 在组件中使用
//	widget.Div(widget.Class("card"),
//	    widget.H2("标题"),
//	    widget.P("内容"),
//	)
//
//	// 3. 内联样式
//	widget.Div(widget.Style{Padding: types.EdgeInsets(8)},
//	    widget.Text("Hello"),
//	)
type Style struct {
	// ── 盒模型（Box Model） ────────────────
	Padding   types.Inset // 内边距（零值=不设置）
	Margin    types.Inset // 外边距（零值=不设置）
	Width     float64     // 宽度（0=自动）
	Height    float64     // 高度（0=自动）
	MinWidth  float64     // 最小宽度
	MinHeight float64     // 最小高度
	MaxWidth  float64     // 最大宽度
	MaxHeight float64     // 最大高度
	Gap       float64     // flex 子项间距（CSS gap）

	// ── Flex 布局（L2） ────────────────────
	FlexDirection  string // "row"/"column"（默认 column）
	JustifyContent string // "start"/"center"/"end"/"space-between"/"space-around"
	AlignItems     string // "start"/"center"/"end"/"stretch"
	Overflow       string // "visible"(默认)/"hidden"/"scroll"/"clip"

	// ── 绝对定位（L3） ────────────────────
	Position string   // "relative"/"absolute"
	Top      *float64 // 绝对定位偏移（nil=auto，指针区分 0 与未设）
	Right    *float64
	Bottom   *float64
	Left     *float64
	ZIndex   int

	// ── 背景与边框 ─────────────────────────
	BackgroundColor *types.Color    // 背景色（nil=不设置）
	BorderColor     *types.Color    // 边框颜色（nil=不设置）
	BorderWidth     float64         // 边框宽度（0=不设置）
	BorderRadius    float64         // 圆角半径（0=不设置）
	Gradient        *paint.Gradient // 背景渐变（覆盖 BackgroundColor）
	Shadow          *paint.Shadow   // 盒阴影

	// ── 排版 ──────────────────────────────
	TextColor     *types.Color      // 文字颜色（nil=不设置）
	FontSize      float64           // 字号（0=不设置）
	FontFamily    string            // 字体族（空=不设置）
	FontWeight    canvas.FontWeight // 字重（0=常规）
	LineHeight    float64           // 行高（0=用字体默认）
	TextAlign     canvas.TextHAlign // 文字水平对齐（0=左）
	LetterSpacing float64           // 字间距

	// ── 其他视觉 ──────────────────────────
	Opacity *float64 // 不透明度 0~1（nil=不设；0 有意义故用指针）
	Cursor  string   // 光标样式（如 "pointer"/"text"）

	// ── 伪类（CSS :hover / :focus / :active）──
	// 状态样式覆盖：进入对应状态时，其中已设的视觉属性
	//（BackgroundColor / BorderColor / BorderWidth / BorderRadius / Shadow / Opacity）
	// 覆盖基础样式。生效优先级 hover < focus < active。
	Hover  *Style // :hover  鼠标悬停
	Focus  *Style // :focus  获得焦点
	Active *Style // :active 鼠标按下
}

// styleRegistry 存储所有已命名的样式类。
// 类似 CSS 的 .className { ... } 定义。
var styleRegistry = map[string]Style{}

// ── 样式定义与引用 ────────────────────────────

// Define 注册一个可复用的命名样式类（类似 CSS .class { ... }）。
//
//	在应用初始化时集中定义所有样式类：
//	  widget.Define("card", widget.Style{...})
//	  widget.Define("btn", widget.Style{...})
//
//	使用时通过 Class() 引用：
//	  widget.Div(widget.Class("card"), ...)
func Define(name string, style Style) {
	styleRegistry[name] = style
}

// Class 返回一个命名样式类的引用。
// 必须在 Define() 之后调用，否则返回空样式。
//
//	用法: widget.Div(widget.Class("card"), widget.P("内容"))
func Class(name string) Style {
	if s, ok := styleRegistry[name]; ok {
		return s
	}
	return Style{}
}

// Merge 合并多个 Style，后面的覆盖前面的同名属性。
// 用于组合多个样式类或覆写特定属性。
//
//	style := widget.Merge(widget.Class("card"), widget.Style{BorderRadius: 12})
func Merge(styles ...Style) Style {
	result := Style{}
	for _, s := range styles {
		result = mergeStyle(result, s)
	}
	return result
}

// mergeStyle 把 src 中"已设置"(非零/非 nil)的字段覆盖到 dst。
// 反射式实现：新增任何 Style 字段都自动支持合并，无需在此逐字段加 if
// —— 这正是把"加一个 CSS 属性要改多处"的横切债一次还清。
func mergeStyle(dst, src Style) Style {
	d := reflect.ValueOf(&dst).Elem()
	s := reflect.ValueOf(src)
	for i := 0; i < s.NumField(); i++ {
		if !s.Field(i).IsZero() {
			d.Field(i).Set(s.Field(i))
		}
	}
	return dst
}

// ── 样式应用 ──────────────────────────────────

// applyToContainer 将盒模型和视觉样式应用到 Container。
func (s Style) applyToContainer(c *Container) {
	if s.Padding.Left != 0 || s.Padding.Right != 0 ||
		s.Padding.Top != 0 || s.Padding.Bottom != 0 {
		c.Padding = s.Padding
	}
	if s.Margin.Left != 0 || s.Margin.Right != 0 ||
		s.Margin.Top != 0 || s.Margin.Bottom != 0 {
		c.Margin = s.Margin
	}
	if s.BackgroundColor != nil {
		c.Background = &PaintWidget{Color: s.BackgroundColor}
	}
	if s.BorderColor != nil {
		bw := s.BorderWidth
		if bw == 0 {
			bw = 1
		}
		border := paint.AllBorder(*s.BorderColor, bw)
		c.Border = &border
	}
	// CSS 伪类：原样挂到 Container，由 ContainerElement 在 Paint 时按状态解析覆盖
	c.Hover = s.Hover
	c.Focus = s.Focus
	c.Active = s.Active
	if s.Width != 0 {
		c.Width = s.Width
	}
	if s.Height != 0 {
		c.Height = s.Height
	}
	if s.MaxWidth != 0 {
		c.MaxWidth = s.MaxWidth
	}
	if s.MaxHeight != 0 {
		c.MaxHeight = s.MaxHeight
	}
	if s.BorderRadius != 0 {
		c.BorderRadius = s.BorderRadius
	}
	if s.Gradient != nil {
		c.Gradient = s.Gradient
	}
	if s.Shadow != nil {
		c.Shadow = s.Shadow
	}
	if s.Opacity != nil {
		c.Opacity = s.Opacity
	}
	if s.Overflow == "hidden" || s.Overflow == "scroll" || s.Overflow == "clip" {
		c.ClipContent = true
	}
}

// applyToText 将排版样式应用到 Text。
func (s Style) applyToText(t *Text) {
	if s.TextColor != nil {
		t.Color = *s.TextColor
	}
	if s.FontFamily != "" {
		t.Font.Family = s.FontFamily
	}
	if s.FontSize > 0 {
		t.Font.Size = s.FontSize
	}
	if s.FontWeight != 0 {
		t.Font.Weight = s.FontWeight
	}
	if s.TextAlign != 0 {
		t.Align = s.TextAlign
	}
	if s.LineHeight > 0 {
		t.LineHeight = s.LineHeight
	}
}

// applyToButton 将视觉和排版样式应用到 Button。
// Button 的 Color 字段表示背景色（填充色）。
func (s Style) applyToButton(b *Button) {
	if s.BackgroundColor != nil {
		b.Color = *s.BackgroundColor
	}
	if s.TextColor != nil {
		b.TextColor = *s.TextColor
	}
	if s.FontSize > 0 {
		b.FontSize = s.FontSize
	}
	// CSS 伪类：:hover 覆盖悬停背景色、:focus 覆盖聚焦环色
	if s.Hover != nil && s.Hover.BackgroundColor != nil {
		b.HoverColor = *s.Hover.BackgroundColor
	}
	if s.Focus != nil {
		if s.Focus.BorderColor != nil {
			b.FocusRingColor = *s.Focus.BorderColor
		} else if s.Focus.BackgroundColor != nil {
			b.FocusRingColor = *s.Focus.BackgroundColor
		}
	}
}

// applyToInput 将排版与交互态样式应用到 Input。
func (s Style) applyToInput(in *Input) {
	if s.TextColor != nil {
		in.Color = *s.TextColor
	}
	if s.FontSize > 0 {
		in.Font.Size = s.FontSize
	}
	if s.BorderColor != nil {
		in.BorderColor = *s.BorderColor
	}
	// CSS 伪类：:focus / :hover 覆盖对应交互态边框色（组件级聚焦/悬停定制）
	if s.Focus != nil && s.Focus.BorderColor != nil {
		in.FocusBorderColor = *s.Focus.BorderColor
	}
	if s.Hover != nil && s.Hover.BorderColor != nil {
		in.HoverBorderColor = *s.Hover.BorderColor
	}
}

// applyToCheckbox 将样式应用到 Checkbox。
func (s Style) applyToCheckbox(cb *Checkbox) {
	if s.TextColor != nil {
		cb.LabelColor = *s.TextColor
	}
}

// ── 查询方法 ──────────────────────────────────

// HasBoxModel 返回 Style 是否包含盒模型属性。
func (s Style) HasBoxModel() bool {
	return s.Padding.Left != 0 || s.Padding.Right != 0 ||
		s.Padding.Top != 0 || s.Padding.Bottom != 0 ||
		s.Margin.Left != 0 || s.Margin.Right != 0 ||
		s.Margin.Top != 0 || s.Margin.Bottom != 0 ||
		s.Width != 0 || s.Height != 0 ||
		s.MinWidth != 0 || s.MinHeight != 0
}

// HasVisual 返回 Style 是否包含视觉效果。
func (s Style) HasVisual() bool {
	return s.BackgroundColor != nil || s.BorderColor != nil ||
		s.BorderRadius != 0 || s.BorderWidth != 0 ||
		s.Gradient != nil || s.Shadow != nil || s.Opacity != nil ||
		s.Hover != nil || s.Focus != nil || s.Active != nil // 伪类需 Container 承载状态
}

// HasTypography 返回 Style 是否包含排版属性。
func (s Style) HasTypography() bool {
	return s.TextColor != nil || s.FontSize > 0 || s.FontFamily != "" ||
		s.FontWeight != 0 || s.LineHeight != 0 || s.TextAlign != 0 || s.LetterSpacing != 0
}

// hasInteraction 返回 Style 是否含 CSS 伪类（:hover / :focus / :active）。
func (s Style) hasInteraction() bool {
	return s.Hover != nil || s.Focus != nil || s.Active != nil
}

// needsContainer 返回 Style 是否需要 Container 包装。
// 如果只有排版属性，不需要 Container，可以直接应用到子组件。
func (s Style) needsContainer() bool {
	return s.HasBoxModel() || s.HasVisual()
}

// ── 便捷构造器 ────────────────────────────────

// InlineStyle 快速创建内联样式的辅助函数。
// 比直接写 Style{} 更短的别名。
func InlineStyle(padding float64, bgColor *types.Color) Style {
	s := Style{}
	if padding > 0 {
		s.Padding = types.EdgeInsets(padding)
	}
	if bgColor != nil {
		s.BackgroundColor = bgColor
	}
	return s
}

// ── 内部：StyledWidget（带样式的组件包装器） ────

// StyledWidget 是带样式声明的基础容器。
// 由 Div/Styled 内部创建，将 Style 应用于子组件。
type StyledWidget struct {
	StatelessWidget
	Style    Style
	Children []Widget
}

// CreateElement 创建 StyledElement
func (sw *StyledWidget) CreateElement() Element {
	return &StyledElement{
		BaseElement: BaseElement{widget: sw},
		styled:      sw,
	}
}

// Update 更新 Widget 配置并同步内部 styled 引用。
func (e *StyledElement) Update(newWidget Widget) {
	e.widget = newWidget
	e.dirty = true
	if s, ok := newWidget.(*StyledWidget); ok {
		e.styled = s
	}
}

// StyledElement StyledWidget 的运行时 Element
type StyledElement struct {
	BaseElement
	styled   *StyledWidget
	elements []Element
}

// Build 构建子 Element
// 复用已有的 Container Element，避免丢失运行时状态。
func (e *StyledElement) Build() []Element {
	sw := e.styled
	style := sw.Style

	// 如果有盒模型或视觉效果，用 Container 包装
	if style.needsContainer() {
		container := &Container{
			SingleChildWidget: SingleChildWidget{},
		}
		style.applyToContainer(container)

		if sw.Children == nil || len(sw.Children) == 0 {
			// 清理旧的子元素
			for _, child := range e.children {
				child.Unmount()
			}
			e.children = nil
			return nil
		}

		// 将子组件放入 Flex 布局
		if len(sw.Children) == 1 {
			container.Child = applyStyleToWidget(sw.Children[0], style)
		} else {
			styledChildren := make([]Widget, len(sw.Children))
			for i, w := range sw.Children {
				styledChildren[i] = applyStyleToWidget(w, style)
			}
			container.Child = VBox(styledChildren...)
		}

		// 复用已存在的 Container Element（当 Widget 类型匹配时）
		containerWidget := Widget(container)
		if len(e.children) == 1 && e.children[0].WidgetType() == reflect.TypeOf(containerWidget) {
			e.children[0].Update(containerWidget)
		} else {
			for _, child := range e.children {
				child.Unmount()
			}
			el := CreateElementFor(containerWidget)
			el.Mount(e, 0)
			e.children = []Element{el}
		}
		return e.children
	}

	// 没有盒模型属性：直接创建子组件列表
	// 尝试复用旧的子 Element
	oldChildren := e.children
	if len(oldChildren) == len(sw.Children) {
		allMatch := true
		for i, w := range sw.Children {
			styledChild := applyStyleToWidget(w, style)
			if i < len(oldChildren) && oldChildren[i].WidgetType() == reflect.TypeOf(styledChild) {
				oldChildren[i].Update(styledChild)
			} else {
				allMatch = false
				break
			}
		}
		if allMatch {
			return e.children
		}
	}

	// 数量不匹配或类型不匹配：全部重建
	for _, child := range oldChildren {
		child.Unmount()
	}
	e.children = make([]Element, 0, len(sw.Children))
	for _, w := range sw.Children {
		styledChild := applyStyleToWidget(w, style)
		el := CreateElementFor(styledChild)
		el.Mount(e, len(e.children))
		e.children = append(e.children, el)
	}
	return e.children
}

// Layout 默认布局（将约束传递给子组件）
func (e *StyledElement) Layout(ctx *layout.LayoutContext) layout.LayoutResult {
	// 如果已包装为 Container，子组件已完成布局
	if len(e.children) > 0 {
		// 如果是 Container 包装模式，让 Container 完成布局
		child := e.children[0]
		child.Layout(ctx)
		e.size = child.Size()
		return layout.LayoutResult{Size: e.size}
	}
	e.size = types.Size{}
	return layout.LayoutResult{Size: e.size}
}

// Paint 绘制
func (e *StyledElement) Paint(cvs canvas.Canvas, offset types.Point) {
	for _, child := range e.children {
		child.Paint(cvs, offset)
	}
}

// applyStyleToWidget 根据 Widget 类型应用样式属性。
// Text → 应用排版样式
// Button → 应用视觉+排版样式
// 其他 → 原样返回（样式由父容器处理）
func applyStyleToWidget(w Widget, style Style) Widget {
	if !style.HasTypography() && !style.hasInteraction() {
		return w
	}
	switch v := w.(type) {
	case *Text:
		style.applyToText(v)
		return v
	case *Input:
		style.applyToInput(v)
		return v
	case *Button:
		style.applyToButton(v)
		return v
	case *Checkbox:
		style.applyToCheckbox(v)
		return v
	}
	// CSS 伪类：内嵌 pseudoColors 的组件（Select 系触发器）接受交互态色覆盖
	if style.hasInteraction() {
		if is, ok := w.(interactionStyler); ok {
			is.applyInteraction(style)
		}
	}
	return w
}

// Styled 将 Style 应用到 Widget 上，返回带样式的组件。
// 类似于 CSS 中给元素设置 class 或 style 属性。
//
//	// 单个子组件
//	styledBtn := widget.Styled(myButton, widget.Class("btn-primary"))
//
//	// 多个子组件（用 VBox 包裹）
//	styledBox := widget.Styled(widget.VBox(text1, text2), widget.Style{Padding: types.EdgeInsets(8)})
func Styled(child Widget, style Style) Widget {
	return &StyledWidget{
		Style:    style,
		Children: []Widget{child},
	}
}
