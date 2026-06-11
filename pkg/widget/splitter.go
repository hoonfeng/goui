package widget

import (
	"reflect"

	"github.com/hoonfeng/goui/pkg/canvas"
	"github.com/hoonfeng/goui/pkg/event"
	"github.com/hoonfeng/goui/internal/layout"
	"github.com/hoonfeng/goui/pkg/paint"
	"github.com/hoonfeng/goui/pkg/types"
)

// ─── Splitter 分隔面板（el-splitter，自绘拖动）──────────────────
// 左右两面板 + 中间可拖动分隔条；拖动改变分配比例(relayout)。嵌任意 Left/Right。

type Splitter struct {
	StatelessWidget
	Left   Widget
	Right  Widget
	Height float64
}

func NewSplitter(left, right Widget) *Splitter { return &Splitter{Left: left, Right: right, Height: 240} }
func (s *Splitter) WithHeight(h float64) *Splitter { s.Height = h; return s }

func (s *Splitter) CreateElement() Element {
	return &SplitterElement{BaseElement: BaseElement{widget: s}, splitter: s, ratio: 0.5}
}

const splitterBar = 8.0

type SplitterElement struct {
	BaseElement
	splitter        *Splitter
	leftEl, rightEl Element
	ratio           float64
	dragging        bool
}

func (e *SplitterElement) reconcile(old Element, w Widget, idx int) Element {
	if old != nil && reflect.TypeOf(old.Widget()) == reflect.TypeOf(w) {
		old.Update(w)
		return old
	}
	if old != nil {
		old.Unmount()
	}
	el := CreateElementFor(w)
	el.Mount(e, idx)
	return el
}

func (e *SplitterElement) Build() []Element {
	e.leftEl = e.reconcile(e.leftEl, e.splitter.Left, 0)
	e.rightEl = e.reconcile(e.rightEl, e.splitter.Right, 1)
	e.children = []Element{e.leftEl, e.rightEl}
	return e.children
}

func (e *SplitterElement) Layout(ctx *layout.LayoutContext) layout.LayoutResult {
	w := ctx.Constraints.MaxWidth
	if w >= float64(1<<30) {
		w = 400
	}
	h := e.splitter.Height
	leftW := (w - splitterBar) * e.ratio
	rightW := w - splitterBar - leftW
	tight := func(el Element, cw, x float64) {
		el.Layout(&layout.LayoutContext{Constraints: layout.BoxConstraints{MinWidth: cw, MaxWidth: cw, MinHeight: h, MaxHeight: h}})
		el.SetPosition(types.Point{X: x, Y: 0})
	}
	tight(e.leftEl, leftW, 0)
	tight(e.rightEl, rightW, leftW+splitterBar)
	e.size = ctx.Constraints.Constrain(types.Size{Width: w, Height: h})
	return layout.LayoutResult{Size: e.size}
}

func (e *SplitterElement) Paint(cvs canvas.Canvas, offset types.Point) {
	pos := e.Offset()
	e.leftEl.Paint(cvs, offset)
	e.rightEl.Paint(cvs, offset)

	// 分隔条
	barX := pos.X + (e.size.Width-splitterBar)*e.ratio
	bp := paint.DefaultPaint()
	bp.Color = types.ColorFromRGB(240, 242, 245)
	if e.dragging {
		bp.Color = elPrimary()
	}
	cvs.DrawRect(barX, pos.Y, splitterBar, e.size.Height, bp)
	// 手柄（中间三条短竖线）
	hp := paint.DefaultStrokePaint()
	hp.Color = elPlaceholder()
	hp.StrokeWidth = 1
	cy := pos.Y + e.size.Height/2
	for i := -1; i <= 1; i++ {
		x := barX + splitterBar/2 + float64(i)*2.5
		cvs.DrawLine(x, cy-6, x, cy+6, hp)
	}
}

func (e *SplitterElement) HandleEvent(ev event.Event) bool {
	me, ok := ev.(*event.MouseEvent)
	if !ok {
		return false
	}
	lx := me.X - e.Offset().X
	barX := (e.size.Width - splitterBar) * e.ratio
	switch ev.Type() {
	case event.TypeMouseDown:
		if lx >= barX-4 && lx <= barX+splitterBar+4 {
			e.dragging = true
			if RequestPointerCapture != nil {
				RequestPointerCapture(e)
			}
			return true
		}
	case event.TypeMouseMove:
		if e.dragging {
			e.ratio = clamp(lx/(e.size.Width-splitterBar), 0.1, 0.9)
			if OnNeedsLayout != nil {
				OnNeedsLayout()
			}
			return true
		}
	case event.TypeMouseUp:
		// 仅 MouseUp 结束拖动（MouseDown 已 RequestPointerCapture，move/up 必达本元素）。
		// 不在 MouseLeave 结束——拖动中鼠标移出分隔条会触发 MouseLeave，据此清 dragging
		// 会让拖动半途中断。
		if e.dragging {
			e.dragging = false
			if OnNeedsRepaint != nil {
				OnNeedsRepaint()
			}
			return true
		}
	}
	return false
}

func (e *SplitterElement) Update(newWidget Widget) {
	if ns, ok := newWidget.(*Splitter); ok {
		e.splitter = ns
		e.BaseElement.widget = newWidget
		e.dirty = true
	}
}
