package widget

import (
	"github.com/hoonfeng/goui/pkg/canvas"
	"github.com/hoonfeng/goui/pkg/types"
)

// ──────────────────────────────────────────────
// HTML 风格语义组件 — 类似 <div>, <h1>, <p>, <span>
// ──────────────────────────────────────────────
//
// 这些组件让 GUI 构建更具声明式风格，类似 HTML/JSX：
//
//	widget.Div(
//	    widget.Class("card"),
//	    widget.H2("用户信息"),
//	    widget.Div(
//	        widget.Style{Padding: types.EdgeInsets(8)},
//	        widget.P("欢迎使用 goui"),
//	        widget.Button("确定"),
//	    ),
//	)
//
// Div 的变参同时接受 Style 和 Widget，样式与内容可以交织传入。

// ── Div ──────────────────────────────────────

// Div 创建块级容器（类似 HTML 的 <div>）。
// 默认使用 Flex 垂直布局，可选的 Style 参数控制盒模型和视觉效果。
//
// 参数可以是 Style 或 Widget 的混合：
//
//	// 仅有子组件
//	Div(text1, text2)
//
//	// 内联样式 + 子组件
//	Div(Style{Padding: types.EdgeInsets(8)}, text1)
//
//	// 使用命名样式类
//	Div(Class("card"), text1, text2)
//
//	// 无子组件：创建空白占位容器
//	Div()
func Div(args ...interface{}) Widget {
	style, children := extractStyleAndWidgets(args)

	// L3 绝对定位：position:absolute → Positioned 包裹；relative → Stack 定位上下文。
	switch style.Position {
	case "absolute":
		p := &Positioned{SingleChildWidget: SingleChildWidget{Child: divCore(style, children)}, ZIndex: style.ZIndex}
		if style.Top != nil {
			p.Top, p.HasTop = *style.Top, true
		}
		if style.Left != nil {
			p.Left, p.HasLeft = *style.Left, true
		}
		if style.Right != nil {
			p.Right, p.HasRight = *style.Right, true
		}
		if style.Bottom != nil {
			p.Bottom, p.HasBottom = *style.Bottom, true
		}
		return p
	case "relative":
		// relative Div：保留自身尺寸/背景(Container)，内部用 Stack 作绝对定位上下文。
		stack := NewStack(applyStyleToWidgets(children, style)...)
		if style.HasBoxModel() || style.HasVisual() {
			container := &Container{}
			style.applyToContainer(container)
			container.Child = stack
			return container
		}
		return stack
	}
	return divCore(style, children)
}

// divCore 构建普通 Div（Flex/Container），不含 L3 绝对定位。
func divCore(style Style, children []Widget) Widget {
	if len(children) == 0 && !style.HasBoxModel() && !style.HasVisual() {
		return &Container{}
	}

	if !style.needsContainer() {
		if len(children) == 0 {
			return &Container{}
		}
		styledChildren := applyStyleToWidgets(children, style)
		return &Flex{
			MultiChildWidget:   MultiChildWidget{Children: styledChildren},
			Direction:          flexDir(style.FlexDirection),
			MainAxisAlignment:  parseJustify(style.JustifyContent),
			CrossAxisAlignment: parseAlign(style.AlignItems),
			Gap:                style.Gap,
		}
	}

	// 有视觉效果：用 Container 包装。统一走 applyToContainer —— Div 因此自动支持
	// Style 的所有视觉/盒模型属性(含 Gradient/Shadow/Opacity/MaxWidth)，
	// 不再各自硬编码(消除与 applyToContainer 的重复，这正是渐变之前丢失的原因)。
	container := &Container{}
	style.applyToContainer(container)

	styledChildren := applyStyleToWidgets(children, style)

	if len(styledChildren) == 1 {
		container.Child = styledChildren[0]
	} else if len(styledChildren) > 1 {
		container.Child = &Flex{
			MultiChildWidget:   MultiChildWidget{Children: styledChildren},
			Direction:          flexDir(style.FlexDirection),
			MainAxisAlignment:  parseJustify(style.JustifyContent),
			CrossAxisAlignment: parseAlign(style.AlignItems),
			Gap:                style.Gap,
		}
	}

	return container
}

// ── Section (语义区块) ───────────────────────

// Section 创建语义化的区块容器（类似 HTML 的 <section>）。
// 本质是带有默认间距的 Div，用于内容分区。
func Section(args ...interface{}) Widget {
	style, children := extractStyleAndWidgets(args)
	if style.Padding.Left == 0 && style.Padding.Right == 0 &&
		style.Padding.Top == 0 && style.Padding.Bottom == 0 {
		style.Padding = types.EdgeInsets(16)
	}
	return Div(append([]interface{}{style}, widgetsToInterface(children)...)...)
}

func widgetsToInterface(widgets []Widget) []interface{} {
	result := make([]interface{}, len(widgets))
	for i, w := range widgets {
		result[i] = w
	}
	return result
}

// ── 标题组件 ─────────────────────────────────

// H1 创建一级标题（28px 粗体）
func H1(text string) *Text {
	return &Text{
		Text:  text,
		Color: elTextPrimary(),
		Font: canvas.Font{
			Family: "sans-serif",
			Size:   28,
			Weight: canvas.FontWeightBold,
		},
	}
}

// H2 创建二级标题（22px 粗体）
func H2(text string) *Text {
	return &Text{
		Text:  text,
		Color: elTextPrimary(),
		Font: canvas.Font{
			Family: "sans-serif",
			Size:   22,
			Weight: canvas.FontWeightBold,
		},
	}
}

// H3 创建三级标题（18px 粗体）
func H3(text string) *Text {
	return &Text{
		Text:  text,
		Color: elTextPrimary(),
		Font: canvas.Font{
			Family: "sans-serif",
			Size:   18,
			Weight: canvas.FontWeightBold,
		},
	}
}

// H4 创建四级标题（16px 粗体）
func H4(text string) *Text {
	return &Text{
		Text:  text,
		Color: elTextPrimary(),
		Font: canvas.Font{
			Family: "sans-serif",
			Size:   16,
			Weight: canvas.FontWeightBold,
		},
	}
}

// ── 段落与行内文本 ──────────────────────────

// P 创建段落文本（14px Medium 字重）
// 使用 Medium 而非 Normal 字重，避免阿里普惠体 Regular(55) 偏细的问题
func P(text string) *Text {
	return &Text{
		Text:  text,
		Color: types.ColorFromRGB(60, 60, 60),
		Font: canvas.Font{
			Family: "sans-serif",
			Size:   14,
			Weight: canvas.FontWeightMedium,
		},
	}
}

// Small 创建小号文本（12px Medium 字重）
// 使用 Medium 字重使小字也清晰可读
func Small(text string) *Text {
	return &Text{
		Text:  text,
		Color: types.ColorFromRGB(128, 128, 128),
		Font: canvas.Font{
			Family: "sans-serif",
			Size:   12,
			Weight: canvas.FontWeightMedium,
		},
	}
}

// Strong 创建加粗文本
func Strong(text string) *Text {
	return &Text{
		Text:  text,
		Color: elTextPrimary(),
		Font: canvas.Font{
			Family: "sans-serif",
			Size:   14,
			Weight: canvas.FontWeightBold,
		},
	}
}

// ── Spacer + Separator ──────────────────────

// SpacerDiv 创建弹性空间占位（与 CSS flex:1 类似）
//
//	HBox(button1, SpacerDiv(), button2)
//	// button1 在最左，button2 在最右
func SpacerDiv() *Spacer {
	return &Spacer{}
}

// Separator 创建水平分割线（类似 HTML <hr>）
//
//	Div(text1, Separator(), text2)
func Separator() *Divider {
	return &Divider{
		Direction: DividerHorizontal,
		Color:     types.ColorFromRGB(222, 226, 230),
		Thickness: 1,
	}
}

// ── 内部辅助 ─────────────────────────────────

// extractStyleAndWidgets 从混合变参中提取 Style 和 Widget。
// 允许多个 Style 参数（自动合并），Widget 保持原有顺序。
func extractStyleAndWidgets(args []interface{}) (Style, []Widget) {
	merged := Style{}
	widgets := []Widget{}

	for _, arg := range args {
		switch v := arg.(type) {
		case Style:
			merged = mergeStyle(merged, v)
		case *Style:
			if v != nil {
				merged = mergeStyle(merged, *v)
			}
		case Widget:
			widgets = append(widgets, v)
		case []Widget: // 便利：动态构造的子列表可整段传入，免去调用处自拼 []interface{}
			widgets = append(widgets, v...)
		}
	}

	return merged, widgets
}

// applyStyleToWidgets 对一组 Widget 应用排版样式。
func applyStyleToWidgets(widgets []Widget, style Style) []Widget {
	if !style.HasTypography() {
		return widgets
	}
	result := make([]Widget, len(widgets))
	for i, w := range widgets {
		result[i] = applyStyleToWidget(w, style)
	}
	return result
}
