package widget

import (
	"github.com/hoonfeng/goui/pkg/canvas"
	"github.com/hoonfeng/goui/pkg/event"
	"github.com/hoonfeng/goui/internal/layout"
	"github.com/hoonfeng/goui/pkg/paint"
	"github.com/hoonfeng/goui/pkg/types"
)

// ─── Tooltip 文字提示 ──────────────────────────────────────────
// 1:1 复刻 el-tooltip 核心：包裹子元素，鼠标悬停/点击时在其旁弹出气泡。
//   props : content / placement(12 方位) / effect(dark·light) / trigger(hover·click) / disabled
//   依赖  : Overlay 浮层定位地基

// TooltipEffect 提示风格。
type TooltipEffect int

const (
	TooltipDark  TooltipEffect = iota // dark（默认）：深底白字
	TooltipLight                      // light：白底深字 + 边框
)

// TooltipTrigger 触发方式。
type TooltipTrigger int

const (
	TooltipHover TooltipTrigger = iota // hover（默认）
	TooltipClick                       // click
)

// Tooltip 文字提示。
type Tooltip struct {
	StatelessWidget
	Child     Widget
	Content   string
	Placement Placement
	Effect    TooltipEffect
	Trigger   TooltipTrigger
	Disabled  bool
	Gap       float64
}

// NewTooltip 创建默认（dark / hover / bottom）提示。
func NewTooltip(content string, child Widget) *Tooltip {
	return &Tooltip{
		Child:     child,
		Content:   content,
		Placement: PlacementBottom,
		Effect:    TooltipDark,
		Trigger:   TooltipHover,
	}
}

func (t *Tooltip) WithPlacement(p Placement) *Tooltip      { t.Placement = p; return t }
func (t *Tooltip) WithEffect(e TooltipEffect) *Tooltip     { t.Effect = e; return t }
func (t *Tooltip) WithTrigger(tr TooltipTrigger) *Tooltip  { t.Trigger = tr; return t }
func (t *Tooltip) WithDisabled(b bool) *Tooltip            { t.Disabled = b; return t }

// CreateElement 创建 TooltipElement。
func (t *Tooltip) CreateElement() Element {
	return &TooltipElement{BaseElement: BaseElement{widget: t}, tooltip: t}
}

// TooltipElement 透明包裹子元素，并管理浮层的显示/隐藏。
type TooltipElement struct {
	BaseElement
	tooltip   *Tooltip
	childEl   Element
	overlayID int
	shown     bool
}

func (e *TooltipElement) Build() []Element {
	if e.tooltip.Child != nil {
		e.childEl = CreateElementFor(e.tooltip.Child)
		e.childEl.Mount(e, 0)
		e.children = []Element{e.childEl}
		return e.children
	}
	e.children = nil
	return nil
}

func (e *TooltipElement) Layout(ctx *layout.LayoutContext) layout.LayoutResult {
	if e.childEl != nil {
		res := e.childEl.Layout(ctx)
		e.childEl.SetPosition(types.Point{})
		e.size = res.Size
	}
	return layout.LayoutResult{Size: e.size}
}

func (e *TooltipElement) Paint(cvs canvas.Canvas, offset types.Point) {
	if e.childEl != nil {
		e.childEl.Paint(cvs, offset)
	}
}

// HandleEvent 依触发方式控制浮层显隐。返回 false 不拦截，子元素仍可交互。
func (e *TooltipElement) HandleEvent(ev event.Event) bool {
	switch ev.Type() {
	case event.TypeMouseEnter:
		if e.tooltip.Trigger == TooltipHover {
			e.show()
		}
	case event.TypeMouseLeave:
		if e.tooltip.Trigger == TooltipHover {
			e.hide()
		}
	case event.TypeMouseDown:
		if e.tooltip.Trigger == TooltipClick {
			if e.shown {
				e.hide()
			} else {
				e.show()
			}
		}
	}
	return false
}

func (e *TooltipElement) show() {
	if e.shown || e.tooltip.Disabled || e.childEl == nil || e.tooltip.Content == "" {
		return
	}
	off := e.childEl.Offset()
	sz := e.childEl.Size()
	bubble := &tooltipBubble{text: e.tooltip.Content, dark: e.tooltip.Effect == TooltipDark}
	entry := &OverlayEntry{
		AnchorRect: types.Rect{X: off.X, Y: off.Y, Width: sz.Width, Height: sz.Height},
		Content:    bubble,
		Placement:  e.tooltip.Placement,
		Gap:        e.tooltip.Gap,
		Transition: "fade",
	}
	if e.tooltip.Trigger == TooltipClick {
		entry.OnClickOutside = func() { e.hide() }
	}
	bubble.entry = entry
	e.overlayID = ShowOverlay(entry)
	e.shown = true
}

func (e *TooltipElement) hide() {
	if !e.shown {
		return
	}
	HideOverlay(e.overlayID)
	e.shown = false
}

func (e *TooltipElement) Update(newWidget Widget) {
	if nt, ok := newWidget.(*Tooltip); ok {
		e.tooltip = nt
		e.BaseElement.widget = newWidget
		e.dirty = true
	}
}

// ShowTooltip 以命令式方式在指定锚点矩形旁显示一个提示气泡，返回浮层 id。
// 适合非 hover 场景(新手引导等)或演示/测试；用 HideOverlay(id) 关闭。
func ShowTooltip(content string, anchor types.Rect, placement Placement, dark bool) int {
	bubble := &tooltipBubble{text: content, dark: dark}
	entry := &OverlayEntry{AnchorRect: anchor, Content: bubble, Placement: placement, Transition: "fade"}
	bubble.entry = entry
	return ShowOverlay(entry)
}

// ─── tooltipBubble 气泡(浮层内容) ──────────────────────────────

type tooltipBubble struct {
	StatelessWidget
	text  string
	dark  bool
	entry *OverlayEntry // 读翻转后的实际方位以确定箭头朝向
}

func (b *tooltipBubble) CreateElement() Element {
	return &tooltipBubbleElement{BaseElement: BaseElement{widget: b}, bubble: b}
}

type tooltipBubbleElement struct {
	BaseElement
	bubble *tooltipBubble
}

func (e *tooltipBubbleElement) Build() []Element { return nil }

func (e *tooltipBubbleElement) Layout(ctx *layout.LayoutContext) layout.LayoutResult {
	t := CurrentTheme().Tooltip
	font := canvas.DefaultFont()
	font.Size = t.FontSize
	tw := canvas.MeasureTextGlobal(e.bubble.text, font).Width
	w := tw + t.PaddingH*2
	h := canvas.GetFaceLineHeight(t.FontSize) + t.PaddingV*2
	e.size = ctx.Constraints.Constrain(types.Size{Width: w, Height: h})
	return layout.LayoutResult{Size: e.size}
}

func (e *tooltipBubbleElement) Paint(cvs canvas.Canvas, offset types.Point) {
	pos := e.Offset()
	t := CurrentTheme().Tooltip
	var bg, fg, border types.Color
	if e.bubble.dark {
		bg, fg = t.DarkBg, t.DarkText
	} else {
		bg, fg, border = t.LightBg, t.LightText, t.LightBorder
	}

	bgp := paint.DefaultPaint()
	bgp.Color = bg
	cvs.DrawRoundedRect(pos.X, pos.Y, e.size.Width, e.size.Height, t.Radius, bgp)
	if !e.bubble.dark {
		bp := paint.DefaultStrokePaint()
		bp.Color = border
		bp.StrokeWidth = 1
		cvs.DrawRoundedRect(pos.X+0.5, pos.Y+0.5, e.size.Width-1, e.size.Height-1, t.Radius, bp)
	}

	font := canvas.DefaultFont()
	font.Size = t.FontSize
	textRect := types.Rect{X: pos.X + t.PaddingH, Y: pos.Y, Width: e.size.Width - 2*t.PaddingH, Height: e.size.Height}
	canvas.DrawTextAligned(cvs, e.bubble.text, textRect, font, fg, canvas.HAlignLeft, canvas.VAlignMiddle)

	e.paintArrow(cvs, pos, bg)
}

// paintArrow 在气泡对着锚点的一侧画一个指向锚点的小三角。
func (e *tooltipBubbleElement) paintArrow(cvs canvas.Canvas, pos types.Point, bg types.Color) {
	pl := PlacementBottom
	if e.bubble.entry != nil {
		pl = e.bubble.entry.ResolvedPlacement()
	}
	m, _ := splitPlacement(pl)
	const a = 5.0
	cx := pos.X + e.size.Width/2
	cy := pos.Y + e.size.Height/2
	path := canvas.NewPath()
	switch m {
	case mBottom: // 气泡在锚点下方 → 箭头朝上(顶边中点)
		path.MoveTo(cx-a, pos.Y)
		path.LineTo(cx, pos.Y-a)
		path.LineTo(cx+a, pos.Y)
	case mTop: // 气泡在上 → 箭头朝下(底边)
		by := pos.Y + e.size.Height
		path.MoveTo(cx-a, by)
		path.LineTo(cx, by+a)
		path.LineTo(cx+a, by)
	case mRight: // 气泡在右 → 箭头朝左(左边)
		path.MoveTo(pos.X, cy-a)
		path.LineTo(pos.X-a, cy)
		path.LineTo(pos.X, cy+a)
	case mLeft: // 气泡在左 → 箭头朝右(右边)
		bx := pos.X + e.size.Width
		path.MoveTo(bx, cy-a)
		path.LineTo(bx+a, cy)
		path.LineTo(bx, cy+a)
	}
	path.Close()
	ap := paint.DefaultPaint()
	ap.Color = bg
	cvs.DrawPath(path, ap)
}
