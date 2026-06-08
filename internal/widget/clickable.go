package widget

import (
	"github.com/user/goui/internal/canvas"
	"github.com/user/goui/internal/event"
	"github.com/user/goui/internal/layout"
	"github.com/user/goui/internal/paint"
	"github.com/user/goui/internal/types"
)

// Clickable 给任意子内容加「点击 + 悬停背景」的轻量容器。与 Button 不同：**不居中**子内容
// （左上对齐），适合列表行/卡片——子内容左对齐、整行可点、悬停高亮、撑满父给的紧约束（铺满宽）。
type Clickable struct {
	SingleChildWidget
	OnClick    func()
	OnHoverChange func(bool) // 悬停进入/离开**整个子树**时回调(true/false)。用于 hover 揭示（如消息上的操作按钮）——
	// goui hover 走子树语义（见 app.dispatchHoverChange），鼠标移到子元素上时本元素仍算 hover，故无闪烁。
	Color      types.Color // 常态背景（零值=透明）
	HoverColor types.Color // 悬停背景（零值=不变）
	Padding    types.Inset
	// StopPropagation: 点击后阻断冒泡，使外层 Clickable/Button 不再收到——嵌套可点（如标签
	// 里的关闭×按钮，点×只关闭、不触发外层标签切换）用。默认 false（冒泡，浮层点外关闭等需要）。
	StopPropagation bool
}

// CreateElement 创建 ClickableElement。
func (c *Clickable) CreateElement() Element {
	return &ClickableElement{BaseElement: BaseElement{widget: c}, clickable: c}
}

// ClickableElement Clickable 的运行时 Element。
type ClickableElement struct {
	BaseElement
	clickable *Clickable
	child     Element
	hovered   bool
	pressed   bool
}

// Update 同步配置。
func (e *ClickableElement) Update(nw Widget) {
	e.widget = nw
	e.dirty = true
	if c, ok := nw.(*Clickable); ok {
		e.clickable = c
	}
}

// Build 构建子 Element。
func (e *ClickableElement) Build() []Element {
	if e.clickable.Child != nil {
		e.child = CreateElementFor(e.clickable.Child)
		e.child.Mount(e, 0)
		e.children = []Element{e.child}
		return e.children
	}
	e.child = nil
	e.children = nil
	return nil
}

// Layout 子左上对齐 + padding；自身撑满父给的紧约束（列表行铺满宽），否则贴合内容。
func (e *ClickableElement) Layout(ctx *layout.LayoutContext) layout.LayoutResult {
	c := e.clickable
	padW := c.Padding.Left + c.Padding.Right
	padH := c.Padding.Top + c.Padding.Bottom
	var childSize types.Size
	if e.child != nil {
		maxW := ctx.Constraints.MaxWidth
		if maxW < float64(1<<30) {
			maxW -= padW // 给子留 padding，使其按行铺满可用宽
		}
		// 把父的高度下限(AlignItems:stretch 会把它顶到容器高)传给子，使子填满被拉伸的高度。
		// 否则子按内容高、贴顶留空，且子上的边框/底色只有文字大小（如标签栏里的标签）。
		// MaxHeight 仍给无界，保留「不锁死内容高」语义（多行文本/列表可自然长高）。
		minH := ctx.Constraints.MinHeight - padH
		if minH < 0 {
			minH = 0
		}
		res := e.child.Layout(&layout.LayoutContext{Constraints: layout.BoxConstraints{
			MaxWidth: maxW, MinHeight: minH, MaxHeight: float64(1 << 30),
		}})
		childSize = res.Size
	}
	e.size = ctx.Constraints.Constrain(types.Size{
		Width:  childSize.Width + padW,
		Height: childSize.Height + padH,
	})
	if e.child != nil {
		// 子若仍比自身矮(未填满)，垂直居中兜底——否则贴顶下方留空。
		childY := c.Padding.Top
		if extra := e.size.Height - padH - childSize.Height; extra > 0 {
			childY += extra / 2
		}
		e.child.SetPosition(types.Point{X: c.Padding.Left, Y: childY})
	}
	return layout.LayoutResult{Size: e.size}
}

// Paint 画背景（悬停/常态）+ 子。
func (e *ClickableElement) Paint(cvs canvas.Canvas, offset types.Point) {
	pos := e.Offset()
	bg := e.clickable.Color
	if e.hovered && e.clickable.HoverColor != (types.Color{}) {
		bg = e.clickable.HoverColor
	}
	if bg != (types.Color{}) {
		p := paint.DefaultPaint()
		p.Color = bg
		cvs.DrawRect(pos.X, pos.Y, e.size.Width, e.size.Height, p)
	}
	if e.child != nil {
		e.child.Paint(cvs, pos)
	}
}

// HandleEvent 点击（按下→松手触发 OnClick）+ 悬停高亮。点击落在内层子上靠捕获路径冒泡上来。
func (e *ClickableElement) HandleEvent(ev event.Event) bool {
	switch ev.Type() {
	case event.TypeMouseDown:
		e.pressed = true
		if e.clickable.StopPropagation {
			ev.StopPropagation()
		}
		return true
	case event.TypeMouseUp:
		if e.pressed && e.clickable.OnClick != nil {
			e.clickable.OnClick()
		}
		e.pressed = false
		if e.clickable.StopPropagation {
			ev.StopPropagation()
		}
		return true
	case event.TypeMouseEnter:
		if !e.hovered {
			e.hovered = true
			e.MarkNeedsPaint()
			if e.clickable.OnHoverChange != nil {
				e.clickable.OnHoverChange(true)
			}
		}
	case event.TypeMouseLeave:
		if e.hovered || e.pressed {
			wasHovered := e.hovered
			e.hovered, e.pressed = false, false
			e.MarkNeedsPaint()
			if wasHovered && e.clickable.OnHoverChange != nil {
				e.clickable.OnHoverChange(false)
			}
		}
	}
	return false
}

// Cursor 可点击时手型；仅 hover 揭示（OnClick=nil）不强加手型，免误导成可点。
func (e *ClickableElement) Cursor() int {
	if e.clickable.OnClick != nil {
		return CursorHand
	}
	return CursorDefault
}
