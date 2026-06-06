package widget

import (
	"sort"

	"github.com/user/goui/internal/canvas"
	"github.com/user/goui/internal/layout"
	"github.com/user/goui/internal/types"
)

// ─── L3 绝对定位：Stack + Positioned ───────────────────────────
// Stack 是绝对定位容器；子用 Positioned 包裹并设 top/right/bottom/left 定位，
// 非 Positioned 子默认左上；所有子按 z-index 层叠绘制与命中。
// 对应 CSS 的 position:relative(Stack 自身) + position:absolute(Positioned 子)。

// Positioned 在 Stack 中绝对定位子控件。
type Positioned struct {
	SingleChildWidget
	Left, Top, Right, Bottom             float64
	HasLeft, HasTop, HasRight, HasBottom bool
	ZIndex                               int
}

func NewPositioned(child Widget) *Positioned {
	return &Positioned{SingleChildWidget: SingleChildWidget{Child: child}}
}
func (p *Positioned) WithLeft(v float64) *Positioned   { p.Left = v; p.HasLeft = true; return p }
func (p *Positioned) WithTop(v float64) *Positioned    { p.Top = v; p.HasTop = true; return p }
func (p *Positioned) WithRight(v float64) *Positioned  { p.Right = v; p.HasRight = true; return p }
func (p *Positioned) WithBottom(v float64) *Positioned { p.Bottom = v; p.HasBottom = true; return p }
func (p *Positioned) WithZIndex(z int) *Positioned     { p.ZIndex = z; return p }

func (p *Positioned) CreateElement() Element {
	return &PositionedElement{BaseElement: BaseElement{widget: p}, positioned: p}
}

type PositionedElement struct {
	BaseElement
	positioned *Positioned
	child      Element
}

func (e *PositionedElement) Build() []Element {
	if e.positioned.Child != nil {
		e.child = CreateElementFor(e.positioned.Child)
		e.child.Mount(e, 0)
		e.children = []Element{e.child}
		return e.children
	}
	e.children = nil
	return nil
}

func (e *PositionedElement) Layout(ctx *layout.LayoutContext) layout.LayoutResult {
	if e.child != nil {
		res := e.child.Layout(ctx)
		e.child.SetPosition(types.Point{})
		e.size = res.Size
	}
	return layout.LayoutResult{Size: e.size}
}

func (e *PositionedElement) Paint(cvs canvas.Canvas, offset types.Point) {
	if e.child != nil {
		e.child.Paint(cvs, offset)
	}
}

func (e *PositionedElement) Update(newWidget Widget) {
	if np, ok := newWidget.(*Positioned); ok {
		e.positioned = np
		e.BaseElement.widget = newWidget
		e.dirty = true
	}
}

// Stack 绝对定位容器。
type Stack struct {
	MultiChildWidget
	Alignment layout.Alignment // 非定位子的对齐(默认左上)
}

func NewStack(children ...Widget) *Stack {
	return &Stack{MultiChildWidget: MultiChildWidget{Children: children}}
}

func (s *Stack) CreateElement() Element {
	return &StackElement{BaseElement: BaseElement{widget: s}, stack: s}
}

type StackElement struct {
	BaseElement
	stack *Stack
}

func (e *StackElement) Build() []Element {
	e.children = nil
	for i, w := range e.stack.Children {
		ce := CreateElementFor(w)
		ce.Mount(e, i)
		e.children = append(e.children, ce)
	}
	return e.children
}

// Children 按 z-index 升序返回，使绘制(顺序)与命中(逆序)都遵循层叠顺序。
func (e *StackElement) Children() []Element {
	order := make([]int, len(e.children))
	for i := range order {
		order[i] = i
	}
	sort.SliceStable(order, func(a, b int) bool {
		return zIndexOf(e.children[order[a]]) < zIndexOf(e.children[order[b]])
	})
	out := make([]Element, len(order))
	for i, idx := range order {
		out[i] = e.children[idx]
	}
	return out
}

func zIndexOf(child Element) int {
	if p, ok := child.Widget().(*Positioned); ok {
		return p.ZIndex
	}
	return 0
}

// Px 返回 float64 的指针，便于在 Style 中设置定位偏移：
// Style{Position: "absolute", Top: Px(10), Left: Px(20), ZIndex: 2}。
func Px(v float64) *float64 { return &v }

func (e *StackElement) Layout(ctx *layout.LayoutContext) layout.LayoutResult {
	// 先各自测量(loose)，再决定 Stack 尺寸：有界则填充，否则取最大子。
	maxW, maxH := 0.0, 0.0
	loose := layout.BoxConstraints{
		MinWidth: 0, MaxWidth: ctx.Constraints.MaxWidth, MinHeight: 0, MaxHeight: ctx.Constraints.MaxHeight,
	}
	for _, child := range e.children {
		res := child.Layout(&layout.LayoutContext{Constraints: loose})
		if res.Size.Width > maxW {
			maxW = res.Size.Width
		}
		if res.Size.Height > maxH {
			maxH = res.Size.Height
		}
	}
	w, h := maxW, maxH
	if ctx.Constraints.MaxWidth < float64(1<<30) {
		w = ctx.Constraints.MaxWidth
	}
	if ctx.Constraints.MaxHeight < float64(1<<30) {
		h = ctx.Constraints.MaxHeight
	}
	e.size = ctx.Constraints.Constrain(types.Size{Width: w, Height: h})

	// 定位每个子
	for _, child := range e.children {
		cs := child.Size()
		var x, y float64
		if p, ok := child.Widget().(*Positioned); ok {
			// 同轴两端都设 → 拉伸该轴填满（CSS absolute 的 left+right / top+bottom 拉伸语义）。
			tw, th := cs.Width, cs.Height
			if p.HasLeft && p.HasRight {
				tw = e.size.Width - p.Left - p.Right
			}
			if p.HasTop && p.HasBottom {
				th = e.size.Height - p.Top - p.Bottom
			}
			if tw != cs.Width || th != cs.Height {
				child.Layout(&layout.LayoutContext{Constraints: layout.BoxConstraints{MinWidth: tw, MaxWidth: tw, MinHeight: th, MaxHeight: th}})
				cs = child.Size()
			}
			if p.HasLeft {
				x = p.Left
			} else if p.HasRight {
				x = e.size.Width - cs.Width - p.Right
			}
			if p.HasTop {
				y = p.Top
			} else if p.HasBottom {
				y = e.size.Height - cs.Height - p.Bottom
			}
		}
		// 非定位子默认左上(0,0)
		child.SetPosition(types.Point{X: x, Y: y})
	}
	return layout.LayoutResult{Size: e.size}
}

func (e *StackElement) Paint(cvs canvas.Canvas, offset types.Point) {
	for _, child := range e.Children() { // z 升序：z 大者后画，在上层
		child.Paint(cvs, offset)
	}
}

func (e *StackElement) Update(newWidget Widget) {
	if ns, ok := newWidget.(*Stack); ok {
		e.stack = ns
		e.BaseElement.widget = newWidget
		e.dirty = true
	}
}
