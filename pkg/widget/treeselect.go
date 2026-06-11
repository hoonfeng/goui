package widget

import (
	"github.com/hoonfeng/goui/pkg/canvas"
	"github.com/hoonfeng/goui/pkg/event"
	"github.com/hoonfeng/goui/internal/layout"
	"github.com/hoonfeng/goui/pkg/paint"
	"github.com/hoonfeng/goui/pkg/types"
)

// ─── TreeSelect 树选择（el-tree-select）─────────────────────────
// 复刻 Select 触发器(框+值+箭头) + 点击弹出浮层，浮层内容复用 TreeView；
// 选中节点回填其 Label 并关闭。

type TreeSelect struct {
	StatelessWidget
	pseudoColors // CSS 伪类：交互态边框色覆盖（:hover/:focus）
	Nodes       []*TreeNode
	Value       string
	Placeholder string
	OnChange    func(string)
	Width       float64
}

func NewTreeSelect(nodes ...*TreeNode) *TreeSelect {
	return &TreeSelect{Nodes: nodes, Width: 240}
}
func (t *TreeSelect) WithValue(v string) *TreeSelect          { t.Value = v; return t }
func (t *TreeSelect) WithPlaceholder(p string) *TreeSelect    { t.Placeholder = p; return t }
func (t *TreeSelect) WithOnChange(fn func(string)) *TreeSelect { t.OnChange = fn; return t }
func (t *TreeSelect) WithWidth(w float64) *TreeSelect         { t.Width = w; return t }

func (t *TreeSelect) CreateElement() Element {
	return &TreeSelectElement{BaseElement: BaseElement{widget: t}, ts: t}
}

type TreeSelectElement struct {
	BaseElement
	ts        *TreeSelect
	overlayID int
	open      bool
	hovered   bool
}

func (e *TreeSelectElement) Build() []Element { return nil }

func (e *TreeSelectElement) Layout(ctx *layout.LayoutContext) layout.LayoutResult {
	w := e.ts.Width
	if w <= 0 {
		w = 240
	}
	e.size = ctx.Constraints.Constrain(types.Size{Width: w, Height: 32})
	return layout.LayoutResult{Size: e.size}
}

func (e *TreeSelectElement) Paint(cvs canvas.Canvas, offset types.Point) {
	pos := e.Offset()
	W, H := e.size.Width, e.size.Height
	border := e.ts.borderOr(elBorder())
	if e.hovered || e.open {
		if e.open {
			border = e.ts.focusBorderOr(elPrimary())
		} else {
			border = e.ts.hoverBorderOr(elPrimary())
		}
	}
	bg := paint.DefaultPaint()
	bg.Color = elSurface()
	cvs.DrawRoundedRect(pos.X, pos.Y, W, H, 4, bg)
	bp := paint.DefaultStrokePaint()
	bp.Color = border
	bp.StrokeWidth = 1
	cvs.DrawRoundedRect(pos.X+0.5, pos.Y+0.5, W-1, H-1, 4, bp)

	label := e.ts.Value
	tc := elTextRegular()
	if label == "" {
		label, tc = orI18n(e.ts.Placeholder, "el.select.placeholder"), elPlaceholder()
	}
	font := canvas.DefaultFont()
	font.Size = 14
	canvas.DrawTextAligned(cvs, label, types.Rect{X: pos.X + 12, Y: pos.Y, Width: W - 40, Height: H},
		font, tc, canvas.HAlignLeft, canvas.VAlignMiddle)

	arrow := paint.DefaultStrokePaint()
	arrow.Color = elPlaceholder()
	arrow.StrokeWidth = 1.4
	acx, acy := pos.X+W-16, pos.Y+H/2
	if e.open {
		cvs.DrawLine(acx-4, acy+2, acx, acy-2, arrow)
		cvs.DrawLine(acx, acy-2, acx+4, acy+2, arrow)
	} else {
		cvs.DrawLine(acx-4, acy-2, acx, acy+2, arrow)
		cvs.DrawLine(acx, acy+2, acx+4, acy-2, arrow)
	}
}

func (e *TreeSelectElement) HandleEvent(ev event.Event) bool {
	switch ev.Type() {
	case event.TypeMouseEnter:
		e.hovered = true
		e.MarkNeedsPaint()
		return true
	case event.TypeMouseLeave:
		e.hovered = false
		e.MarkNeedsPaint()
		return true
	case event.TypeMouseDown:
		if e.open {
			e.hide()
		} else {
			e.show()
		}
		e.MarkNeedsPaint()
		return true
	}
	return false
}

func (e *TreeSelectElement) show() {
	if e.open {
		return
	}
	off := e.Offset()
	tree := &TreeView{
		RootNodes: e.ts.Nodes,
		OnSelect: func(n *TreeNode) {
			e.ts.Value = n.Label
			if e.ts.OnChange != nil {
				e.ts.OnChange(n.Label)
			}
			e.hide()
			e.MarkNeedsPaint()
		},
	}
	panel := Div(Style{
		Width: e.size.Width, Padding: types.EdgeInsets(6),
		BackgroundColor: types.ColorRef(255, 255, 255),
		BorderColor:     types.ColorRef(228, 231, 237), BorderWidth: 1, BorderRadius: 4,
		Shadow: &paint.Shadow{Offset: types.Point{X: 0, Y: 2}, Blur: 12, Color: types.ColorFromRGBA(0, 0, 0, 22)},
	}, tree)
	e.overlayID = ShowOverlay(&OverlayEntry{
		AnchorRect:     types.Rect{X: off.X, Y: off.Y, Width: e.size.Width, Height: e.size.Height},
		Content:        panel,
		Placement:      PlacementBottomStart,
		Gap:            4,
		Transition:     "fade",
		OnClickOutside: func() { e.hide(); e.MarkNeedsPaint() },
	})
	e.open = true
}

func (e *TreeSelectElement) hide() {
	if !e.open {
		return
	}
	HideOverlay(e.overlayID)
	e.open = false
}

func (e *TreeSelectElement) Update(newWidget Widget) {
	if nt, ok := newWidget.(*TreeSelect); ok {
		e.ts = nt
		e.BaseElement.widget = newWidget
		e.dirty = true
	}
}
