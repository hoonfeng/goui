package widget

import (
	"fmt"
	"reflect"
	"strings"
	"time"

	"github.com/hoonfeng/goui/pkg/animation"
	"github.com/hoonfeng/goui/pkg/canvas"
	"github.com/hoonfeng/goui/pkg/event"
	"github.com/hoonfeng/goui/internal/i18n"
	"github.com/hoonfeng/goui/internal/layout"
	"github.com/hoonfeng/goui/pkg/paint"
	"github.com/hoonfeng/goui/pkg/types"
)

// Container 是一个灵活的容器控件，可以设置内边距、边框、背景色、圆角等。
type Container struct {
	SingleChildWidget
	Padding      types.Inset
	Margin       types.Inset
	Background   *PaintWidget
	Border       *paint.Border
	Width        float64
	Height       float64
	MaxWidth     float64 // 最大宽度（0=无限制）
	MaxHeight    float64 // 最大高度（0=无限制）
	Alignment    layout.Alignment
	BorderRadius float64         // 圆角半径（0=无圆角）
	Gradient     *paint.Gradient // 背景渐变（设置后覆盖 Background 纯色）
	Shadow       *paint.Shadow   // 盒阴影
	Opacity      *float64        // 不透明度 0~1（nil=完全不透明）
	ClipContent  bool            // 裁剪溢出内容（overflow:hidden）

	// CSS 伪类 —— 状态样式覆盖（nil=该状态无覆盖）。进入对应状态时，其中已设的
	// 背景/边框(颜色+宽度，可从无到有)/圆角/阴影/透明度覆盖基础值。优先级 hover < focus < active。
	Hover  *Style // :hover  鼠标悬停
	Focus  *Style // :focus  获得焦点（元素自身被点中获焦时）
	Active *Style // :active 鼠标按下
}

// PaintWidget 存储绘制样式
type PaintWidget struct {
	Color *types.Color
}

// CreateElement 创建 Container 的 Element
func (c *Container) CreateElement() Element {
	return &ContainerElement{
		BaseElement: BaseElement{widget: c},
		container:   c,
	}
}

// Update 更新 Widget 配置并同步内部 container 引用，
// 使 Build/Layout/Paint 使用最新配置（否则父级 SetState 无法传播到子树）。
func (e *ContainerElement) Update(newWidget Widget) {
	e.widget = newWidget
	e.dirty = true
	if c, ok := newWidget.(*Container); ok {
		e.container = c
	}
}

// ContainerElement Container 的运行时 Element
type ContainerElement struct {
	BaseElement
	container  *Container
	child      Element
	hovered    bool // :hover 状态
	pressed    bool // :active 状态（鼠标在其上按下）
	cssFocused bool // :focus 状态（元素自身获得焦点）
}

// interactive 是否设了任意 CSS 伪类状态样式（决定是否响应 hover/press 事件）。
func (e *ContainerElement) interactive() bool {
	c := e.container
	return c.Hover != nil || c.Active != nil || c.Focus != nil
}

// HandleEvent 跟踪 :hover / :active 状态（仅当设了伪类样式时记录）。
func (e *ContainerElement) HandleEvent(ev event.Event) bool {
	if !e.interactive() {
		return false
	}
	switch ev.Type() {
	case event.TypeMouseEnter:
		if !e.hovered {
			e.hovered = true
			e.MarkNeedsPaint()
		}
	case event.TypeMouseLeave:
		// 移出时同时取消 hover 与 press（按下后拖出不应保持按压态）
		if e.hovered || e.pressed {
			e.hovered, e.pressed = false, false
			e.MarkNeedsPaint()
		}
	case event.TypeMouseDown:
		if e.container.Active != nil && !e.pressed {
			e.pressed = true
			e.MarkNeedsPaint()
		}
	case event.TypeMouseUp:
		if e.pressed {
			e.pressed = false
			e.MarkNeedsPaint()
		}
	}
	return false // 不阻断冒泡：容器只观察状态，点击仍应传给子/父
}

// Focus / Blur —— :focus 伪类。元素自身被 App.RequestFocus 选中时点亮 focus 样式，
// 失焦时熄灭。不重写 IsFocused（保持 false），避免触发光标闪烁式的持续重绘。
func (e *ContainerElement) Focus() {
	if e.container.Focus != nil && !e.cssFocused {
		e.cssFocused = true
		e.MarkNeedsPaint()
	}
}

func (e *ContainerElement) Blur() {
	if e.cssFocused {
		e.cssFocused = false
		e.MarkNeedsPaint()
	}
}

// stateStyles 返回当前激活的状态样式，按优先级低→高排列（后者覆盖前者）：
// :hover < :focus < :active。eff* 系列从尾向头取首个已设属性即得最高优先级值。
func (e *ContainerElement) stateStyles() []*Style {
	c := e.container
	var ss []*Style
	if e.hovered && c.Hover != nil {
		ss = append(ss, c.Hover)
	}
	if e.cssFocused && c.Focus != nil {
		ss = append(ss, c.Focus)
	}
	if e.pressed && c.Active != nil {
		ss = append(ss, c.Active)
	}
	return ss
}

// effBg 生效背景色（状态样式优先，否则基础背景）。
func (e *ContainerElement) effBg() *types.Color {
	ss := e.stateStyles()
	for i := len(ss) - 1; i >= 0; i-- {
		if ss[i].BackgroundColor != nil {
			return ss[i].BackgroundColor
		}
	}
	if e.container.Background != nil {
		return e.container.Background.Color
	}
	return nil
}

// effOpacity 生效不透明度。
func (e *ContainerElement) effOpacity() *float64 {
	ss := e.stateStyles()
	for i := len(ss) - 1; i >= 0; i-- {
		if ss[i].Opacity != nil {
			return ss[i].Opacity
		}
	}
	return e.container.Opacity
}

// effRadius 生效圆角。
func (e *ContainerElement) effRadius() float64 {
	ss := e.stateStyles()
	for i := len(ss) - 1; i >= 0; i-- {
		if ss[i].BorderRadius != 0 {
			return ss[i].BorderRadius
		}
	}
	return e.container.BorderRadius
}

// effShadow 生效阴影。
func (e *ContainerElement) effShadow() *paint.Shadow {
	ss := e.stateStyles()
	for i := len(ss) - 1; i >= 0; i-- {
		if ss[i].Shadow != nil {
			return ss[i].Shadow
		}
	}
	return e.container.Shadow
}

// effBorder 生效边框：状态样式可覆盖颜色/宽度，甚至让无边框容器在 hover/focus 时长出边框
//（focus ring 常见诉求）。返回 nil 表示不画。
func (e *ContainerElement) effBorder() *paint.Border {
	ss := e.stateStyles()
	var ovColor *types.Color
	var ovWidth float64
	for i := len(ss) - 1; i >= 0; i-- {
		if ovColor == nil && ss[i].BorderColor != nil {
			ovColor = ss[i].BorderColor
		}
		if ovWidth == 0 && ss[i].BorderWidth != 0 {
			ovWidth = ss[i].BorderWidth
		}
	}
	if ovColor == nil && ovWidth == 0 {
		return e.container.Border // 无状态覆盖：用基础边框（允许四边不同）
	}
	base := paint.Border{}
	if e.container.Border != nil {
		base = *e.container.Border
	}
	col := base.Top.Color
	w := base.Top.Width
	if ovColor != nil {
		col = *ovColor
	}
	if ovWidth != 0 {
		w = ovWidth
	}
	if w == 0 {
		w = 1
	}
	nb := paint.AllBorder(col, w)
	return &nb
}

// Build 构建子 Element
// 复用已有的子 Element（当 Widget 类型匹配时），避免丢失运行时状态。
func (e *ContainerElement) Build() []Element {
	if e.container.Child != nil {
		// 尝试复用已有子 Element
		if e.child != nil && reflect.TypeOf(e.child.Widget()) == reflect.TypeOf(e.container.Child) {
			e.child.Update(e.container.Child)
		} else {
			// 类型不匹配或第一次构建：卸载旧的创建新的
			if e.child != nil {
				e.child.Unmount()
			}
			e.child = CreateElementFor(e.container.Child)
			e.child.Mount(e, 0)
		}
		e.children = []Element{e.child}
		return e.children
	}
	// 没有子控件
	if e.child != nil {
		e.child.Unmount()
		e.child = nil
	}
	e.children = nil
	return nil
}

// Layout 执行布局
func (e *ContainerElement) Layout(ctx *layout.LayoutContext) layout.LayoutResult {
	c := e.container
	var maxWidth, maxHeight float64

	// 如果有固定尺寸，使用固定尺寸
	if c.Width > 0 {
		maxWidth = c.Width
	} else {
		maxWidth = ctx.Constraints.MaxWidth
	}
	if c.Height > 0 {
		maxHeight = c.Height
	} else {
		maxHeight = ctx.Constraints.MaxHeight
	}
	// 应用 max-width / max-height 上限
	if c.MaxWidth > 0 && maxWidth > c.MaxWidth {
		maxWidth = c.MaxWidth
	}
	if c.MaxHeight > 0 && maxHeight > c.MaxHeight {
		maxHeight = c.MaxHeight
	}

	// 减去内边距后传递给子控件
	childConstraints := ctx.Constraints
	childConstraints.MaxWidth = maxWidth - c.Padding.Left - c.Padding.Right
	childConstraints.MaxHeight = maxHeight - c.Padding.Top - c.Padding.Bottom
	childConstraints.MinWidth = 0
	childConstraints.MinHeight = 0
	// 子是 Flex（列/行布局，自带 justify-content 主轴对齐）→ 把父强制的高/宽下传，让 Flex 自己撑满并按
	// 其对齐摆放（默认顶/左），不要在容器层把整个 Flex 当「单一内容」垂直/水平居中——否则定高容器(如 Tabs
	// 强制 MinHeight 撑满面板)里的列布局会被整体居中/挤偏。非 Flex 子(文字/水印等)仍按原居中逻辑。
	if _, ok := e.child.(*FlexElement); ok {
		if m := ctx.Constraints.MinHeight - c.Padding.Top - c.Padding.Bottom; m > childConstraints.MinHeight {
			childConstraints.MinHeight = m
		}
		if m := ctx.Constraints.MinWidth - c.Padding.Left - c.Padding.Right; m > childConstraints.MinWidth {
			childConstraints.MinWidth = m
		}
	}

	if e.child != nil {
		result := e.child.Layout(&layout.LayoutContext{Constraints: childConstraints})
		childW := result.Size.Width
		childH := result.Size.Height
		w := childW + c.Padding.Left + c.Padding.Right
		h := childH + c.Padding.Top + c.Padding.Bottom
		if c.Width > 0 {
			w = c.Width // 固定宽度生效（之前被忽略）
		}
		if c.Height > 0 {
			h = c.Height // 固定高度生效（之前被忽略，导致 box 矮、文字偏上、水印铺不满）
		}
		// 被父强制更高/更宽时（AlignItems:stretch 或定高容器经 MinH/MinW 撑大）按强制尺寸，
		// 否则容器虽被撑高、却仍按内容算高 → 居中失效、子贴顶下方留空（如编辑器标签内容贴顶）。
		if h < ctx.Constraints.MinHeight {
			h = ctx.Constraints.MinHeight
		}
		if w < ctx.Constraints.MinWidth {
			w = ctx.Constraints.MinWidth
		}
		e.size = types.Size{Width: w, Height: h}
		// 子在可用区内垂直居中（容器高大于内容时），水平保持左对齐
		cy := c.Padding.Top
		if availH := h - c.Padding.Top - c.Padding.Bottom; availH > childH {
			cy = c.Padding.Top + (availH-childH)/2
		}
		e.child.SetPosition(types.Point{X: c.Padding.Left, Y: cy})
	} else {
		// 没有子控件：尺寸由内边距和显式尺寸决定，不要消耗无界空间
		w := ctx.Constraints.MinWidth + c.Padding.Left + c.Padding.Right
		h := ctx.Constraints.MinHeight + c.Padding.Top + c.Padding.Bottom
		if c.Width > 0 {
			w = c.Width
		} else if w <= 0 {
			// 无界宽度不膨胀（如在 Row 中接收 Unbounded 约束时）
			if maxWidth >= float64(1<<30) {
				w = ctx.Constraints.MinWidth // 用 MinWidth（通常为 0）而不是 INF
			} else {
				w = maxWidth // 有界上下文中保持填充行为
			}
		}
		if c.Height > 0 {
			h = c.Height
		} else if h <= 0 {
			// 无界高度不膨胀（如在 Column/ScrollView 中）
			if maxHeight >= float64(1<<30) {
				h = ctx.Constraints.MinHeight // 用 MinHeight（通常为 0）而不是 INF
			} else {
				h = maxHeight // 有界上下文中保持填充行为
			}
		}
		e.size = types.Size{Width: w, Height: h}
	}

	// 约束最终尺寸
	e.size = ctx.Constraints.Constrain(e.size)
	if c.MaxWidth > 0 && e.size.Width > c.MaxWidth {
		e.size.Width = c.MaxWidth
	}
	if c.MaxHeight > 0 && e.size.Height > c.MaxHeight {
		e.size.Height = c.MaxHeight
	}

	layout.LogLayoutDetailed("Container", fmt.Sprintf("%T", e.Widget()),
		ctx.Constraints, e.size,
		fmt.Sprintf("padding=(%.0f,%.0f,%.0f,%.0f) hasChild=%v",
			c.Padding.Left, c.Padding.Top, c.Padding.Right, c.Padding.Bottom,
			e.child != nil))
	return layout.LayoutResult{Size: e.size}
}

// Paint 绘制 Container（支持圆角背景、圆角边框和子内容裁剪）
func (e *ContainerElement) Paint(cvs canvas.Canvas, offset types.Point) {
	c := e.container
	pos := e.Offset()
	br := e.effRadius() // :hover/:focus/:active 可覆盖圆角

	// 不透明度：用图层 alpha 包裹整个容器（背景+边框+子内容）
	if op := e.effOpacity(); op != nil && *op < 1 {
		cvs.SaveLayerAlpha(*op)
		defer cvs.Restore()
	}

	// 盒阴影：多层向外扩散的半透明圆角块叠加，模拟柔和模糊。
	// （旧实现是单个偏移实心块，会在 Offset.Y 方向露出一坨硬边深色——尤其在深色遮罩上很突兀。）
	if sh := e.effShadow(); sh != nil {
		blur := sh.Blur
		if blur <= 0 {
			blur = 8
		}
		cx := pos.X + sh.Offset.X
		cy := pos.Y + sh.Offset.Y
		const shadowLayers = 6
		la := uint8(float64(sh.Color.A) / float64(shadowLayers))
		if la < 1 {
			la = 1
		}
		sp := paint.DefaultPaint()
		sp.Color = types.ColorFromRGBA(sh.Color.R, sh.Color.G, sh.Color.B, la)
		for i := shadowLayers; i >= 1; i-- {
			spread := blur * float64(i) / float64(shadowLayers)
			if br > 0 {
				cvs.DrawRoundedRect(cx-spread, cy-spread, e.size.Width+2*spread, e.size.Height+2*spread, br+spread, sp)
			} else {
				cvs.DrawRect(cx-spread, cy-spread, e.size.Width+2*spread, e.size.Height+2*spread, sp)
			}
		}
	}

	// 绘制背景：渐变优先，否则纯色（均支持圆角）
	if c.Gradient != nil {
		gp := paint.DefaultPaint()
		// Gradient 的 Start/End/Center 用 0~1 相对坐标(相对容器自身矩形，符合 CSS 习惯)，
		// 此处转为画布绝对坐标。
		g := *c.Gradient
		g.Start = types.Point{X: pos.X + g.Start.X*e.size.Width, Y: pos.Y + g.Start.Y*e.size.Height}
		g.End = types.Point{X: pos.X + g.End.X*e.size.Width, Y: pos.Y + g.End.Y*e.size.Height}
		if g.Type == paint.GradientRadial {
			g.Center = types.Point{X: pos.X + g.Center.X*e.size.Width, Y: pos.Y + g.Center.Y*e.size.Height}
			if g.Radius <= 1 {
				g.Radius = g.Radius * e.size.Width
			}
			gp.RadialGradient = &g
		} else {
			gp.LinearGradient = &g
		}
		if br > 0 {
			cvs.DrawRoundedRect(pos.X, pos.Y, e.size.Width, e.size.Height, br, gp)
		} else {
			cvs.DrawRect(pos.X, pos.Y, e.size.Width, e.size.Height, gp)
		}
	} else if bgC := e.effBg(); bgC != nil {
		rectPaint := paint.DefaultPaint()
		rectPaint.Color = *bgC
		if br > 0 {
			cvs.DrawRoundedRect(pos.X, pos.Y, e.size.Width, e.size.Height, br, rectPaint)
		} else {
			cvs.DrawRect(pos.X, pos.Y, e.size.Width, e.size.Height, rectPaint)
		}
	}

	// 绘制边框（支持圆角）。effBorder 已把 :hover/:focus/:active 的颜色/宽度覆盖合成进来。
	if bd := e.effBorder(); bd != nil {
		if br > 0 {
			// 圆角边框：使用 DrawRoundedRect 的描边模式绘制
			borderPaint := paint.DefaultStrokePaint()
			// 使用上边颜色和宽度作为代表（四边颜色可能不同，但圆角边框只能统一）
			borderPaint.Color = bd.Top.Color
			borderPaint.StrokeWidth = bd.Top.Width
			if bd.Top.Width > 0 {
				cvs.DrawRoundedRect(pos.X, pos.Y, e.size.Width, e.size.Height, br, borderPaint)
			}
		} else {
			// 非圆角：原来的四条线画法
			borderPaint := paint.DefaultStrokePaint()
			// 上边
			borderPaint.Color = bd.Top.Color
			borderPaint.StrokeWidth = bd.Top.Width
			if bd.Top.Width > 0 {
				cvs.DrawLine(pos.X, pos.Y, pos.X+e.size.Width, pos.Y, borderPaint)
			}
			// 下边
			borderPaint.Color = bd.Bottom.Color
			borderPaint.StrokeWidth = bd.Bottom.Width
			if bd.Bottom.Width > 0 {
				cvs.DrawLine(pos.X, pos.Y+e.size.Height, pos.X+e.size.Width, pos.Y+e.size.Height, borderPaint)
			}
			// 左边
			borderPaint.Color = bd.Left.Color
			borderPaint.StrokeWidth = bd.Left.Width
			if bd.Left.Width > 0 {
				cvs.DrawLine(pos.X, pos.Y, pos.X, pos.Y+e.size.Height, borderPaint)
			}
			// 右边
			borderPaint.Color = bd.Right.Color
			borderPaint.StrokeWidth = bd.Right.Width
			if bd.Right.Width > 0 {
				cvs.DrawLine(pos.X+e.size.Width, pos.Y, pos.X+e.size.Width, pos.Y+e.size.Height, borderPaint)
			}
		}
	}

	// 子内容裁剪：圆角容器或 overflow:hidden
	if br > 0 || c.ClipContent {
		cvs.Save()
		cvs.ClipRect(pos.X, pos.Y, e.size.Width, e.size.Height)
		defer cvs.Restore()
	}

	// 绘制子控件（偏移内边距）
	if e.child != nil {
		childOffset := types.Point{
			X: pos.X + c.Padding.Left,
			Y: pos.Y + c.Padding.Top,
		}
		e.child.Paint(cvs, childOffset)
	}
}

// Text 文本控件
type Text struct {
	StatelessWidget
	Text       string
	Font       canvas.Font
	Color      types.Color
	Align      canvas.TextAlign
	MaxLines   int
	LineHeight float64 // 自定义行高（0=用字体默认行高）
	Selectable bool    // 是否支持鼠标选中文本（选中后可 Ctrl+C 复制）
}

// TextElement 文本 Element
type TextElement struct {
	BaseElement
	text  *Text
	lines []string // 换行后的文本行缓存

	// 文本选中（Selectable=true 时启用）
	selStart int  // 选中起始字符索引（在整个文本中的 rune 偏移）
	selEnd   int  // 选中结束字符索引
	selecting bool // 鼠标正在拖动选择中

	// 测量缓存：文本/字体不变则复用换行与宽度，免每帧 layout+paint 反复做 O(N) 次 Skia CGO
	// 测量（自绘 UI 拖动/动画时 layout 反复跑，这是卡顿的真正根因）。多 maxWidth 槽：Flex 一帧
	// 会以不同约束多次 Layout（measure/final），单槽会互相抖动失效。
	cvText    string
	cvFont    canvas.Font
	cvEntries []tmEntry
	cvWidth   float64 // 最近一次 visibleLines 的整段文本宽度（Layout 定宽用）
	lyMaxW    float64 // Layout 用的换行宽度，Paint 复用它→命中同槽
}

// tmEntry 一个换行宽度下的测量缓存槽。
type tmEntry struct {
	maxW  float64
	maxLn int
	lines []string
	width float64
}

// CreateElement 创建 TextElement
func (t *Text) CreateElement() Element {
	return &TextElement{
		BaseElement: BaseElement{widget: t},
		text:        t,
	}
}

// Update 更新 Widget 配置并同步内部 text 引用（否则文本更新不会反映到渲染）。
func (e *TextElement) Update(newWidget Widget) {
	e.widget = newWidget
	e.dirty = true
	if t, ok := newWidget.(*Text); ok {
		e.text = t
		e.lines = nil // 失效换行缓存，下次 Layout 按新文本重算
		e.selStart = 0
		e.selEnd = 0
		e.selecting = false
	}
}

// splitLines 将文本按可用宽度分割成多行
func (e *TextElement) splitLines(maxWidth float64) []string {
	if e.text.Text == "" || maxWidth <= 0 {
		return []string{e.text.Text}
	}

	// 先用换行符分割段落
	var result []string
	paragraphs := splitByNewline(e.text.Text)

	for _, para := range paragraphs {
		if para == "" {
			result = append(result, "")
			continue
		}

		// 对每个段落进行单词换行
		runes := []rune(para)
		start := 0
		for start < len(runes) {
			lastBreak := -1 // 最后一个可换行位置（空格/制表符后）

			for i := start; i < len(runes); i++ {
				// 测量从 start 到 i 的宽度
				partialText := string(runes[start : i+1])
				metrics := canvas.MeasureTextGlobal(partialText, e.text.Font)
				w := metrics.Width

				// 记录空格/制表符位置作为可换行点
				if runes[i] == ' ' || runes[i] == '\t' {
					lastBreak = i + 1
				}

				// 判断是否超过可用宽度
				if w > maxWidth && i > start {
					if lastBreak > start {
						// 在最后一个空格处换行（保留空格在行尾）
						result = append(result, string(runes[start:lastBreak]))
						start = lastBreak
					} else {
						// 无空格可换，按字符硬换行
						result = append(result, string(runes[start:i]))
						start = i
					}
					break
				}

				// 到达段落末尾
				if i == len(runes)-1 {
					result = append(result, string(runes[start:]))
					start = len(runes)
				}
			}
		}
	}

	return result
}

// visibleLines 在 splitLines 基础上应用 MaxLines：超出则截到 MaxLines 行、末行加省略号。
// MaxLines<=0 表示不限制。Layout 与 Paint 都用它，保证测高与渲染一致。
func (e *TextElement) visibleLines(maxWidth float64) []string {
	// 文本/字体变了 → 整组槽失效。
	if e.cvText != e.text.Text || e.cvFont != e.text.Font {
		e.cvText, e.cvFont = e.text.Text, e.text.Font
		e.cvEntries = e.cvEntries[:0]
	}
	// 命中某宽度槽 → 直接复用，跳过 splitLines 的 O(N) Skia 测量。
	for i := range e.cvEntries {
		if e.cvEntries[i].maxW == maxWidth && e.cvEntries[i].maxLn == e.text.MaxLines {
			e.cvWidth = e.cvEntries[i].width
			return e.cvEntries[i].lines
		}
	}
	lines := e.splitLines(maxWidth)
	ml := e.text.MaxLines
	if ml > 0 && len(lines) > ml {
		out := make([]string, ml)
		copy(out, lines[:ml])
		out[ml-1] = ellipsizeToWidth(out[ml-1], e.text.Font, maxWidth) // 末行省略号
		lines = out
	}
	e.cvWidth = canvas.MeasureTextGlobal(e.text.Text, e.text.Font).Width
	if len(e.cvEntries) >= 6 { // 上限 6 槽（measure/final + 几个尺寸），超了丢最旧
		e.cvEntries = e.cvEntries[1:]
	}
	e.cvEntries = append(e.cvEntries, tmEntry{maxWidth, e.text.MaxLines, lines, e.cvWidth})
	return lines
}

// ellipsizeToWidth 给单行末尾加省略号，必要时削减字符以容纳 … 不超过 maxWidth。
func ellipsizeToWidth(s string, font canvas.Font, maxWidth float64) string {
	const ell = "…"
	if maxWidth <= 0 || canvas.MeasureTextGlobal(s+ell, font).Width <= maxWidth {
		return s + ell
	}
	runes := []rune(strings.TrimRight(s, " \t"))
	for len(runes) > 0 {
		runes = runes[:len(runes)-1]
		if canvas.MeasureTextGlobal(string(runes)+ell, font).Width <= maxWidth {
			return string(runes) + ell
		}
	}
	return ell
}

// splitByNewline 按换行符分割文本
func splitByNewline(text string) []string {
	var result []string
	start := 0
	for i, r := range text {
		if r == '\n' {
			result = append(result, text[start:i])
			start = i + 1
		}
	}
	result = append(result, text[start:])
	return result
}

// Layout 测量文本尺寸（支持多行，使用实际文本度量）
func (e *TextElement) Layout(ctx *layout.LayoutContext) layout.LayoutResult {
	t := e.text
	fontSize := t.Font.Size
	if fontSize <= 0 {
		fontSize = 16
	}

	maxWidth := ctx.Constraints.MaxWidth
	if maxWidth <= 0 || maxWidth >= float64(1<<30) {
		// 无界约束下使用默认宽度（如 Row 中），避免膨胀为 INF
		maxWidth = 800
	}

	lineHeight := canvas.GetFaceLineHeight(fontSize)
	if t.LineHeight > 0 {
		lineHeight = t.LineHeight
	}

	// 换行 + 测量（带缓存）：visibleLines 同时填充整段宽度缓存 cvWidth，Paint 复用 lyMaxW 命中同缓存。
	// 与 Paint 用完全相同的换行算法，保证测高与渲染行数一致。
	e.lyMaxW = maxWidth
	totalLines := len(e.visibleLines(maxWidth)) // 受 MaxLines 限制后的行数（测高与渲染一致）
	if totalLines < 1 {
		totalLines = 1
	}

	// 总高度 = 行数 * 行高
	height := float64(totalLines) * lineHeight
	// 宽度 = 最小可用宽度，但不超过测量宽度
	width := e.cvWidth
	if width > maxWidth {
		width = maxWidth
	}
	if width < ctx.Constraints.MinWidth {
		width = ctx.Constraints.MinWidth
	}

	e.size = ctx.Constraints.Constrain(types.Size{
		Width:  width,
		Height: height,
	})
	return layout.LayoutResult{Size: e.size}
}

// Paint 绘制文本（支持多行，使用准确的基线位置）
func (e *TextElement) Paint(cvs canvas.Canvas, offset types.Point) {
	t := e.text
	pos := e.Offset()
	fontSize := t.Font.Size
	if fontSize <= 0 {
		fontSize = 16
	}

	// 使用实际行高（从字体度量获取）
	lineHeight := canvas.GetFaceLineHeight(fontSize)
	if t.LineHeight > 0 {
		lineHeight = t.LineHeight
	}

	// 用 Layout 时的换行宽度（lyMaxW）→ 与 Layout 同 key、命中测量缓存，免 Paint 再做 O(N) 测量。
	maxWidth := e.lyMaxW
	if maxWidth <= 0 || maxWidth >= float64(1<<30) {
		maxWidth = e.size.Width
		if maxWidth <= 0 || maxWidth >= float64(1<<30) {
			maxWidth = 800
		}
	}

	// 计算换行（受 MaxLines 限制，末行省略号）
	lines := e.visibleLines(maxWidth)
	if len(lines) == 0 {
		return
	}

	p := paint.DefaultPaint()
	p.Color = t.Color

	// 解析对齐方式：RTL 语言下未显式左对齐的文本默认右对齐
	align := t.Align
	if i18n.IsRTL() && align == canvas.TextAlignLeft {
		align = canvas.TextAlignRight
	}

	// ── 选中高亮背景（Selectable=true 且有选中范围时）──
	if t.Selectable && e.selStart != e.selEnd {
		e.paintSelection(cvs, lines, pos, maxWidth, lineHeight, align)
	}

	// 逐行绘制：每行在自己的行盒(高 lineHeight)内**按实际墨迹**垂直居中。
	for i, line := range lines {
		y := canvas.BaselineForMiddle(cvs, line, t.Font, pos.Y+float64(i)*lineHeight, lineHeight)
		lineX := pos.X
		if align != canvas.TextAlignLeft {
			lw := cvs.MeasureText(line, t.Font).Width
			switch align {
			case canvas.TextAlignCenter:
				lineX = pos.X + (maxWidth-lw)/2
			case canvas.TextAlignRight:
				lineX = pos.X + maxWidth - lw
			}
			if lineX < pos.X {
				lineX = pos.X
			}
		}
		cvs.DrawText(line, lineX, y, t.Font, p)
	}
}

// paintSelection 绘制选中区域高亮背景。
func (e *TextElement) paintSelection(cvs canvas.Canvas, lines []string, pos types.Point, maxWidth, lineHeight float64, align canvas.TextAlign) {
	selStart, selEnd := e.selStart, e.selEnd
	if selStart > selEnd {
		selStart, selEnd = selEnd, selStart
	}
	font := e.text.Font
	// 选中高亮色（使用主题主色的半透明版本）
	selColor := CurrentTheme().PrimaryColor
	selColor.A = 60
	selPaint := paint.DefaultPaint()
	selPaint.Color = selColor

	runes := []rune(strings.Join(lines, "\n"))
	totalRunes := len(runes)
	if selStart < 0 {
		selStart = 0
	}
	if selEnd > totalRunes {
		selEnd = totalRunes
	}
	if selStart >= selEnd {
		return
	}

	// 构建每行的 rune 偏移范围 [startRune, endRune)
	lineRanges := make([]struct{ start, end int }, len(lines))
	runeOffset := 0
	for li, line := range lines {
		lineLen := len([]rune(line))
		lineRanges[li] = struct{ start, end int }{runeOffset, runeOffset + lineLen}
		runeOffset += lineLen + 1 // +1 为换行符
	}

	for li, line := range lines {
		lr := lineRanges[li]
		// 检查此行是否与选中范围重叠
		if lr.end <= selStart || lr.start >= selEnd {
			continue
		}
		// 计算此行中选中范围的列偏移
		colStart := 0
		if selStart > lr.start {
			colStart = selStart - lr.start
		}
		colEnd := len([]rune(line))
		if selEnd < lr.end {
			colEnd = selEnd - lr.start
		}
		if colStart >= colEnd {
			continue
		}
		// 计算选中矩形位置
		lineRunes := []rune(line)
		preText := string(lineRunes[:colStart])
		selText := string(lineRunes[colStart:colEnd])
		preW := cvs.MeasureText(preText, font).Width
		selW := cvs.MeasureText(selText, font).Width

		lineX := pos.X
		if align != canvas.TextAlignLeft {
			lw := cvs.MeasureText(line, font).Width
			switch align {
			case canvas.TextAlignCenter:
				lineX = pos.X + (maxWidth-lw)/2
			case canvas.TextAlignRight:
				lineX = pos.X + maxWidth - lw
			}
		}

		selX := lineX + preW
		selY := pos.Y + float64(li)*lineHeight
		cvs.DrawRect(selX, selY, selW, lineHeight, selPaint)
	}
}

// charIndexAtPos 将鼠标坐标转换为文本中的字符索引（rune 偏移）。
func (e *TextElement) charIndexAtPos(lines []string, mx, my, posX, posY, maxWidth, lineHeight float64, align canvas.TextAlign) int {
	font := e.text.Font
	li := int((my - posY) / lineHeight)
	if li < 0 || li >= len(lines) {
		return -1
	}
	line := lines[li]
	if line == "" {
		runes := []rune(strings.Join(lines, "\n"))
		offset := 0
		for i := 0; i < li; i++ {
			offset += len([]rune(lines[i])) + 1
		}
		if offset < len(runes) {
			return offset
		}
		return len(runes)
	}
	lineX := posX
	if align != canvas.TextAlignLeft {
		lw := canvas.MeasureTextGlobal(line, font).Width
		switch align {
		case canvas.TextAlignCenter:
			lineX = posX + (maxWidth-lw)/2
		case canvas.TextAlignRight:
			lineX = posX + maxWidth - lw
		}
	}
	lineRunes := []rune(line)
	bestIdx := len(lineRunes)
	for i := range lineRunes {
		preW := canvas.MeasureTextGlobal(string(lineRunes[:i+1]), font).Width
		charCenterX := lineX + preW
		if mx < charCenterX {
			bestIdx = i
			break
		}
	}
	runeOffset := 0
	for i := 0; i < li; i++ {
		runeOffset += len([]rune(lines[i])) + 1
	}
	return runeOffset + bestIdx
}

// isWordChar 判断 rune 是否为单词字符（字母/数字/下划线/CJK）。
func isWordChar(r rune) bool {
	switch {
	case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9', r == '_':
		return true
	case r >= 0x4E00 && r <= 0x9FFF, // CJK 统一表意文字
		r >= 0x3400 && r <= 0x4DBF, // CJK 扩展 A
		r >= 0xF900 && r <= 0xFAFF: // CJK 兼容表意文字
		return true
	default:
		return false
	}
}

// HandleEvent 处理鼠标事件（文本选中）。
func (e *TextElement) HandleEvent(ev event.Event) bool {
	if !e.text.Selectable {
		return false
	}
	switch ev.Type() {
	case event.TypeMouseDoubleClick:
		me, ok := ev.(*event.MouseEvent)
		if !ok || me.Button != event.ButtonLeft {
			return false
		}
		maxWidth := e.lyMaxW
		if maxWidth <= 0 {
			maxWidth = e.size.Width
		}
		lineHeight := canvas.GetFaceLineHeight(e.text.Font.Size)
		if e.text.LineHeight > 0 {
			lineHeight = e.text.LineHeight
		}
		pos := e.Offset()
		lines := e.visibleLines(maxWidth)
		idx := e.charIndexAtPos(lines, me.X, me.Y, pos.X, pos.Y, maxWidth, lineHeight, e.text.Align)
		if idx >= 0 {
			runes := []rune(e.text.Text)
			// 向左扩展至单词起始
			start := idx
			for start > 0 && isWordChar(runes[start-1]) {
				start--
			}
			// 向右扩展至单词结束
			end := idx
			for end < len(runes) && isWordChar(runes[end]) {
				end++
			}
			// 如果点击位置不是单词字符，则选中单个字符
			if start == end {
				if idx < len(runes) {
					end = idx + 1
				} else if idx > 0 {
					start = idx - 1
				}
			}
			e.selStart = start
			e.selEnd = end
			e.selecting = false // 双击完成，不进入拖选模式
			ev.StopPropagation()
			e.MarkNeedsPaint()
			return true
		}
	case event.TypeMouseDown:
		me, ok := ev.(*event.MouseEvent)
		if !ok || me.Button != event.ButtonLeft {
			return false
		}
		maxWidth := e.lyMaxW
		if maxWidth <= 0 {
			maxWidth = e.size.Width
		}
		lineHeight := canvas.GetFaceLineHeight(e.text.Font.Size)
		if e.text.LineHeight > 0 {
			lineHeight = e.text.LineHeight
		}
		pos := e.Offset()
		lines := e.visibleLines(maxWidth)
		idx := e.charIndexAtPos(lines, me.X, me.Y, pos.X, pos.Y, maxWidth, lineHeight, e.text.Align)
		if idx >= 0 {
			e.selStart = idx
			e.selEnd = idx
			e.selecting = true
			ev.StopPropagation()
			e.MarkNeedsPaint()
			return true
		}
	case event.TypeMouseMove:
		me, ok := ev.(*event.MouseEvent)
		if !ok || !e.selecting {
			return false
		}
		maxWidth := e.lyMaxW
		if maxWidth <= 0 {
			maxWidth = e.size.Width
		}
		lineHeight := canvas.GetFaceLineHeight(e.text.Font.Size)
		if e.text.LineHeight > 0 {
			lineHeight = e.text.LineHeight
		}
		pos := e.Offset()
		lines := e.visibleLines(maxWidth)
		idx := e.charIndexAtPos(lines, me.X, me.Y, pos.X, pos.Y, maxWidth, lineHeight, e.text.Align)
		if idx >= 0 {
			e.selEnd = idx
			ev.StopPropagation()
			e.MarkNeedsPaint()
			return true
		}
	case event.TypeMouseUp:
		if !e.selecting {
			return false
		}
		e.selecting = false
		ev.StopPropagation()
		if e.selStart != e.selEnd && ClipboardWrite != nil {
			runes := []rune(e.text.Text)
			start, end := e.selStart, e.selEnd
			if start > end {
				start, end = end, start
			}
			if start < 0 {
				start = 0
			}
			if end > len(runes) {
				end = len(runes)
			}
			if start < end {
				ClipboardWrite(string(runes[start:end]))
			}
		}
		e.MarkNeedsPaint()
		return true
	case event.TypeKeyDown:
		ke, ok := ev.(*event.KeyEvent)
		if !ok || ke.Mods&event.ModCtrl == 0 || ke.KeyCode != 0x43 {
			return false
		}
		// Ctrl+C: 复制选中的文本到剪贴板
		if e.selStart != e.selEnd && ClipboardWrite != nil {
			runes := []rune(e.text.Text)
			start, end := e.selStart, e.selEnd
			if start > end {
				start, end = end, start
			}
			if start < 0 {
				start = 0
			}
			if end > len(runes) {
				end = len(runes)
			}
			if start < end {
				ClipboardWrite(string(runes[start:end]))
				ev.StopPropagation()
			}
		}
		return true
	}
	return false
}

// ─── Web 风格便捷构造函数 ──────────────────────────

// NewText 创建文本控件。
// 用法: widget.NewText("Hello", theme.PrimaryColor)
func NewText(text string, color types.Color) *Text {
	t := &Text{
		Text:  text,
		Font:  canvas.DefaultFont(),
		Color: color,
	}
	if t.Color == (types.Color{}) {
		t.Color = CurrentTheme().TextColor
	}
	return t
}

// WithFont 设置文本字体（链式调用）
func (t *Text) WithFont(font canvas.Font) *Text {
	t.Font = font
	return t
}

// WithColor 设置文本颜色（链式调用）
func (t *Text) WithColor(color types.Color) *Text {
	t.Color = color
	return t
}

// WithAlign 设置文本对齐方式（链式调用）
func (t *Text) WithAlign(align canvas.TextAlign) *Text {
	t.Align = align
	return t
}

// NewButton 创建按钮控件。
// 用法: widget.NewButton("Click", fn)
func NewButton(text string, onClick func()) *Button {
	t := CurrentTheme()
	return &Button{
		Text:       text,
		OnClick:    onClick,
		Color:      t.Button.DefaultColor,
		HoverColor: t.Button.HoverColor,
		TextColor:  t.Button.TextColor,
		FontSize:   16,
		Padding:    types.EdgeInsetsLTRB(12, 4, 12, 4),
		MinWidth:   t.Button.MinWidth,
		MinHeight:  t.Button.MinHeight,
	}
}

// WithColor 设置按钮颜色（链式调用）
func (b *Button) WithColor(color types.Color) *Button {
	b.Color = color
	return b
}

// WithHoverColor 设置悬停颜色（链式调用）
func (b *Button) WithHoverColor(color types.Color) *Button {
	b.HoverColor = color
	return b
}

// WithMinWidth 设置最小宽度（链式调用）
func (b *Button) WithMinWidth(w float64) *Button {
	b.MinWidth = w
	return b
}

// WithMinHeight 设置最小高度（链式调用）
func (b *Button) WithMinHeight(h float64) *Button {
	b.MinHeight = h
	return b
}

// WithTextColor 设置按钮文字颜色（链式调用）
func (b *Button) WithTextColor(color types.Color) *Button {
	b.TextColor = color
	return b
}

// WithFontSize 设置按钮字体大小（链式调用）
func (b *Button) WithFontSize(size float64) *Button {
	b.FontSize = size
	return b
}

// WithPadding 设置按钮内边距（链式调用）
func (b *Button) WithPadding(padding types.Inset) *Button {
	b.Padding = padding
	return b
}

// Button 按钮控件
type Button struct {
	SingleChildWidget
	Text       string
	OnClick    func()
	Color      types.Color
	HoverColor types.Color
	TextColor  types.Color // 文字颜色（从主题读取）
	Padding    types.Inset // 内边距
	MinWidth   float64
	MinHeight  float64
	FontSize   float64 // 字体大小

	// 原生图标支持（CSS/Element/Flutter 风格）：设了 Icon 即在按钮内渲染 Lucide 图标，
	// 与 Text 组合时自动排成「图标 + 间距 + 文字」一行并整体居中——调用方无需自己拼 Div。
	Icon      string       // Lucide 图标名（空=无图标）
	IconSize  float64      // 图标边长（默认 14）
	IconColor *types.Color // 图标色（默认跟随 TextColor）
	IconGap   float64      // 图标与文字间距（默认 6）
	IconRight bool         // 图标置于文字右侧（默认左侧）

	BorderColor  *types.Color // 边框颜色（nil=无边框）
	BorderWidth  float64      // 边框宽度（0=无）
	BorderRadius float64      // 圆角半径（0=默认 4）

	FocusRingColor types.Color // CSS :focus —— 聚焦环颜色（零值=按钮基色半透明）
}

// effectiveChild 返回按钮实际渲染的子内容：显式 Child 优先；否则若设了 Icon，
// 自动合成「图标[ + 间距 + 文字]」一行（图标按钮 / 图标+文字按钮无需调用方自己拼 Div）。
// 返回 nil 表示纯文字（或空）按钮，走 Paint 的 DrawTextAligned 文字居中路径。
func (b *Button) effectiveChild() Widget {
	if b.Child != nil {
		return b.Child
	}
	if b.Icon == "" {
		return nil
	}
	isz := b.IconSize
	if isz <= 0 {
		isz = 14
	}
	icol := types.ColorWhite
	if b.TextColor != (types.Color{}) {
		icol = b.TextColor
	}
	if b.IconColor != nil {
		icol = *b.IconColor
	}
	icon := Lucide(b.Icon, IconSize(isz), IconColor(icol))
	if b.Text == "" {
		return icon // 纯图标按钮
	}
	gap := b.IconGap
	if gap <= 0 {
		gap = 6
	}
	tcol := b.TextColor
	if tcol == (types.Color{}) {
		tcol = types.ColorWhite
	}
	txt := NewText(b.Text, tcol)
	if b.FontSize > 0 {
		txt.Font.Size = b.FontSize
	}
	spacer := Div(Style{Width: gap})
	if b.IconRight {
		return Div(Style{FlexDirection: "row", AlignItems: "center"}, txt, spacer, icon)
	}
	return Div(Style{FlexDirection: "row", AlignItems: "center"}, icon, spacer, txt)
}

// ButtonElement 按钮 Element
type ButtonElement struct {
	BaseElement
	button        *Button
	child         Element
	hovered       bool
	pressed       bool
	focused       bool                  // CSS :focus —— 聚焦态（画 focus ring）
	hoverProgress float64               // hover 动画进度 0→1（颜色/高亮平滑过渡）
	hoverCtrl     *animation.Controller // 驱动 hoverProgress
}

// animateHover 把 hover 进度平滑过渡到 target（0 或 1）。
// 只触发重绘（不重建/布局），hover 不改变结构，开销很小。
func (e *ButtonElement) animateHover(target float64) {
	from := e.hoverProgress
	if e.hoverCtrl != nil {
		e.hoverCtrl.Stop()
	}
	e.hoverCtrl = animation.NewController(150*time.Millisecond, animation.EaseOutQuad)
	e.hoverCtrl.OnUpdate = func(v float64) {
		e.hoverProgress = animation.LerpFloat(from, target, v)
		e.MarkNeedsPaint()
	}
	e.hoverCtrl.Start()
}

// Focus / Blur —— CSS :focus（聚焦环）。不重写 IsFocused（保持 false），
// 避免 app 把按钮当作需持续重绘的焦点元素（光标闪烁逻辑）。
func (e *ButtonElement) Focus() {
	if !e.focused {
		e.focused = true
		e.MarkNeedsPaint()
	}
}

func (e *ButtonElement) Blur() {
	if e.focused {
		e.focused = false
		e.MarkNeedsPaint()
	}
}

// CreateElement 创建 ButtonElement
func (b *Button) CreateElement() Element {
	return &ButtonElement{
		BaseElement: BaseElement{widget: b},
		button:      b,
	}
}

// Update 更新 Widget 配置并同步内部 button 引用（否则文本/样式/回调更新失效）。
func (e *ButtonElement) Update(newWidget Widget) {
	e.widget = newWidget
	e.dirty = true
	if b, ok := newWidget.(*Button); ok {
		e.button = b
	}
}

// Build 构建子 Element。显式 Child 或 Icon 合成的子内容都走这里，
// 纯文字按钮返回 nil（文字在 Paint 里用 DrawTextAligned 居中绘制）。
func (e *ButtonElement) Build() []Element {
	if child := e.button.effectiveChild(); child != nil {
		e.child = CreateElementFor(child)
		e.child.Mount(e, 0)
		e.children = []Element{e.child}
		return e.children
	}
	e.child = nil
	e.children = nil
	return nil
}

// Layout 执行布局
func (e *ButtonElement) Layout(ctx *layout.LayoutContext) layout.LayoutResult {
	b := e.button
	minW := b.MinWidth
	minH := b.MinHeight
	// 64 默认最小宽只给「文本按钮」；有自定义子内容（图标/图标+文字）时按内容紧凑，
	// 否则小内容（如「自动」胶囊 ~36px）会浮在 64 宽盒子里、看着像没居中。需要最小宽自设 MinWidth。
	if minW <= 0 && e.child == nil {
		minW = 64
	}
	if minH <= 0 {
		minH = 32
	}

	// 最小尺寸考虑内边距
	paddingTotalW := b.Padding.Left + b.Padding.Right
	paddingTotalH := b.Padding.Top + b.Padding.Bottom
	minW += paddingTotalW
	minH += paddingTotalH

	if e.child != nil {
		// 子（图标/任意内容）按内容布局，按钮取 max(子+padding, min)，并把子居中。
		// ⚠️ 高度不限（MaxHeight 放开）：否则把按钮的可用高传给子，flex 行/占位 Div 会**拉伸**
		// 到该高（如行内 gap 占位 Div 撑满 → 行高被顶到 24），按钮随之变高、内容被 paddingTop
		// 顶偏，看着「偏下」。放开高度让子取自然高，再居中才准。
		result := e.child.Layout(&layout.LayoutContext{Constraints: layout.BoxConstraints{
			MaxWidth: ctx.Constraints.MaxWidth, MaxHeight: float64(1 << 30),
		}})
		w := result.Size.Width + paddingTotalW
		h := result.Size.Height + paddingTotalH
		if w < minW {
			w = minW
		}
		if h < minH {
			h = minH
		}
		e.size = types.Size{Width: w, Height: h}
		e.child.SetPosition(types.Point{
			X: (e.size.Width - result.Size.Width) / 2,
			Y: (e.size.Height - result.Size.Height) / 2,
		})
	} else {
		e.size = types.Size{Width: minW, Height: minH}
	}
	e.size = ctx.Constraints.Constrain(e.size)
	return layout.LayoutResult{Size: e.size}
}

// Paint 绘制按钮
func (e *ButtonElement) Paint(cvs canvas.Canvas, offset types.Point) {
	pos := e.Offset()
	b := e.button
	base := b.Color
	if base == (types.Color{}) {
		base = types.ColorFromRGB(66, 133, 244)
	}
	// 悬停目标色：HoverColor 或自动变暗 20%
	hover := b.HoverColor
	if hover == (types.Color{}) {
		hover = types.Color{
			R: uint8(float64(base.R) * 0.80),
			G: uint8(float64(base.G) * 0.80),
			B: uint8(float64(base.B) * 0.80),
			A: base.A,
		}
	}
	// 按 hover 进度在常态色与悬停色之间平滑插值
	color := animation.LerpColor(base, hover, e.hoverProgress)

	// 按下时使用更深的颜色（RGB 各分量乘以 0.70，比原来的 0.85 更明显）
	if e.pressed {
		color = types.Color{
			R: uint8(float64(color.R) * 0.70),
			G: uint8(float64(color.G) * 0.70),
			B: uint8(float64(color.B) * 0.70),
			A: color.A,
		}
	}

	// 绘制按钮背景（圆角可配，默认 4）
	radius := b.BorderRadius
	if radius == 0 {
		radius = 4
	}
	bgPaint := paint.DefaultPaint()
	bgPaint.Color = color
	cvs.DrawRoundedRect(pos.X, pos.Y, e.size.Width, e.size.Height, radius, bgPaint)
	// 自定义边框（CSS border）
	if b.BorderColor != nil && b.BorderWidth > 0 {
		bp := paint.DefaultStrokePaint()
		bp.Color = *b.BorderColor
		bp.StrokeWidth = b.BorderWidth
		cvs.DrawRoundedRect(pos.X, pos.Y, e.size.Width, e.size.Height, radius, bp)
	}

	// 焦点环（CSS :focus）——聚焦时在按钮外围画半透明描边环（a11y 反馈）
	if e.focused {
		ring := b.FocusRingColor
		if ring == (types.Color{}) {
			ring = types.Color{R: base.R, G: base.G, B: base.B, A: 120} // 默认基色半透明
		}
		ringPaint := paint.DefaultStrokePaint()
		ringPaint.Color = ring
		ringPaint.StrokeWidth = 2
		cvs.DrawRoundedRect(pos.X-2, pos.Y-2, e.size.Width+4, e.size.Height+4, 6, ringPaint)
	}

	// 悬停高亮边框（随 hover 进度淡入）
	if e.hoverProgress > 0.01 && !e.pressed {
		highlightPaint := paint.DefaultStrokePaint()
		highlightPaint.Color = types.Color{R: 255, G: 255, B: 255, A: uint8(60 * e.hoverProgress)}
		highlightPaint.StrokeWidth = 1.5
		cvs.DrawRoundedRect(pos.X+1, pos.Y+1, e.size.Width-2, e.size.Height-2, 3, highlightPaint)
	}

	// 有子内容（图标/任意 Widget）：渲染居中的子，跳过文字绘制。
	if e.child != nil {
		e.child.Paint(cvs, e.Offset())
		return
	}

	// 确定文字颜色和字体
	textColor := b.TextColor
	if textColor == (types.Color{}) {
		textColor = types.ColorWhite
	}
	font := canvas.DefaultFont()
	font.Size = b.FontSize
	font.Weight = canvas.FontWeightBold
	if font.Size <= 0 {
		font.Size = 14
	}

	// 计算内容区域（扣除内边距）
	contentX := pos.X + b.Padding.Left
	contentY := pos.Y + b.Padding.Top
	contentW := e.size.Width - b.Padding.Left - b.Padding.Right
	contentH := e.size.Height - b.Padding.Top - b.Padding.Bottom
	if contentW < 0 {
		contentW = 0
	}
	if contentH < 0 {
		contentH = 0
	}

	// 在内容区域内水平 + 垂直居中文字（复用统一对齐绘制原语）
	contentRect := types.Rect{X: contentX, Y: contentY, Width: contentW, Height: contentH}
	canvas.DrawTextAligned(cvs, b.Text, contentRect, font, textColor, canvas.HAlignCenter, canvas.VAlignMiddle)
}

// HandleEvent 处理事件
func (e *ButtonElement) HandleEvent(ev event.Event) bool {
	switch ev.Type() {
	case event.TypeMouseDown:
		e.pressed = true
		// 按下状态变化需要重绘显示 pressed 颜色
		if OnNeedsRepaint != nil {
			OnNeedsRepaint()
		}
		return true
	case event.TypeMouseUp:
		if e.pressed && e.button.OnClick != nil {
			e.button.OnClick()
		}
		e.pressed = false
		// 释放状态变化需要重绘恢复原始颜色
		if OnNeedsRepaint != nil {
			OnNeedsRepaint()
		}
		return true
	case event.TypeMouseEnter:
		e.hovered = true
		e.animateHover(1) // 平滑过渡到悬停态
		return true
	case event.TypeMouseLeave:
		e.hovered = false
		e.animateHover(0) // 平滑过渡回常态
		return true
	case event.TypeKeyDown:
		keyEv, ok := ev.(*event.KeyEvent)
		if ok && (keyEv.Key == "Enter" || keyEv.Key == "Space") {
			e.pressed = true
			if OnNeedsRepaint != nil {
				OnNeedsRepaint()
			}
			return true
		}
		return false
	case event.TypeKeyUp:
		keyEv, ok := ev.(*event.KeyEvent)
		if ok && (keyEv.Key == "Enter" || keyEv.Key == "Space") {
			if e.pressed && e.button.OnClick != nil {
				e.button.OnClick()
			}
			e.pressed = false
			if OnNeedsRepaint != nil {
				OnNeedsRepaint()
			}
			return true
		}
		return false
	}
	return false
}
