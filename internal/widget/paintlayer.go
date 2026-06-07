package widget

import (
	"github.com/user/goui/internal/canvas"
	"github.com/user/goui/internal/event"
	"github.com/user/goui/internal/layout"
	"github.com/user/goui/internal/types"
)

// PaintLayer 自绘叠层：铺满父约束，在 OnPaint 里自由绘制（读外部状态）。事件透传（不拦截）。
// 用途：拖拽影子/高亮等「每帧只变绘制、不变结构」的叠层——配合 OnNeedsRepaint 仅重绘、
// 不触发 SetState 重建/重布局，于是拖动跟手不卡（重建整棵树才是卡顿源）。
type PaintLayer struct {
	StatelessWidget
	OnPaint func(cvs canvas.Canvas, x, y, w, h float64)
}

func (w *PaintLayer) CreateElement() Element {
	return &paintLayerElement{BaseElement: BaseElement{widget: w}, w: w}
}

type paintLayerElement struct {
	BaseElement
	w *PaintLayer
}

func (e *paintLayerElement) Build() []Element { return nil }

func (e *paintLayerElement) Layout(ctx *layout.LayoutContext) layout.LayoutResult {
	e.size = ctx.Constraints.Constrain(types.Size{Width: ctx.Constraints.MaxWidth, Height: ctx.Constraints.MaxHeight})
	return layout.LayoutResult{Size: e.size}
}

func (e *paintLayerElement) Paint(cvs canvas.Canvas, offset types.Point) {
	if e.w.OnPaint != nil {
		pos := e.Offset()
		e.w.OnPaint(cvs, pos.X, pos.Y, e.size.Width, e.size.Height)
	}
}

func (e *paintLayerElement) HandleEvent(ev event.Event) bool { return false } // 事件透传

func (e *paintLayerElement) Update(nw Widget) {
	if w, ok := nw.(*PaintLayer); ok {
		e.w = w
	}
	e.widget = nw
	e.dirty = true
}
