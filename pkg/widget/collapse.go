package widget

import (
	"github.com/hoonfeng/goui/pkg/canvas"
	"github.com/hoonfeng/goui/pkg/event"
	"github.com/hoonfeng/goui/internal/layout"
	"github.com/hoonfeng/goui/pkg/paint"
	"github.com/hoonfeng/goui/pkg/types"
)

// ─── Collapse 折叠面板（1:1 复刻 el-collapse）─────────────────────
// 多个折叠项：标题栏(点击展开/收起 + 方向箭头) + 内容区。
// 内容元素始终构建(状态不丢)，收起时不布局/不绘制；支持手风琴模式。

// CollapseItem 一个折叠项。
type CollapseItem struct {
	Title   string
	Content Widget
}

// Collapse 折叠面板。
type Collapse struct {
	StatelessWidget
	Items           []CollapseItem
	Accordion       bool  // 手风琴(同时只展开一个)
	DefaultExpanded []int // 默认展开的项下标
}

func NewCollapse(items ...CollapseItem) *Collapse  { return &Collapse{Items: items} }
func (c *Collapse) WithAccordion(a bool) *Collapse { c.Accordion = a; return c }
func (c *Collapse) WithExpanded(idx ...int) *Collapse {
	c.DefaultExpanded = idx
	return c
}

const collapseHeaderH = 48.0

func (c *Collapse) CreateElement() Element {
	return &CollapseElement{BaseElement: BaseElement{widget: c}, collapse: c, hoverIdx: -1}
}

type CollapseElement struct {
	BaseElement
	collapse    *Collapse
	contentEls  []Element
	expanded    []bool
	headerRects []types.Rect
	hoverIdx    int
}

func (e *CollapseElement) Build() []Element {
	n := len(e.collapse.Items)
	if e.contentEls == nil {
		e.contentEls = make([]Element, n)
		e.expanded = make([]bool, n)
		for _, idx := range e.collapse.DefaultExpanded {
			if idx >= 0 && idx < n {
				e.expanded[idx] = true
			}
		}
		for i, item := range e.collapse.Items {
			e.contentEls[i] = CreateElementFor(item.Content)
			e.contentEls[i].Mount(e, i)
		}
	} else {
		for i, item := range e.collapse.Items {
			if i < len(e.contentEls) {
				e.contentEls[i].Update(item.Content)
			}
		}
	}
	e.children = e.contentEls
	return e.children
}

func (e *CollapseElement) Layout(ctx *layout.LayoutContext) layout.LayoutResult {
	w := ctx.Constraints.MaxWidth
	y := 0.0
	e.headerRects = e.headerRects[:0]
	for i := range e.collapse.Items {
		e.headerRects = append(e.headerRects, types.Rect{X: 0, Y: y, Width: w, Height: collapseHeaderH})
		y += collapseHeaderH
		if e.expanded[i] && e.contentEls[i] != nil {
			res := e.contentEls[i].Layout(&layout.LayoutContext{Constraints: layout.BoxConstraints{
				MinWidth: 0, MaxWidth: w - 32, MinHeight: 0, MaxHeight: ctx.Constraints.MaxHeight,
			}})
			e.contentEls[i].SetPosition(types.Point{X: 16, Y: y + 12})
			y += res.Size.Height + 24
		}
	}
	e.size = ctx.Constraints.Constrain(types.Size{Width: w, Height: y})
	return layout.LayoutResult{Size: e.size}
}

func (e *CollapseElement) Paint(cvs canvas.Canvas, offset types.Point) {
	pos := e.Offset()
	font := canvas.DefaultFont()
	font.Size = 14
	font.Weight = canvas.FontWeightBold

	for i, item := range e.collapse.Items {
		hr := e.headerRects[i]
		hy := pos.Y + hr.Y

		// header hover 背景
		if i == e.hoverIdx {
			hp := paint.DefaultPaint()
			hp.Color = elFill()
			cvs.DrawRect(pos.X, hy, hr.Width, collapseHeaderH, hp)
		}

		// 标题
		titleRect := types.Rect{X: pos.X + 16, Y: hy, Width: hr.Width - 48, Height: collapseHeaderH}
		canvas.DrawTextAligned(cvs, item.Title, titleRect, font, elTextPrimary(), canvas.HAlignLeft, canvas.VAlignMiddle)

		// 方向箭头(右侧)：展开 ▼ / 收起 ▶
		acx := pos.X + hr.Width - 24
		acy := hy + collapseHeaderH/2
		ap := paint.DefaultStrokePaint()
		ap.Color = elTextSecondary()
		ap.StrokeWidth = 1.5
		if e.expanded[i] {
			cvs.DrawLine(acx-4, acy-2, acx, acy+2, ap)
			cvs.DrawLine(acx, acy+2, acx+4, acy-2, ap)
		} else {
			cvs.DrawLine(acx-2, acy-4, acx+2, acy, ap)
			cvs.DrawLine(acx+2, acy, acx-2, acy+4, ap)
		}

		// 底部分隔线
		lp := paint.DefaultStrokePaint()
		lp.Color = elBorderLight()
		lp.StrokeWidth = 1
		cvs.DrawLine(pos.X, hy+collapseHeaderH, pos.X+hr.Width, hy+collapseHeaderH, lp)

		// 内容(展开时)
		if e.expanded[i] && e.contentEls[i] != nil {
			e.contentEls[i].Paint(cvs, offset)
		}
	}
}

func (e *CollapseElement) headerAt(localX, localY float64) int {
	for i, hr := range e.headerRects {
		if localY >= hr.Y && localY < hr.Y+collapseHeaderH && localX >= 0 && localX < hr.Width {
			return i
		}
	}
	return -1
}

func (e *CollapseElement) HandleEvent(ev event.Event) bool {
	me, ok := ev.(*event.MouseEvent)
	if !ok {
		return false
	}
	pos := e.Offset()
	idx := e.headerAt(me.X-pos.X, me.Y-pos.Y)
	switch ev.Type() {
	case event.TypeMouseMove, event.TypeMouseEnter:
		e.hoverIdx = idx
		e.MarkNeedsPaint()
		return true
	case event.TypeMouseLeave:
		e.hoverIdx = -1
		e.MarkNeedsPaint()
		return true
	case event.TypeMouseDown:
		if idx >= 0 {
			if e.collapse.Accordion {
				was := e.expanded[idx]
				for j := range e.expanded {
					e.expanded[j] = false
				}
				e.expanded[idx] = !was
			} else {
				e.expanded[idx] = !e.expanded[idx]
			}
			if OnNeedsLayout != nil {
				OnNeedsLayout() // 展开/收起改变高度 → relayout
			}
			return true
		}
	}
	return false
}

func (e *CollapseElement) Update(newWidget Widget) {
	if nc, ok := newWidget.(*Collapse); ok {
		e.collapse = nc
		e.BaseElement.widget = newWidget
		e.dirty = true
	}
}
