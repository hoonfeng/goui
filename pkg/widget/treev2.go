package widget

import (
	"github.com/hoonfeng/goui/pkg/canvas"
	"github.com/hoonfeng/goui/pkg/event"
	"github.com/hoonfeng/goui/internal/layout"
	"github.com/hoonfeng/goui/pkg/paint"
	"github.com/hoonfeng/goui/pkg/types"
)

// ─── TreeV2 虚拟树（el-tree-v2，复用 VirtualList）───────────────
// 展开的节点扁平化为可见行 → VirtualList 只渲染可视行(支撑大树)。
// 行 treeV2Row 自绘 缩进+箭头+label；点箭头展开/折叠、点行选中。

type TreeV2 struct {
	StatefulWidget
	Nodes    []*TreeNode
	Height   float64
	OnSelect func(*TreeNode)
}

func NewTreeV2(nodes ...*TreeNode) *TreeV2          { return &TreeV2{Nodes: nodes, Height: 300} }
func (t *TreeV2) WithHeight(h float64) *TreeV2      { t.Height = h; return t }
func (t *TreeV2) WithOnSelect(fn func(*TreeNode)) *TreeV2 { t.OnSelect = fn; return t }

func (t *TreeV2) CreateState() State { return &treeV2State{expanded: map[*TreeNode]bool{}} }

type treeV2State struct {
	BaseState
	expanded map[*TreeNode]bool
}

func (s *treeV2State) cfg() *TreeV2 {
	if w, ok := s.Widget().(*TreeV2); ok {
		return w
	}
	return &TreeV2{}
}

type flatNode struct {
	node  *TreeNode
	level int
}

func (s *treeV2State) flatten() []flatNode {
	var out []flatNode
	var walk func(nodes []*TreeNode, level int)
	walk = func(nodes []*TreeNode, level int) {
		for _, n := range nodes {
			out = append(out, flatNode{n, level})
			if s.expanded[n] && len(n.Children) > 0 {
				walk(n.Children, level+1)
			}
		}
	}
	walk(s.cfg().Nodes, 0)
	return out
}

func (s *treeV2State) Build(ctx BuildContext) Widget {
	flat := s.flatten()
	c := s.cfg()
	return NewVirtualList(len(flat), 30, func(i int) Widget {
		fn := flat[i]
		node := fn.node
		return &treeV2Row{
			level:    fn.level,
			hasArrow: len(node.Children) > 0,
			expanded: s.expanded[node],
			label:    node.Label,
			onToggle: func() { s.expanded[node] = !s.expanded[node]; s.SetState() },
			onSelect: func() {
				if c.OnSelect != nil {
					c.OnSelect(node)
				}
			},
		}
	}).WithHeight(c.Height)
}

// ─── treeV2Row 虚拟树一行 ───────────────────────────────────────

type treeV2Row struct {
	StatelessWidget
	level    int
	hasArrow bool
	expanded bool
	label    string
	onToggle func()
	onSelect func()
}

func (r *treeV2Row) CreateElement() Element {
	return &treeV2RowElement{BaseElement: BaseElement{widget: r}, row: r}
}

const treeV2RowH = 30.0
const treeV2Indent = 20.0

type treeV2RowElement struct {
	BaseElement
	row     *treeV2Row
	hovered bool
}

func (e *treeV2RowElement) Build() []Element { return nil }

func (e *treeV2RowElement) Layout(ctx *layout.LayoutContext) layout.LayoutResult {
	e.size = ctx.Constraints.Constrain(types.Size{Width: ctx.Constraints.MaxWidth, Height: treeV2RowH})
	return layout.LayoutResult{Size: e.size}
}

func (e *treeV2RowElement) Paint(cvs canvas.Canvas, offset types.Point) {
	pos := e.Offset()
	W := e.size.Width
	r := e.row
	if e.hovered {
		hp := paint.DefaultPaint()
		hp.Color = elFill()
		cvs.DrawRect(pos.X, pos.Y, W, treeV2RowH, hp)
	}
	indent := float64(r.level)*treeV2Indent + 10
	// 箭头 ▶/▼
	if r.hasArrow {
		ap := paint.DefaultStrokePaint()
		ap.Color = elTextSecondary()
		ap.StrokeWidth = 1.4
		ax, ay := pos.X+indent, pos.Y+treeV2RowH/2
		if r.expanded {
			cvs.DrawLine(ax-3, ay-2, ax+1, ay+2, ap)
			cvs.DrawLine(ax+1, ay+2, ax+5, ay-2, ap)
		} else {
			cvs.DrawLine(ax-1, ay-4, ax+3, ay, ap)
			cvs.DrawLine(ax+3, ay, ax-1, ay+4, ap)
		}
	}
	f := canvas.DefaultFont()
	f.Size = 14
	canvas.DrawTextAligned(cvs, r.label, types.Rect{X: pos.X + indent + 14, Y: pos.Y, Width: W - indent - 18, Height: treeV2RowH},
		f, elTextRegular(), canvas.HAlignLeft, canvas.VAlignMiddle)
}

func (e *treeV2RowElement) HandleEvent(ev event.Event) bool {
	me, ok := ev.(*event.MouseEvent)
	if !ok {
		return false
	}
	switch ev.Type() {
	case event.TypeMouseEnter, event.TypeMouseMove:
		if !e.hovered {
			e.hovered = true
			e.MarkNeedsPaint()
		}
		return true
	case event.TypeMouseLeave:
		e.hovered = false
		e.MarkNeedsPaint()
		return true
	case event.TypeMouseDown:
		lx := me.X - e.Offset().X
		indent := float64(e.row.level)*treeV2Indent + 10
		if e.row.hasArrow && lx < indent+12 {
			if e.row.onToggle != nil {
				e.row.onToggle()
			}
		} else if e.row.onSelect != nil {
			e.row.onSelect()
		}
		return true
	}
	return false
}

func (e *treeV2RowElement) Update(newWidget Widget) {
	if nr, ok := newWidget.(*treeV2Row); ok {
		e.row = nr
		e.BaseElement.widget = newWidget
		e.dirty = true
	}
}
