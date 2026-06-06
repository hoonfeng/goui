package widget

import (
	"github.com/user/goui/internal/canvas"
	"github.com/user/goui/internal/event"
	"github.com/user/goui/internal/layout"
	"github.com/user/goui/internal/paint"
	"github.com/user/goui/internal/types"
)

// ─── Popover 弹出框 ────────────────────────────────────────────
// 1:1 复刻 el-popover 核心：包裹触发元素，弹出带 title + 富内容的面板。
//   props : title / content / placement / trigger(hover·click) / width
//   依赖  : Overlay 浮层定位地基；复用 TooltipTrigger 作触发方式

// Popover 弹出框。
type Popover struct {
	StatelessWidget
	Child     Widget         // 触发元素
	Title     string         // 标题(可选)
	Content   string         // 纯文本内容(Body 为空时使用)
	Body      Widget         // 富内容(优先于 Content)
	Placement Placement
	Trigger   TooltipTrigger // 复用 hover/click
	Width     float64
	Gap       float64
}

// NewPopover 创建默认（click 触发 / bottom / 宽 200）弹出框。
func NewPopover(child Widget) *Popover {
	return &Popover{Child: child, Placement: PlacementBottom, Trigger: TooltipClick, Width: 200}
}

func (p *Popover) WithTitle(s string) *Popover            { p.Title = s; return p }
func (p *Popover) WithContent(s string) *Popover          { p.Content = s; return p }
func (p *Popover) WithBody(w Widget) *Popover             { p.Body = w; return p }
func (p *Popover) WithPlacement(pl Placement) *Popover    { p.Placement = pl; return p }
func (p *Popover) WithTrigger(tr TooltipTrigger) *Popover { p.Trigger = tr; return p }
func (p *Popover) WithWidth(w float64) *Popover           { p.Width = w; return p }

func (p *Popover) CreateElement() Element {
	return &PopoverElement{BaseElement: BaseElement{widget: p}, popover: p}
}

type PopoverElement struct {
	BaseElement
	popover   *Popover
	childEl   Element
	overlayID int
	shown     bool
}

func (e *PopoverElement) Build() []Element {
	if e.popover.Child != nil {
		e.childEl = CreateElementFor(e.popover.Child)
		e.childEl.Mount(e, 0)
		e.children = []Element{e.childEl}
		return e.children
	}
	e.children = nil
	return nil
}

func (e *PopoverElement) Layout(ctx *layout.LayoutContext) layout.LayoutResult {
	if e.childEl != nil {
		res := e.childEl.Layout(ctx)
		e.childEl.SetPosition(types.Point{})
		e.size = res.Size
	}
	return layout.LayoutResult{Size: e.size}
}

func (e *PopoverElement) Paint(cvs canvas.Canvas, offset types.Point) {
	if e.childEl != nil {
		e.childEl.Paint(cvs, offset)
	}
}

func (e *PopoverElement) HandleEvent(ev event.Event) bool {
	switch ev.Type() {
	case event.TypeMouseEnter:
		if e.popover.Trigger == TooltipHover {
			e.show()
		}
	case event.TypeMouseLeave:
		if e.popover.Trigger == TooltipHover {
			e.hide()
		}
	case event.TypeMouseDown:
		if e.popover.Trigger == TooltipClick {
			if e.shown {
				e.hide()
			} else {
				e.show()
			}
		}
	}
	return false
}

func (e *PopoverElement) show() {
	if e.shown || e.childEl == nil {
		return
	}
	off := e.childEl.Offset()
	sz := e.childEl.Size()
	panel := &popoverPanel{
		title:   e.popover.Title,
		content: e.popover.Content,
		body:    e.popover.Body,
		width:   e.popover.Width,
	}
	entry := &OverlayEntry{
		AnchorRect:     types.Rect{X: off.X, Y: off.Y, Width: sz.Width, Height: sz.Height},
		Content:        panel,
		Placement:      e.popover.Placement,
		Gap:            e.popover.Gap,
		Transition:     "fade",
		OnClickOutside: func() { e.hide() },
	}
	panel.entry = entry
	e.overlayID = ShowOverlay(entry)
	e.shown = true
}

func (e *PopoverElement) hide() {
	if !e.shown {
		return
	}
	HideOverlay(e.overlayID)
	e.shown = false
}

func (e *PopoverElement) Update(newWidget Widget) {
	if np, ok := newWidget.(*Popover); ok {
		e.popover = np
		e.BaseElement.widget = newWidget
		e.dirty = true
	}
}

// ShowPopover 以命令式方式在锚点旁显示一个带 title+内容的弹出框，返回浮层 id。
// 适合演示/测试或程序化弹层；用 HideOverlay(id) 关闭。
func ShowPopover(title, content string, anchor types.Rect, placement Placement, width float64) int {
	panel := &popoverPanel{title: title, content: content, width: width}
	entry := &OverlayEntry{AnchorRect: anchor, Content: panel, Placement: placement, Transition: "fade"}
	panel.entry = entry
	return ShowOverlay(entry)
}

// ─── popoverPanel 面板(浮层内容) ───────────────────────────────

type popoverPanel struct {
	StatelessWidget
	title   string
	content string
	body    Widget
	width   float64
	entry   *OverlayEntry
}

func (p *popoverPanel) CreateElement() Element {
	return &popoverPanelElement{BaseElement: BaseElement{widget: p}, panel: p}
}

type popoverPanelElement struct {
	BaseElement
	panel  *popoverPanel
	bodyEl Element
	titleH float64
}

func (e *popoverPanelElement) Build() []Element {
	body := e.panel.body
	if body == nil && e.panel.content != "" {
		body = NewText(e.panel.content, CurrentTheme().Popover.TextColor)
	}
	if body != nil {
		e.bodyEl = CreateElementFor(body)
		e.bodyEl.Mount(e, 0)
		e.children = []Element{e.bodyEl}
		return e.children
	}
	e.bodyEl = nil
	e.children = nil
	return nil
}

func (e *popoverPanelElement) Layout(ctx *layout.LayoutContext) layout.LayoutResult {
	t := CurrentTheme().Popover
	width := e.panel.width
	if width <= 0 {
		width = 200
	}
	innerW := width - t.PaddingH*2

	e.titleH = 0
	if e.panel.title != "" {
		e.titleH = canvas.GetFaceLineHeight(t.TitleSize) + 6
	}
	bodyH := 0.0
	if e.bodyEl != nil {
		res := e.bodyEl.Layout(&layout.LayoutContext{Constraints: layout.BoxConstraints{
			MinWidth: 0, MaxWidth: innerW, MinHeight: 0, MaxHeight: 100000,
		}})
		e.bodyEl.SetPosition(types.Point{X: t.PaddingH, Y: t.PaddingV + e.titleH})
		bodyH = res.Size.Height
	}
	h := t.PaddingV*2 + e.titleH + bodyH
	e.size = ctx.Constraints.Constrain(types.Size{Width: width, Height: h})
	return layout.LayoutResult{Size: e.size}
}

func (e *popoverPanelElement) Paint(cvs canvas.Canvas, offset types.Point) {
	pos := e.Offset()
	t := CurrentTheme().Popover

	// 轻微阴影
	shadow := paint.DefaultPaint()
	shadow.Color = types.ColorFromRGBA(0, 0, 0, 18)
	cvs.DrawRoundedRect(pos.X, pos.Y+2, e.size.Width, e.size.Height, t.Radius, shadow)

	// 面板背景 + 边框
	bgp := paint.DefaultPaint()
	bgp.Color = t.Bg
	cvs.DrawRoundedRect(pos.X, pos.Y, e.size.Width, e.size.Height, t.Radius, bgp)
	bp := paint.DefaultStrokePaint()
	bp.Color = t.Border
	bp.StrokeWidth = 1
	cvs.DrawRoundedRect(pos.X+0.5, pos.Y+0.5, e.size.Width-1, e.size.Height-1, t.Radius, bp)

	// 箭头
	e.paintArrow(cvs, pos, t.Bg)

	// title
	if e.panel.title != "" {
		font := canvas.DefaultFont()
		font.Size = t.TitleSize
		font.Weight = canvas.FontWeightBold
		tp := paint.DefaultPaint()
		tp.Color = t.TitleColor
		ascent := canvas.GetFaceAscent(t.TitleSize)
		cvs.DrawText(e.panel.title, pos.X+t.PaddingH, pos.Y+t.PaddingV+ascent, font, tp)
	}

	// body
	if e.bodyEl != nil {
		e.bodyEl.Paint(cvs, offset)
	}
}

func (e *popoverPanelElement) paintArrow(cvs canvas.Canvas, pos types.Point, bg types.Color) {
	pl := PlacementBottom
	if e.panel.entry != nil {
		pl = e.panel.entry.ResolvedPlacement()
	}
	m, _ := splitPlacement(pl)
	const a = 6.0
	cx := pos.X + e.size.Width/2
	cy := pos.Y + e.size.Height/2
	path := canvas.NewPath()
	switch m {
	case mBottom:
		path.MoveTo(cx-a, pos.Y)
		path.LineTo(cx, pos.Y-a)
		path.LineTo(cx+a, pos.Y)
	case mTop:
		by := pos.Y + e.size.Height
		path.MoveTo(cx-a, by)
		path.LineTo(cx, by+a)
		path.LineTo(cx+a, by)
	case mRight:
		path.MoveTo(pos.X, cy-a)
		path.LineTo(pos.X-a, cy)
		path.LineTo(pos.X, cy+a)
	case mLeft:
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
