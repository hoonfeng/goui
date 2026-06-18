package widget

import (
	"reflect"
	"time"

	"github.com/hoonfeng/goui/pkg/animation"
	"github.com/hoonfeng/goui/pkg/canvas"
	"github.com/hoonfeng/goui/internal/layout"
	"github.com/hoonfeng/goui/pkg/types"
)

// Opacity 以指定整体透明度（0~1）绘制其子组件（组透明度）。
type Opacity struct {
	Opacity float64
	Child   Widget
}

// CreateElement 创建 opacityElement。
func (o *Opacity) CreateElement() Element {
	return &opacityElement{
		BaseElement: BaseElement{widget: o},
		opacity:     o,
	}
}

type opacityElement struct {
	BaseElement
	opacity *Opacity
	child   Element
}

func (e *opacityElement) Update(newWidget Widget) {
	e.widget = newWidget
	e.dirty = true
	if o, ok := newWidget.(*Opacity); ok {
		e.opacity = o
	}
}

func (e *opacityElement) Build() []Element {
	if e.opacity.Child == nil {
		if e.child != nil {
			e.child.Unmount()
			e.child = nil
		}
		e.children = nil
		return nil
	}
	if e.child != nil && e.child.WidgetType() == reflect.TypeOf(e.opacity.Child) {
		e.child.Update(e.opacity.Child)
	} else {
		if e.child != nil {
			e.child.Unmount()
		}
		e.child = CreateElementFor(e.opacity.Child)
		e.child.Mount(e, 0)
	}
	e.children = []Element{e.child}
	return e.children
}

func (e *opacityElement) Layout(ctx *layout.LayoutContext) layout.LayoutResult {
	if e.child != nil {
		result := e.child.Layout(ctx)
		e.size = result.Size
		return result
	}
	e.size = ctx.Constraints.Constrain(types.Size{})
	return layout.LayoutResult{Size: e.size}
}

func (e *opacityElement) Paint(cvs canvas.Canvas, offset types.Point) {
	if e.child == nil {
		return
	}
	a := e.opacity.Opacity
	switch {
	case a >= 1:
		e.child.Paint(cvs, offset)
	case a <= 0:
		// 完全透明：不绘制
	default:
		cvs.SaveLayerAlpha(a)
		e.child.Paint(cvs, offset)
		cvs.Restore()
	}
}

// ─────────────────────────────────────────────
// AnimatedOpacity — 隐式淡入淡出
// ─────────────────────────────────────────────

// AnimatedOpacity 在 Opacity 变化时自动以动画淡入/淡出。
type AnimatedOpacity struct {
	StatefulWidget
	Opacity  float64
	Duration time.Duration
	Curve    animation.Easing
	Child    Widget
}

// CreateState 创建隐式淡入淡出状态。
func (a *AnimatedOpacity) CreateState() State { return &animatedOpacityState{} }

type animatedOpacityState struct {
	BaseState
	ctrl *animation.Controller
	cur  float64
	from float64
}

func (s *animatedOpacityState) cfg() *AnimatedOpacity {
	if w, ok := s.Widget().(*AnimatedOpacity); ok {
		return w
	}
	return &AnimatedOpacity{Opacity: 1}
}

func (s *animatedOpacityState) InitState() { s.cur = s.cfg().Opacity }

func (s *animatedOpacityState) DidUpdateWidget(old Widget) {
	oldO, _ := old.(*AnimatedOpacity)
	w := s.cfg()
	if oldO != nil && oldO.Opacity == w.Opacity {
		return
	}
	s.from = s.cur
	dur := w.Duration
	if dur <= 0 {
		dur = 250 * time.Millisecond
	}
	curve := w.Curve
	if curve == nil {
		curve = animation.EaseInOutCubic
	}
	if s.ctrl != nil {
		s.ctrl.Stop()
	}
	s.ctrl = animation.NewController(dur, curve)
	s.ctrl.OnUpdate = func(v float64) {
		s.cur = animation.LerpFloat(s.from, w.Opacity, v)
		s.SetState()
	}
	s.ctrl.Start()
}

func (s *animatedOpacityState) Build(ctx BuildContext) Widget {
	w := s.cfg()
	return &Opacity{Opacity: s.cur, Child: w.Child}
}
