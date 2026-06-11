package widget

import (
	"time"

	"github.com/hoonfeng/goui/pkg/animation"
	"github.com/hoonfeng/goui/pkg/types"
)

// AnimatedContainer 是隐式动画容器：当 Width/Height/Radius/Color 等配置变化时，
// 自动从旧值平滑过渡到新值，而非瞬间跳变。用法与 Container 类似——
// 每次父级重建时，只要目标值改变就会触发一段过渡动画。
//
//	widget.AnimatedContainer{
//	    Width: w, Height: h, Color: &col,
//	    Duration: 250 * time.Millisecond, Curve: animation.EaseInOutCubic,
//	    Child: ...,
//	}
type AnimatedContainer struct {
	StatefulWidget
	Width    float64
	Height   float64
	Radius   float64
	Color    *types.Color
	Duration time.Duration
	Curve    animation.Easing
	Child    Widget
}

// CreateState 创建隐式动画状态。
func (a *AnimatedContainer) CreateState() State {
	return &animatedContainerState{}
}

type animatedContainerState struct {
	BaseState
	ctrl *animation.Controller

	// 当前渲染值（动画插值结果）
	curW, curH, curR float64
	curColor         types.Color

	// 本次过渡的起点
	fromW, fromH, fromR float64
	fromColor           types.Color
}

func (s *animatedContainerState) cfg() *AnimatedContainer {
	if w, ok := s.Widget().(*AnimatedContainer); ok {
		return w
	}
	return &AnimatedContainer{}
}

// InitState 用初始配置作为当前渲染值（首帧不动画）。
func (s *animatedContainerState) InitState() {
	w := s.cfg()
	s.curW, s.curH, s.curR = w.Width, w.Height, w.Radius
	if w.Color != nil {
		s.curColor = *w.Color
	}
}

// DidUpdateWidget 在配置变化时，从当前渲染值平滑过渡到新目标值。
func (s *animatedContainerState) DidUpdateWidget(old Widget) {
	oldC, _ := old.(*AnimatedContainer)
	w := s.cfg()
	if oldC != nil &&
		oldC.Width == w.Width && oldC.Height == w.Height &&
		oldC.Radius == w.Radius && colorPtrEq(oldC.Color, w.Color) {
		return // 目标无变化，无需动画
	}

	// 以当前渲染值为过渡起点
	s.fromW, s.fromH, s.fromR = s.curW, s.curH, s.curR
	s.fromColor = s.curColor

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
		s.curW = animation.LerpFloat(s.fromW, w.Width, v)
		s.curH = animation.LerpFloat(s.fromH, w.Height, v)
		s.curR = animation.LerpFloat(s.fromR, w.Radius, v)
		if w.Color != nil {
			s.curColor = animation.LerpColor(s.fromColor, *w.Color, v)
		}
		s.SetState()
	}
	s.ctrl.Start()
}

// Build 用当前动画值构建一个 Container。
func (s *animatedContainerState) Build(ctx BuildContext) Widget {
	w := s.cfg()
	c := &Container{
		SingleChildWidget: SingleChildWidget{Child: w.Child},
		Width:             s.curW,
		Height:            s.curH,
		BorderRadius:      s.curR,
	}
	if w.Color != nil {
		col := s.curColor
		c.Background = &PaintWidget{Color: &col}
	}
	return c
}

func colorPtrEq(a, b *types.Color) bool {
	if a == nil || b == nil {
		return a == b
	}
	return *a == *b
}
