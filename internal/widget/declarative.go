// Package widget — 声明式配置系统
//
// 此文件提供声明式 UI 构建系统，支持通过 JSON 配置文件或 Go 结构体
// 创建完整的 UI 组件树，并支持事件绑定的声明式配置。
//
// 设计目标：
//  1. 将 UI 结构与逻辑分离 — UI 描述为纯数据，事件处理在 Go 代码中
//  2. 无需重编译即可修改 UI — 外部 JSON 配置热加载
//  3. 与现有组件系统完全兼容 — 所有内置组件均已注册
//  4. 为 Web 组件模型铺路 — 配置格式可直接映射到 Web Component 的 props/events/slots
//
// ──────────────────────────────────────────────
//
// 快速开始:
//
//	// 1. 定义事件处理器
//	handlers := widget.Handlers{
//	    "onClick": func(ctx widget.EventContext) {
//	        log.Println("Button clicked!", ctx.Name)
//	    },
//	}
//
//	// 2a. 从 JSON 加载
//	ui, err := widget.LoadConfig([]byte(`{
//	    "type": "Column",
//	    "class": "page",
//	    "children": [
//	        {"type": "H1", "text": "Hello"},
//	        {"type": "Button", "text": "Click", "events": {"click": "onClick"}}
//	    ]
//	}`), handlers)
//
//	// 2b. 或从 Go 结构体构建（同上格式，不走 JSON 序列化）
//	ui := widget.BuildFromSpec(widget.ComponentSpec{
//	    Type:  "Column",
//	    Class: "page",
//	    Children: []widget.ComponentSpec{
//	        {Type: "H1", Text: "Hello"},
//	        {Type: "Button", Text: "Click", Events: map[string]string{"click": "onClick"}},
//	    },
//	}, handlers)
//
//	// 3. 设置到应用
//	app.SetRootWidget(ui)
//
// ──────────────────────────────────────────────
package widget

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/user/goui/internal/canvas"
	"github.com/user/goui/internal/layout"
	"github.com/user/goui/internal/paint"
	"github.com/user/goui/internal/types"
)

// ─── 类型定义 ────────────────────────────────────

// ComponentSpec 描述一个 UI 组件的完整配置。
// 对应 JSON 中的一个节点，支持嵌套 children 形成组件树。
//
// JSON 示例:
//
//	{
//	  "type": "Column",
//	  "class": "card",
//	  "text": "可选的文本内容",
//	  "style": { "padding": "16", "backgroundColor": "#FFFFFF" },
//	  "props": {
//	    "checked": true,
//	    "minWidth": 100
//	  },
//	  "events": {
//	    "click": "handleClick",
//	    "change": "handleChange"
//	  },
//	  "children": [
//	    { "type": "Text", "text": "子组件" }
//	  ]
//	}
type ComponentSpec struct {
	Type     string            `json:"type"`               // 组件类型名（必填，如 "Button", "Div", "Column"）
	Class    string            `json:"class,omitempty"`    // 引用的样式类名（通过 Define() 注册）
	Text     string            `json:"text,omitempty"`     // 文本内容（对 Text/Button/Input 等生效）
	Style    *StyleSpec        `json:"style,omitempty"`    // 内联样式（可选，覆盖 Class 样式）
	Props    map[string]any    `json:"props,omitempty"`    // 组件特有属性（如 checked、value、minWidth 等）
	Events   map[string]string `json:"events,omitempty"`   // 事件绑定（事件名 → 处理器名称）
	Children []ComponentSpec   `json:"children,omitempty"` // 子组件列表
}

// StyleSpec 是 JSON 兼容的样式声明，与 Style 结构体对应。
// 支持 CSS 风格的盒模型、颜色、排版属性。
type StyleSpec struct {
	Padding         string  `json:"padding,omitempty"`         // "all" 或 "top right bottom left"
	Margin          string  `json:"margin,omitempty"`          // "all" 或 "top right bottom left"
	Width           float64 `json:"width,omitempty"`           // 宽度（0=自动）
	Height          float64 `json:"height,omitempty"`          // 高度（0=自动）
	MinWidth        float64 `json:"minWidth,omitempty"`        // 最小宽度
	MinHeight       float64 `json:"minHeight,omitempty"`       // 最小高度
	MaxWidth        float64 `json:"maxWidth,omitempty"`        // 最大宽度
	MaxHeight       float64 `json:"maxHeight,omitempty"`       // 最大高度
	Gap             float64 `json:"gap,omitempty"`             // flex 子项间距
	FlexDirection   string  `json:"flexDirection,omitempty"`   // "row"/"column"
	JustifyContent  string  `json:"justifyContent,omitempty"`  // 主轴对齐
	AlignItems      string  `json:"alignItems,omitempty"`      // 交叉轴对齐
	Overflow        string   `json:"overflow,omitempty"`        // "hidden"/"scroll"/"visible"
	Position        string   `json:"position,omitempty"`        // "relative"/"absolute"
	Top             *float64 `json:"top,omitempty"`
	Right           *float64 `json:"right,omitempty"`
	Bottom          *float64 `json:"bottom,omitempty"`
	Left            *float64 `json:"left,omitempty"`
	ZIndex          int      `json:"zIndex,omitempty"`
	BackgroundColor string  `json:"backgroundColor,omitempty"` // 背景色（hex: #RRGGBB 或 #RRGGBBAA）
	TextColor       string  `json:"textColor,omitempty"`       // 文字颜色（hex）
	BorderColor     string  `json:"borderColor,omitempty"`     // 边框颜色（hex）
	BorderWidth     float64 `json:"borderWidth,omitempty"`     // 边框宽度
	BorderRadius    float64 `json:"borderRadius,omitempty"`    // 圆角半径
	Shadow          string  `json:"shadow,omitempty"`          // 阴影 "offsetX offsetY blur #color"
	Opacity         float64 `json:"opacity,omitempty"`         // 不透明度 0~1
	FontSize        float64 `json:"fontSize,omitempty"`        // 字号
	FontFamily      string  `json:"fontFamily,omitempty"`      // 字体族
	FontWeight      string  `json:"fontWeight,omitempty"`      // "bold"/"normal"
	TextAlign       string  `json:"textAlign,omitempty"`       // "left"/"center"/"right"
	LineHeight      float64 `json:"lineHeight,omitempty"`      // 行高
	LetterSpacing   float64 `json:"letterSpacing,omitempty"`   // 字间距

	// CSS 伪类（嵌套样式）—— 进入对应状态时覆盖其中已设的视觉属性
	Hover  *StyleSpec `json:"hover,omitempty"`  // :hover  鼠标悬停
	Focus  *StyleSpec `json:"focus,omitempty"`  // :focus  获得焦点
	Active *StyleSpec `json:"active,omitempty"` // :active 鼠标按下
}

// EventHandler 声明式事件处理函数。
// 通过 Handlers 映射注册，供组件配置中的 events 字段引用。
type EventHandler func(ctx EventContext)

// EventContext 提供事件触发时的上下文信息。
type EventContext struct {
	// Name 事件名称（如 "click", "change", "submit"）
	Name string
	// Target 触发事件的组件 Widget 实例
	Target interface{}
	// Data 事件携带的特定数据（如 Checkbox 的 bool 值、Input 的文本等）
	Data interface{}
}

// Handlers 是事件处理器名称到处理函数的映射表。
// 在 LoadConfig / BuildFromSpec 时传入，供组件通过 events 字段引用。
type Handlers map[string]EventHandler

// ComponentFactory 是组件工厂函数，根据 ComponentSpec 创建 Widget。
// ctx 提供完整的配置上下文（spec、handlers、已构建的子组件列表）。
type ComponentFactory func(ctx DeclarativeContext) Widget

// DeclarativeContext 组件构建上下文
type DeclarativeContext struct {
	Spec     ComponentSpec // 当前组件的原始配置
	Handlers Handlers      // 事件处理器映射表
	Children []Widget      // 已构建的子组件列表
}

// ─── 组件注册表 ──────────────────────────────────

// componentRegistry 存储所有已注册的组件类型名到工厂函数的映射。
// 键为类型名（如 "Button", "Column"），值是对应的工厂函数。
// 注册后可通过 ComponentSpec.Type 引用。
var componentRegistry = map[string]ComponentFactory{}

// RegisterComponent 注册一个组件类型。
//
//	name: 组件类型名，在 ComponentSpec.Type 中使用（如 "Button", "MyWidget"）
//	factory: 组件工厂函数，接收 DeclarativeContext 返回 Widget
//
// 内置组件（Button、Text、Div 等）已在 init() 中自动注册。
// 用户可使用此函数注册自定义组件。
func RegisterComponent(name string, factory ComponentFactory) {
	if factory == nil {
		panic("declarative: cannot register nil factory for type: " + name)
	}
	if _, exists := componentRegistry[name]; exists {
		panic("declarative: duplicate component registration: " + name)
	}
	componentRegistry[name] = factory
}

// ─── 核心构建函数 ────────────────────────────────

// BuildFromSpec 根据 ComponentSpec 构建完整的 Widget 组件树。
// 递归处理所有子组件，绑定事件处理器。
//
// 用法:
//
//	ui := widget.BuildFromSpec(widget.ComponentSpec{
//	    Type: "Column",
//	    Children: []widget.ComponentSpec{
//	        {Type: "H1", Text: "Title"},
//	        {Type: "Button", Text: "OK", Events: map[string]string{"click": "handler"}},
//	    },
//	}, handlers)
func BuildFromSpec(spec ComponentSpec, handlers Handlers) Widget {
	return buildComponent(spec, handlers)
}

// LoadConfig 从 JSON 数据加载并构建 UI 组件树。
//
//	data: JSON 格式的组件配置（UTF-8 编码）
//	handlers: 事件处理器映射表（配置中的事件名 → 处理函数）
//
// 返回构建完成的根 Widget，可在 app.SetRootWidget() 中使用。
func LoadConfig(data []byte, handlers Handlers) (Widget, error) {
	var spec ComponentSpec
	if err := json.Unmarshal(data, &spec); err != nil {
		return nil, fmt.Errorf("declarative: JSON parse error: %w", err)
	}
	if spec.Type == "" {
		return nil, fmt.Errorf("declarative: root component must have a type")
	}
	return buildComponent(spec, handlers), nil
}

// LoadConfigFile 从 JSON 文件加载并构建 UI 组件树。
//
//	path: JSON 文件路径
//	handlers: 事件处理器映射表
func LoadConfigFile(path string, handlers Handlers) (Widget, error) {
	data, err := readFileFunc(path)
	if err != nil {
		return nil, fmt.Errorf("declarative: read file error: %w", err)
	}
	return LoadConfig(data, handlers)
}

// readFileFunc 是可替换的文件读取函数，默认使用 os.ReadFile。
// 测试时可替换为 mock 函数。
var readFileFunc = os.ReadFile

// buildComponent 递归构建单个组件及其子组件
func buildComponent(spec ComponentSpec, handlers Handlers) Widget {
	// 查找工厂函数
	factory, ok := componentRegistry[spec.Type]
	if !ok {
		// 类型未注册时返回占位文本组件（避免崩溃）
		return &Text{
			Text:  fmt.Sprintf("[unknown: %s]", spec.Type),
			Color: types.ColorRed,
			Font:  canvas.Font{Family: "sans-serif", Size: 14},
		}
	}

	// 递归构建子组件
	children := make([]Widget, 0, len(spec.Children))
	for _, childSpec := range spec.Children {
		child := buildComponent(childSpec, handlers)
		if child != nil {
			children = append(children, child)
		}
	}

	// 调用工厂函数
	ctx := DeclarativeContext{
		Spec:     spec,
		Handlers: handlers,
		Children: children,
	}
	return factory(ctx)
}

// ─── 辅助函数 ────────────────────────────────────

// parseColor 将字符串颜色解析为 *types.Color。
// 支持 hex 格式：#RRGGBB、#RRGGBBAA、#RGB
// 支持命名颜色：transparent、black、white、red、green、blue
func parseColor(s string) *types.Color {
	if s == "" {
		return nil
	}
	switch strings.ToLower(s) {
	case "transparent":
		return &types.Color{}
	case "black":
		return &types.Color{R: 0, G: 0, B: 0, A: 255}
	case "white":
		return &types.Color{R: 255, G: 255, B: 255, A: 255}
	case "red":
		return &types.Color{R: 255, G: 0, B: 0, A: 255}
	case "green":
		return &types.Color{R: 0, G: 255, B: 0, A: 255}
	case "blue":
		return &types.Color{R: 0, G: 0, B: 255, A: 255}
	}
	if len(s) > 0 && s[0] == '#' {
		c := types.ColorFromHex(s)
		return &c
	}
	return nil
}

// parseFloat 解析 float64，失败返回默认值
func parseFloat(v any, defaultVal float64) float64 {
	switch val := v.(type) {
	case float64:
		return val
	case int:
		return float64(val)
	case string:
		if f, err := strconv.ParseFloat(val, 64); err == nil {
			return f
		}
	}
	return defaultVal
}

// parseBool 解析 bool，失败返回默认值
func parseBool(v any, defaultVal bool) bool {
	switch val := v.(type) {
	case bool:
		return val
	case string:
		if b, err := strconv.ParseBool(val); err == nil {
			return b
		}
	}
	return defaultVal
}

// parseInset 解析内边距字符串。
// 格式: "all" | "top&bottom left&right" | "top right bottom left"
func parseInset(s string) types.Inset {
	if s == "" {
		return types.Inset{}
	}
	parts := strings.Fields(s)
	switch len(parts) {
	case 1:
		v := parseFloatStr(parts[0])
		return types.EdgeInsets(v)
	case 2:
		vy := parseFloatStr(parts[0])
		vx := parseFloatStr(parts[1])
		return types.Inset{Top: vy, Right: vx, Bottom: vy, Left: vx}
	case 4:
		return types.Inset{
			Top:    parseFloatStr(parts[0]),
			Right:  parseFloatStr(parts[1]),
			Bottom: parseFloatStr(parts[2]),
			Left:   parseFloatStr(parts[3]),
		}
	}
	return types.Inset{}
}

// parseFloatStr 解析浮点数字符串
func parseFloatStr(s string) float64 {
	if v, err := strconv.ParseFloat(s, 64); err == nil {
		return v
	}
	return 0
}

// ─── StyleSpec → Style 转换 ─────────────────────

// ToStyle 将 StyleSpec 转换为框架内部的 Style 结构体。
func (s *StyleSpec) ToStyle() Style {
	style := Style{}
	if s == nil {
		return style
	}
	if s.Padding != "" {
		style.Padding = parseInset(s.Padding)
	}
	if s.Margin != "" {
		style.Margin = parseInset(s.Margin)
	}
	style.Width = s.Width
	style.Height = s.Height
	style.MinWidth = s.MinWidth
	style.MinHeight = s.MinHeight
	if s.BackgroundColor != "" {
		style.BackgroundColor = parseColor(s.BackgroundColor)
	}
	if s.TextColor != "" {
		style.TextColor = parseColor(s.TextColor)
	}
	if s.BorderColor != "" {
		style.BorderColor = parseColor(s.BorderColor)
	}
	style.BorderWidth = s.BorderWidth
	style.BorderRadius = s.BorderRadius
	style.MaxWidth = s.MaxWidth
	style.MaxHeight = s.MaxHeight
	style.Gap = s.Gap
	style.FlexDirection = s.FlexDirection
	style.JustifyContent = s.JustifyContent
	style.AlignItems = s.AlignItems
	style.Overflow = s.Overflow
	style.Position = s.Position
	style.Top = s.Top
	style.Right = s.Right
	style.Bottom = s.Bottom
	style.Left = s.Left
	style.ZIndex = s.ZIndex
	style.FontSize = s.FontSize
	style.FontFamily = s.FontFamily
	style.LineHeight = s.LineHeight
	style.LetterSpacing = s.LetterSpacing
	if s.Opacity > 0 {
		o := s.Opacity
		style.Opacity = &o
	}
	switch s.FontWeight {
	case "bold", "700", "600", "500":
		style.FontWeight = canvas.FontWeightBold
	}
	switch s.TextAlign {
	case "center":
		style.TextAlign = canvas.HAlignCenter
	case "right":
		style.TextAlign = canvas.HAlignRight
	}
	if s.Shadow != "" {
		style.Shadow = parseShadow(s.Shadow)
	}
	// CSS 伪类递归转换（嵌套 StyleSpec → Style）
	if s.Hover != nil {
		h := s.Hover.ToStyle()
		style.Hover = &h
	}
	if s.Focus != nil {
		f := s.Focus.ToStyle()
		style.Focus = &f
	}
	if s.Active != nil {
		a := s.Active.ToStyle()
		style.Active = &a
	}
	return style
}

// parseShadow 解析 CSS 风格阴影字符串 "offsetX offsetY blur #color" 为 *paint.Shadow。
func parseShadow(s string) *paint.Shadow {
	parts := strings.Fields(s)
	if len(parts) < 4 {
		return nil
	}
	x, _ := strconv.ParseFloat(parts[0], 64)
	y, _ := strconv.ParseFloat(parts[1], 64)
	blur, _ := strconv.ParseFloat(parts[2], 64)
	c := parseColor(parts[3])
	if c == nil {
		return nil
	}
	return &paint.Shadow{Offset: types.Point{X: x, Y: y}, Blur: blur, Color: *c}
}

// applyStyleToWidgetConfig 根据组件类型应用样式到已创建的 Widget。
func applyStyleToWidgetConfig(w Widget, spec ComponentSpec) {
	style := Style{}
	if spec.Class != "" {
		style = Merge(style, Class(spec.Class))
	}
	if spec.Style != nil {
		style = Merge(style, spec.Style.ToStyle())
	}
	if !style.HasTypography() && !style.HasBoxModel() && !style.HasVisual() {
		return
	}
	switch v := w.(type) {
	case *Text:
		style.applyToText(v)
	case *Input:
		style.applyToInput(v)
	case *Checkbox:
		style.applyToCheckbox(v)
	case *Button:
		style.applyToButton(v)
	}
	// CSS 伪类：内嵌 pseudoColors 的组件接受交互态色覆盖
	if style.hasInteraction() {
		if is, ok := w.(interactionStyler); ok {
			is.applyInteraction(style)
		}
	}
}

// ─── 内置组件工厂注册 ────────────────────────────

func init() {
	registerBuiltinComponents()
}

func registerBuiltinComponents() {
	// ── 容器类型 ───────────────────────────

	// Column — 垂直排列容器（等同 VBox/Column）
	RegisterComponent("Column", func(ctx DeclarativeContext) Widget {
		children := ctx.Children
		style := resolveStyle(ctx.Spec)
		if style.needsContainer() {
			return buildStyledContainer("Column", children, style)
		}
		return &Column{
			Flex: Flex{
				MultiChildWidget:   MultiChildWidget{Children: children},
				Direction:          layout.FlexColumn,
				MainAxisAlignment:  parseJustify(style.JustifyContent),
				CrossAxisAlignment: parseAlign(style.AlignItems),
				Gap:                style.Gap,
			},
		}
	})
	RegisterComponent("VBox", func(ctx DeclarativeContext) Widget {
		// VBox 是 Column 的别名
		ctx.Spec.Type = "Column"
		return componentRegistry["Column"](ctx)
	})

	// Row — 水平排列容器（等同 HBox）
	RegisterComponent("Row", func(ctx DeclarativeContext) Widget {
		children := ctx.Children
		style := resolveStyle(ctx.Spec)
		if style.needsContainer() {
			return buildStyledContainer("Row", children, style)
		}
		return &Row{
			Flex: Flex{
				MultiChildWidget:   MultiChildWidget{Children: children},
				Direction:          layout.FlexRow,
				MainAxisAlignment:  parseJustify(style.JustifyContent),
				CrossAxisAlignment: parseAlign(style.AlignItems),
				Gap:                style.Gap,
			},
		}
	})
	RegisterComponent("HBox", func(ctx DeclarativeContext) Widget {
		ctx.Spec.Type = "Row"
		return componentRegistry["Row"](ctx)
	})

	// Div — 块级容器（根据样式自动选择 Flex/Container）
	RegisterComponent("Div", func(ctx DeclarativeContext) Widget {
		children := ctx.Children
		style := resolveStyle(ctx.Spec)
		if !style.needsContainer() {
			if len(children) == 0 {
				return &Container{}
			}
			return &Flex{
				MultiChildWidget:   MultiChildWidget{Children: children},
				Direction:          flexDir(style.FlexDirection),
				MainAxisAlignment:  parseJustify(style.JustifyContent),
				CrossAxisAlignment: parseAlign(style.AlignItems),
				Gap:                style.Gap,
			}
		}
		container := &Container{
			Padding:      style.Padding,
			BorderRadius: style.BorderRadius,
		}
		if style.BackgroundColor != nil {
			container.Background = &PaintWidget{Color: style.BackgroundColor}
		}
		if style.BorderColor != nil {
			bw := style.BorderWidth
			if bw == 0 {
				bw = 1
			}
			border := paint.AllBorder(*style.BorderColor, bw)
			container.Border = &border
		}
		if style.Width != 0 {
			container.Width = style.Width
		}
		if style.Height != 0 {
			container.Height = style.Height
		}
		if len(children) > 0 {
			container.Child = &Flex{
				MultiChildWidget:   MultiChildWidget{Children: children},
				Direction:          flexDir(style.FlexDirection),
				MainAxisAlignment:  parseJustify(style.JustifyContent),
				CrossAxisAlignment: parseAlign(style.AlignItems),
				Gap:                style.Gap,
			}
		}
		return container
	})

	// Section — 语义区块（带默认内边距的 Div）
	RegisterComponent("Section", func(ctx DeclarativeContext) Widget {
		children := ctx.Children
		style := resolveStyle(ctx.Spec)
		if style.Padding.Left == 0 && style.Padding.Right == 0 &&
			style.Padding.Top == 0 && style.Padding.Bottom == 0 {
			style.Padding = types.EdgeInsets(16)
		}
		if style.needsContainer() {
			return buildStyledContainer("Column", children, style)
		}
		return &Column{
			Flex: Flex{
				MultiChildWidget: MultiChildWidget{Children: children},
				Direction:        layout.FlexColumn,
			},
		}
	})

	// Spacer — 弹性空白
	RegisterComponent("Spacer", func(ctx DeclarativeContext) Widget {
		return &Spacer{}
	})

	// ── 文本类型 ───────────────────────────

	// Text — 纯文本
	RegisterComponent("Text", func(ctx DeclarativeContext) Widget {
		t := &Text{
			Text:  ctx.Spec.Text,
			Color: CurrentTheme().TextColor,
			Font:  canvas.DefaultFont(),
		}
		applyStyleToWidgetConfig(t, ctx.Spec)
		applyTextProps(t, ctx.Spec.Props)
		return t
	})

	// H1 — 一级标题
	RegisterComponent("H1", func(ctx DeclarativeContext) Widget {
		t := H1(ctx.Spec.Text)
		applyHeaderStyle(t, ctx.Spec.Style)
		return t
	})

	// H2 — 二级标题
	RegisterComponent("H2", func(ctx DeclarativeContext) Widget {
		t := H2(ctx.Spec.Text)
		applyHeaderStyle(t, ctx.Spec.Style)
		return t
	})

	// H3 — 三级标题
	RegisterComponent("H3", func(ctx DeclarativeContext) Widget {
		t := H3(ctx.Spec.Text)
		applyHeaderStyle(t, ctx.Spec.Style)
		return t
	})

	// H4 — 四级标题
	RegisterComponent("H4", func(ctx DeclarativeContext) Widget {
		t := H4(ctx.Spec.Text)
		applyHeaderStyle(t, ctx.Spec.Style)
		return t
	})

	// P — 段落文本
	RegisterComponent("P", func(ctx DeclarativeContext) Widget {
		t := P(ctx.Spec.Text)
		if ctx.Spec.Style != nil {
			if c := parseColor(ctx.Spec.Style.TextColor); c != nil {
				t.Color = *c
			}
			if ctx.Spec.Style.FontSize > 0 {
				t.Font.Size = ctx.Spec.Style.FontSize
			}
		}
		applyTextProps(t, ctx.Spec.Props)
		return t
	})

	// Small — 小号文本
	RegisterComponent("Small", func(ctx DeclarativeContext) Widget {
		t := Small(ctx.Spec.Text)
		if ctx.Spec.Style != nil {
			if c := parseColor(ctx.Spec.Style.TextColor); c != nil {
				t.Color = *c
			}
			if ctx.Spec.Style.FontSize > 0 {
				t.Font.Size = ctx.Spec.Style.FontSize
			}
		}
		return t
	})

	// Label — 标签（可用于表单字段标签）
	RegisterComponent("Label", func(ctx DeclarativeContext) Widget {
		t := &Text{
			Text:  ctx.Spec.Text,
			Color: CurrentTheme().TextColor,
			Font:  canvas.Font{Family: "sans-serif", Size: 14, Weight: canvas.FontWeightMedium},
		}
		applyStyleToWidgetConfig(t, ctx.Spec)
		return t
	})

	// ── 交互组件 ───────────────────────────

	// Button — 按钮
	RegisterComponent("Button", func(ctx DeclarativeContext) Widget {
		btn := &Button{
			Text:       ctx.Spec.Text,
			Color:      CurrentTheme().Button.DefaultColor,
			HoverColor: CurrentTheme().Button.HoverColor,
			TextColor:  CurrentTheme().Button.TextColor,
			FontSize:   16,
			Padding:    types.EdgeInsetsLTRB(12, 4, 12, 4),
			MinWidth:   CurrentTheme().Button.MinWidth,
			MinHeight:  CurrentTheme().Button.MinHeight,
		}
		// 应用样式
		applyStyleToWidgetConfig(btn, ctx.Spec)
		// 应用 props
		if v, ok := ctx.Spec.Props["color"]; ok {
			if c := parseColor(fmt.Sprintf("%v", v)); c != nil {
				btn.Color = *c
			}
		}
		if v, ok := ctx.Spec.Props["hoverColor"]; ok {
			if c := parseColor(fmt.Sprintf("%v", v)); c != nil {
				btn.HoverColor = *c
			}
		}
		if v, ok := ctx.Spec.Props["textColor"]; ok {
			if c := parseColor(fmt.Sprintf("%v", v)); c != nil {
				btn.TextColor = *c
			}
		}
		btn.MinWidth = parseFloat(ctx.Spec.Props["minWidth"], btn.MinWidth)
		btn.MinHeight = parseFloat(ctx.Spec.Props["minHeight"], btn.MinHeight)
		btn.FontSize = parseFloat(ctx.Spec.Props["fontSize"], btn.FontSize)
		// 绑定事件
		if handlerName, ok := ctx.Spec.Events["click"]; ok {
			if handler, ok := ctx.Handlers[handlerName]; ok {
				btn.OnClick = func() {
					handler(EventContext{
						Name:   "click",
						Target: btn,
					})
				}
			}
		}
		return btn
	})

	// Input — 输入框
	RegisterComponent("Input", func(ctx DeclarativeContext) Widget {
		t := CurrentTheme()
		in := &Input{
			Placeholder:      ctx.Spec.Text,
			Font:             canvas.DefaultFont(),
			Color:            t.Input.TextColor,
			BGColor:          t.Input.BGColor,
			BorderColor:      t.Input.BorderColor,
			FocusBorderColor: t.Input.FocusBorderColor,
			PlaceholderColor: t.Input.PlaceholderColor,
			CursorColor:      t.Input.CursorColor,
			CursorWidth:      t.Input.CursorWidth,
		}
		applyStyleToWidgetConfig(in, ctx.Spec)
		// 应用 props
		if v, ok := ctx.Spec.Props["placeholder"]; ok {
			in.Placeholder = fmt.Sprintf("%v", v)
		}
		if v, ok := ctx.Spec.Props["maxLength"]; ok {
			in.MaxLength = int(parseFloat(v, 1024))
		}
		// 绑定事件
		if handlerName, ok := ctx.Spec.Events["change"]; ok {
			if handler, ok := ctx.Handlers[handlerName]; ok {
				in.OnTextChanged = func(text string) {
					handler(EventContext{
						Name:   "change",
						Target: in,
						Data:   text,
					})
				}
			}
		}
		if handlerName, ok := ctx.Spec.Events["submit"]; ok {
			if handler, ok := ctx.Handlers[handlerName]; ok {
				in.OnSubmit = func(text string) {
					handler(EventContext{
						Name:   "submit",
						Target: in,
						Data:   text,
					})
				}
			}
		}
		return in
	})

	// Checkbox — 复选框
	RegisterComponent("Checkbox", func(ctx DeclarativeContext) Widget {
		cb := DefaultCheckbox()
		cb.Label = ctx.Spec.Text
		// 应用 props
		if v, ok := ctx.Spec.Props["checked"]; ok {
			cb.Checked = parseBool(v, false)
		}
		if v, ok := ctx.Spec.Props["activeColor"]; ok {
			if c := parseColor(fmt.Sprintf("%v", v)); c != nil {
				cb.ActiveColor = *c
			}
		}
		if v, ok := ctx.Spec.Props["labelColor"]; ok {
			if c := parseColor(fmt.Sprintf("%v", v)); c != nil {
				cb.LabelColor = *c
			}
		}
		// 绑定事件
		if handlerName, ok := ctx.Spec.Events["change"]; ok {
			if handler, ok := ctx.Handlers[handlerName]; ok {
				cb.OnChanged = func(checked bool) {
					handler(EventContext{
						Name:   "change",
						Target: cb,
						Data:   checked,
					})
				}
			}
		}
		return cb
	})

	// Switch — 开关
	RegisterComponent("Switch", func(ctx DeclarativeContext) Widget {
		sw := DefaultSwitch()
		sw.Label = ctx.Spec.Text
		if v, ok := ctx.Spec.Props["value"]; ok {
			sw.Value = parseBool(v, false)
		}
		if v, ok := ctx.Spec.Props["activeColor"]; ok {
			if c := parseColor(fmt.Sprintf("%v", v)); c != nil {
				sw.ActiveColor = *c
			}
		}
		if v, ok := ctx.Spec.Props["inactiveColor"]; ok {
			if c := parseColor(fmt.Sprintf("%v", v)); c != nil {
				sw.InactiveColor = *c
			}
		}
		if v, ok := ctx.Spec.Props["disabled"]; ok {
			sw.Disabled = parseBool(v, false)
		}
		if v, ok := ctx.Spec.Props["size"]; ok {
			sw.Size = fmt.Sprintf("%v", v)
		}
		if v, ok := ctx.Spec.Props["activeText"]; ok {
			sw.ActiveText = fmt.Sprintf("%v", v)
		}
		if v, ok := ctx.Spec.Props["inactiveText"]; ok {
			sw.InactiveText = fmt.Sprintf("%v", v)
		}
		if handlerName, ok := ctx.Spec.Events["change"]; ok {
			if handler, ok := ctx.Handlers[handlerName]; ok {
				sw.OnChanged = func(value bool) {
					handler(EventContext{
						Name:   "change",
						Target: sw,
						Data:   value,
					})
				}
			}
		}
		return sw
	})

	// Rate — 评分
	RegisterComponent("Rate", func(ctx DeclarativeContext) Widget {
		r := DefaultRate()
		if v, ok := ctx.Spec.Props["value"]; ok {
			r.Value = parseFloat(v, 0)
		}
		if v, ok := ctx.Spec.Props["max"]; ok {
			r.Max = int(parseFloat(v, 5))
		}
		if v, ok := ctx.Spec.Props["disabled"]; ok {
			r.Disabled = parseBool(v, false)
		}
		if v, ok := ctx.Spec.Props["allowHalf"]; ok {
			r.AllowHalf = parseBool(v, false)
		}
		if v, ok := ctx.Spec.Props["size"]; ok {
			r.Size = fmt.Sprintf("%v", v)
		}
		if v, ok := ctx.Spec.Props["activeColor"]; ok {
			if c := parseColor(fmt.Sprintf("%v", v)); c != nil {
				r.ActiveColor = *c
			}
		}
		if handlerName, ok := ctx.Spec.Events["change"]; ok {
			if handler, ok := ctx.Handlers[handlerName]; ok {
				r.OnChanged = func(value float64) {
					handler(EventContext{Name: "change", Target: r, Data: value})
				}
			}
		}
		return r
	})

	// InputNumber — 数字输入框
	RegisterComponent("InputNumber", func(ctx DeclarativeContext) Widget {
		n := DefaultInputNumber()
		if v, ok := ctx.Spec.Props["value"]; ok {
			n.Value = parseFloat(v, 0)
		}
		if v, ok := ctx.Spec.Props["min"]; ok {
			n.Min = parseFloat(v, n.Min)
		}
		if v, ok := ctx.Spec.Props["max"]; ok {
			n.Max = parseFloat(v, n.Max)
		}
		if v, ok := ctx.Spec.Props["step"]; ok {
			n.Step = parseFloat(v, 1)
		}
		if v, ok := ctx.Spec.Props["precision"]; ok {
			n.Precision = int(parseFloat(v, 0))
		}
		if v, ok := ctx.Spec.Props["disabled"]; ok {
			n.Disabled = parseBool(v, false)
		}
		if v, ok := ctx.Spec.Props["size"]; ok {
			n.Size = fmt.Sprintf("%v", v)
		}
		if handlerName, ok := ctx.Spec.Events["change"]; ok {
			if handler, ok := ctx.Handlers[handlerName]; ok {
				n.OnChanged = func(value float64) {
					handler(EventContext{Name: "change", Target: n, Data: value})
				}
			}
		}
		return n
	})

	// Select — 下拉选择器
	RegisterComponent("Select", func(ctx DeclarativeContext) Widget {
		var opts []SelectOption
		if v, ok := ctx.Spec.Props["options"]; ok {
			if arr, ok := v.([]interface{}); ok {
				for _, item := range arr {
					if m, ok := item.(map[string]interface{}); ok {
						opt := SelectOption{
							Label: fmt.Sprintf("%v", m["label"]),
							Value: fmt.Sprintf("%v", m["value"]),
						}
						if d, ok := m["disabled"].(bool); ok {
							opt.Disabled = d
						}
						opts = append(opts, opt)
					}
				}
			}
		}
		s := NewSelect(opts)
		if v, ok := ctx.Spec.Props["value"]; ok {
			s.Value = fmt.Sprintf("%v", v)
		}
		if v, ok := ctx.Spec.Props["placeholder"]; ok {
			s.Placeholder = fmt.Sprintf("%v", v)
		}
		if v, ok := ctx.Spec.Props["disabled"]; ok {
			s.Disabled = parseBool(v, false)
		}
		if v, ok := ctx.Spec.Props["width"]; ok {
			s.Width = parseFloat(v, 220)
		}
		if v, ok := ctx.Spec.Props["size"]; ok {
			s.Size = fmt.Sprintf("%v", v)
		}
		if handlerName, ok := ctx.Spec.Events["change"]; ok {
			if handler, ok := ctx.Handlers[handlerName]; ok {
				s.OnChanged = func(value string) {
					handler(EventContext{Name: "change", Target: s, Data: value})
				}
			}
		}
		return s
	})

	// RadioButton — 单选按钮
	RegisterComponent("RadioButton", func(ctx DeclarativeContext) Widget {
		rb := DefaultRadioButton()
		rb.Label = ctx.Spec.Text
		if v, ok := ctx.Spec.Props["selected"]; ok {
			rb.Selected = parseBool(v, false)
		}
		if v, ok := ctx.Spec.Props["groupName"]; ok {
			rb.GroupName = fmt.Sprintf("%v", v)
		}
		if v, ok := ctx.Spec.Props["activeColor"]; ok {
			if c := parseColor(fmt.Sprintf("%v", v)); c != nil {
				rb.ActiveColor = *c
			}
		}
		if handlerName, ok := ctx.Spec.Events["change"]; ok {
			if handler, ok := ctx.Handlers[handlerName]; ok {
				rb.OnChanged = func() {
					handler(EventContext{
						Name:   "change",
						Target: rb,
					})
				}
			}
		}
		return rb
	})

	// Slider — 滑块选择器
	RegisterComponent("Slider", func(ctx DeclarativeContext) Widget {
		sl := DefaultSlider()
		sl.Min = parseFloat(ctx.Spec.Props["min"], sl.Min)
		sl.Max = parseFloat(ctx.Spec.Props["max"], sl.Max)
		sl.Step = parseFloat(ctx.Spec.Props["step"], sl.Step)
		sl.Value = parseFloat(ctx.Spec.Props["value"], sl.Value)
		if v, ok := ctx.Spec.Props["showLabel"]; ok {
			sl.ShowLabel = parseBool(v, false)
		}
		if handlerName, ok := ctx.Spec.Events["change"]; ok {
			if handler, ok := ctx.Handlers[handlerName]; ok {
				sl.OnChanged = func(value float64) {
					handler(EventContext{
						Name:   "change",
						Target: sl,
						Data:   value,
					})
				}
			}
		}
		return sl
	})

	// ProgressBar — 进度条
	RegisterComponent("ProgressBar", func(ctx DeclarativeContext) Widget {
		pb := DefaultProgressBar()
		pb.Value = parseFloat(ctx.Spec.Props["value"], pb.Value)
		if v, ok := ctx.Spec.Props["indeterminate"]; ok {
			pb.Indeterminate = parseBool(v, false)
		}
		if v, ok := ctx.Spec.Props["showLabel"]; ok {
			pb.ShowLabel = parseBool(v, false)
		}
		if v, ok := ctx.Spec.Props["height"]; ok {
			pb.Height = parseFloat(v, pb.Height)
		}
		if v, ok := ctx.Spec.Props["color"]; ok {
			if c := parseColor(fmt.Sprintf("%v", v)); c != nil {
				pb.Color = *c
			}
		}
		return pb
	})

	// Image — 图片
	RegisterComponent("Image", func(ctx DeclarativeContext) Widget {
		img := &Image{
			Width:  parseFloat(ctx.Spec.Props["width"], 0),
			Height: parseFloat(ctx.Spec.Props["height"], 0),
		}
		if v, ok := ctx.Spec.Props["fit"]; ok {
			switch fmt.Sprintf("%v", v) {
			case "contain":
				img.Fit = ImageFitContain
			case "cover":
				img.Fit = ImageFitCover
			default:
				img.Fit = ImageFitFill
			}
		}
		return img
	})

	// Tag — 标签（1:1 复刻 el-tag）
	RegisterComponent("Tag", func(ctx DeclarativeContext) Widget {
		tg := NewTag(ctx.Spec.Text)
		if v, ok := ctx.Spec.Props["type"]; ok {
			switch fmt.Sprintf("%v", v) {
			case "primary":
				tg.Type = TagPrimary
			case "success":
				tg.Type = TagSuccess
			case "info":
				tg.Type = TagInfo
			case "warning":
				tg.Type = TagWarning
			case "danger":
				tg.Type = TagDanger
			}
		}
		if v, ok := ctx.Spec.Props["effect"]; ok {
			switch fmt.Sprintf("%v", v) {
			case "light":
				tg.Effect = TagLight
			case "dark":
				tg.Effect = TagDark
			case "plain":
				tg.Effect = TagPlain
			}
		}
		if v, ok := ctx.Spec.Props["size"]; ok {
			switch fmt.Sprintf("%v", v) {
			case "large":
				tg.Size = TagSizeLarge
			case "small":
				tg.Size = TagSizeSmall
			default:
				tg.Size = TagSizeDefault
			}
		}
		if v, ok := ctx.Spec.Props["closable"]; ok {
			tg.Closable = parseBool(v, false)
		}
		if v, ok := ctx.Spec.Props["round"]; ok {
			tg.Round = parseBool(v, false)
		}
		if v, ok := ctx.Spec.Props["hit"]; ok {
			tg.Hit = parseBool(v, false)
		}
		if v, ok := ctx.Spec.Props["color"]; ok {
			if c := parseColor(fmt.Sprintf("%v", v)); c != nil {
				tg.Color = *c
			}
		}
		if handlerName, ok := ctx.Spec.Events["click"]; ok {
			if handler, ok := ctx.Handlers[handlerName]; ok {
				tg.OnClick = func() {
					handler(EventContext{Name: "click", Target: tg})
				}
			}
		}
		if handlerName, ok := ctx.Spec.Events["close"]; ok {
			if handler, ok := ctx.Handlers[handlerName]; ok {
				tg.OnClose = func() {
					handler(EventContext{Name: "close", Target: tg})
				}
			}
		}
		return tg
	})

	// Tooltip — 文字提示
	RegisterComponent("Tooltip", func(ctx DeclarativeContext) Widget {
		tt := &Tooltip{Placement: PlacementBottom, Effect: TooltipDark, Trigger: TooltipHover, Content: ctx.Spec.Text}
		if v, ok := ctx.Spec.Props["content"]; ok {
			tt.Content = fmt.Sprintf("%v", v)
		}
		if v, ok := ctx.Spec.Props["placement"]; ok {
			tt.Placement = parsePlacement(fmt.Sprintf("%v", v))
		}
		if v, ok := ctx.Spec.Props["effect"]; ok {
			if fmt.Sprintf("%v", v) == "light" {
				tt.Effect = TooltipLight
			}
		}
		if v, ok := ctx.Spec.Props["trigger"]; ok {
			if fmt.Sprintf("%v", v) == "click" {
				tt.Trigger = TooltipClick
			}
		}
		if v, ok := ctx.Spec.Props["disabled"]; ok {
			tt.Disabled = parseBool(v, false)
		}
		if len(ctx.Children) > 0 {
			tt.Child = ctx.Children[0]
		}
		return tt
	})

	// Popover — 弹出框
	RegisterComponent("Popover", func(ctx DeclarativeContext) Widget {
		pp := NewPopover(nil)
		if v, ok := ctx.Spec.Props["title"]; ok {
			pp.Title = fmt.Sprintf("%v", v)
		}
		pp.Content = ctx.Spec.Text
		if v, ok := ctx.Spec.Props["content"]; ok {
			pp.Content = fmt.Sprintf("%v", v)
		}
		if v, ok := ctx.Spec.Props["placement"]; ok {
			pp.Placement = parsePlacement(fmt.Sprintf("%v", v))
		}
		if v, ok := ctx.Spec.Props["trigger"]; ok {
			if fmt.Sprintf("%v", v) == "hover" {
				pp.Trigger = TooltipHover
			}
		}
		if v, ok := ctx.Spec.Props["width"]; ok {
			pp.Width = parseFloat(v, 200)
		}
		if len(ctx.Children) > 0 {
			pp.Child = ctx.Children[0]
			if len(ctx.Children) > 1 {
				pp.Body = ctx.Children[1]
			}
		}
		return pp
	})

	// Badge — 徽章
	RegisterComponent("Badge", func(ctx DeclarativeContext) Widget {
		b := &Badge{Type: BadgeDanger, Value: ctx.Spec.Text}
		if v, ok := ctx.Spec.Props["value"]; ok {
			b.Value = fmt.Sprintf("%v", v)
		}
		if v, ok := ctx.Spec.Props["max"]; ok {
			b.Max = int(parseFloat(v, 0))
		}
		if v, ok := ctx.Spec.Props["isDot"]; ok {
			b.IsDot = parseBool(v, false)
		}
		if v, ok := ctx.Spec.Props["type"]; ok {
			switch fmt.Sprintf("%v", v) {
			case "primary":
				b.Type = BadgePrimary
			case "success":
				b.Type = BadgeSuccess
			case "warning":
				b.Type = BadgeWarning
			case "info":
				b.Type = BadgeInfo
			}
		}
		if len(ctx.Children) > 0 {
			b.Child = ctx.Children[0]
		}
		return b
	})

	// Avatar — 头像
	RegisterComponent("Avatar", func(ctx DeclarativeContext) Widget {
		a := &Avatar{Text: ctx.Spec.Text, Shape: AvatarCircle}
		if v, ok := ctx.Spec.Props["shape"]; ok && fmt.Sprintf("%v", v) == "square" {
			a.Shape = AvatarSquare
		}
		if v, ok := ctx.Spec.Props["size"]; ok {
			a.Size = parseFloat(v, 0)
		}
		if v, ok := ctx.Spec.Props["icon"]; ok {
			a.Icon = fmt.Sprintf("%v", v)
		}
		if v, ok := ctx.Spec.Props["bgColor"]; ok {
			if c := parseColor(fmt.Sprintf("%v", v)); c != nil {
				a.BgColor = *c
			}
		}
		return a
	})

	// Link — 链接
	RegisterComponent("Link", func(ctx DeclarativeContext) Widget {
		l := &Link{Text: ctx.Spec.Text, Underline: true}
		if v, ok := ctx.Spec.Props["type"]; ok {
			switch fmt.Sprintf("%v", v) {
			case "primary":
				l.Type = LinkPrimary
			case "success":
				l.Type = LinkSuccess
			case "warning":
				l.Type = LinkWarning
			case "danger":
				l.Type = LinkDanger
			case "info":
				l.Type = LinkInfo
			}
		}
		if v, ok := ctx.Spec.Props["underline"]; ok {
			l.Underline = parseBool(v, true)
		}
		if v, ok := ctx.Spec.Props["disabled"]; ok {
			l.Disabled = parseBool(v, false)
		}
		if h, ok := ctx.Spec.Events["click"]; ok {
			if fn, ok := ctx.Handlers[h]; ok {
				l.OnClick = func() { fn(EventContext{Name: "click", Target: l}) }
			}
		}
		return l
	})

	// Alert — 警告
	RegisterComponent("Alert", func(ctx DeclarativeContext) Widget {
		a := &Alert{Title: ctx.Spec.Text, ShowIcon: true}
		if v, ok := ctx.Spec.Props["title"]; ok {
			a.Title = fmt.Sprintf("%v", v)
		}
		if v, ok := ctx.Spec.Props["description"]; ok {
			a.Description = fmt.Sprintf("%v", v)
		}
		if v, ok := ctx.Spec.Props["type"]; ok {
			switch fmt.Sprintf("%v", v) {
			case "success":
				a.Type = AlertSuccess
			case "warning":
				a.Type = AlertWarning
			case "error":
				a.Type = AlertError
			case "info":
				a.Type = AlertInfo
			}
		}
		if v, ok := ctx.Spec.Props["effect"]; ok && fmt.Sprintf("%v", v) == "dark" {
			a.Dark = true
		}
		if v, ok := ctx.Spec.Props["closable"]; ok {
			a.Closable = parseBool(v, false)
		}
		if v, ok := ctx.Spec.Props["showIcon"]; ok {
			a.ShowIcon = parseBool(v, true)
		}
		if h, ok := ctx.Spec.Events["close"]; ok {
			if fn, ok := ctx.Handlers[h]; ok {
				a.OnClose = func() { fn(EventContext{Name: "close", Target: a}) }
			}
		}
		return a
	})

	// Empty — 空状态
	RegisterComponent("Empty", func(ctx DeclarativeContext) Widget {
		em := &Empty{Description: ctx.Spec.Text, ImageSize: 80}
		if v, ok := ctx.Spec.Props["description"]; ok {
			em.Description = fmt.Sprintf("%v", v)
		}
		if v, ok := ctx.Spec.Props["imageSize"]; ok {
			em.ImageSize = parseFloat(v, 80)
		}
		em.Children = ctx.Children
		return em
	})

	// Divider — 分割线
	RegisterComponent("Divider", func(ctx DeclarativeContext) Widget {
		d := &Divider{}
		if ctx.Spec.Style != nil {
			if c := parseColor(ctx.Spec.Style.BackgroundColor); c != nil {
				d.Color = *c
			}
		}
		return d
	})
	RegisterComponent("Separator", func(ctx DeclarativeContext) Widget {
		return &Divider{}
	})

	// Container — 通用容器
	RegisterComponent("Container", func(ctx DeclarativeContext) Widget {
		c := &Container{
			Width:  parseFloat(ctx.Spec.Props["width"], 0),
			Height: parseFloat(ctx.Spec.Props["height"], 0),
		}
		if ctx.Spec.Style != nil {
			style := ctx.Spec.Style.ToStyle()
			c.Padding = style.Padding
			c.BorderRadius = style.BorderRadius
			if style.BackgroundColor != nil {
				c.Background = &PaintWidget{Color: style.BackgroundColor}
			}
			if style.BorderColor != nil {
				bw := style.BorderWidth
				if bw == 0 {
					bw = 1
				}
				border := paint.AllBorder(*style.BorderColor, bw)
				c.Border = &border
			}
		}
		if len(ctx.Children) > 0 {
			c.Child = ctx.Children[0]
		}
		return c
	})

	// Card — 卡片容器
	RegisterComponent("Card", func(ctx DeclarativeContext) Widget {
		card := &Card{}
		if ctx.Spec.Style != nil && ctx.Spec.Style.BorderRadius > 0 {
			card.Radius = ctx.Spec.Style.BorderRadius
		}
		if len(ctx.Children) > 0 {
			card.Child = ctx.Children[0]
		}
		return card
	})
}

// ─── 辅助构建函数 ────────────────────────────────

// resolveStyle 从 ComponentSpec 解析合并后的 Style
func resolveStyle(spec ComponentSpec) Style {
	style := Style{}
	if spec.Class != "" {
		style = Merge(style, Class(spec.Class))
	}
	if spec.Style != nil {
		style = Merge(style, spec.Style.ToStyle())
	}
	return style
}

// buildStyledContainer 创建带背景/边框/内边距的容器，内部嵌入子组件。
func buildStyledContainer(widgetType string, children []Widget, style Style) Widget {
	container := &Container{
		Padding:      style.Padding,
		BorderRadius: style.BorderRadius,
	}
	if style.BackgroundColor != nil {
		container.Background = &PaintWidget{Color: style.BackgroundColor}
	}
	if style.BorderColor != nil {
		bw := style.BorderWidth
		if bw == 0 {
			bw = 1
		}
		border := paint.AllBorder(*style.BorderColor, bw)
		container.Border = &border
	}
	if style.Width != 0 {
		container.Width = style.Width
	}
	if style.Height != 0 {
		container.Height = style.Height
	}

	dir := layout.FlexColumn
	if widgetType == "Row" || widgetType == "HBox" {
		dir = layout.FlexRow
	}
	// 单子也走 Flex（而非直挂 container.Child）：否则 alignItems/justifyContent 无处生效。
	container.Child = &Flex{
		MultiChildWidget:   MultiChildWidget{Children: children},
		Direction:          dir,
		MainAxisAlignment:  parseJustify(style.JustifyContent),
		CrossAxisAlignment: parseAlign(style.AlignItems),
		Gap:                style.Gap,
	}
	return container
}

// applyTextProps 应用文本组件专用属性
func applyTextProps(t *Text, props map[string]any) {
	if props == nil {
		return
	}
	if v, ok := props["fontSize"]; ok {
		t.Font.Size = parseFloat(v, t.Font.Size)
	}
	if v, ok := props["fontFamily"]; ok {
		t.Font.Family = fmt.Sprintf("%v", v)
	}
	if v, ok := props["fontWeight"]; ok {
		switch fmt.Sprintf("%v", v) {
		case "bold", "700", "800", "900":
			t.Font.Weight = canvas.FontWeightBold
		case "medium", "500", "600":
			t.Font.Weight = canvas.FontWeightMedium
		default:
			t.Font.Weight = canvas.FontWeightNormal
		}
	}
	if v, ok := props["color"]; ok {
		if c := parseColor(fmt.Sprintf("%v", v)); c != nil {
			t.Color = *c
		}
	}
	if v, ok := props["maxLines"]; ok {
		t.MaxLines = int(parseFloat(v, 0))
	}
}

// applyHeaderStyle 应用标题组件的样式覆盖
func applyHeaderStyle(t *Text, spec *StyleSpec) {
	if spec == nil {
		return
	}
	if c := parseColor(spec.TextColor); c != nil {
		t.Color = *c
	}
	if spec.FontSize > 0 {
		t.Font.Size = spec.FontSize
	}
}
