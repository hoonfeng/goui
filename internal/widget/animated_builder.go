package widget

import "github.com/user/goui/internal/animation"

// AnimatedBuilder 是通用动画组件：给定一个动画 Controller 和 Builder 函数，
// 每帧用 controller 的当前值重建子树。借助它几乎可以构建任意动画——
// 尺寸、颜色、缩放、旋转、平移、透明度、甚至自定义绘制——是最灵活的动画方式。
//
// 注意：AnimatedBuilder 会接管 Controller 的 OnUpdate（用于触发重建），
// 因此动画逻辑应写在 Builder 内，而非 Controller.OnUpdate。
//
//	ctrl := animation.NewController(time.Second, animation.EaseOutBack)
//	ctrl.AutoReverse = true
//	ctrl.Start()
//	ui := &widget.AnimatedBuilder{
//	    Controller: ctrl,
//	    Builder: func(v float64) widget.Widget {
//	        s := animation.LerpFloat(40, 120, v)
//	        return &widget.Container{Width: s, Height: s}
//	    },
//	}
type AnimatedBuilder struct {
	StatefulWidget
	Controller *animation.Controller
	Builder    func(value float64) Widget
}

// CreateState 创建动画构建器状态。
func (a *AnimatedBuilder) CreateState() State { return &animatedBuilderState{} }

type animatedBuilderState struct {
	BaseState
}

func (s *animatedBuilderState) cfg() *AnimatedBuilder {
	if w, ok := s.Widget().(*AnimatedBuilder); ok {
		return w
	}
	return nil
}

func (s *animatedBuilderState) InitState()                 { s.hook() }
func (s *animatedBuilderState) DidUpdateWidget(old Widget) { s.hook() }

// hook 让 controller 每帧推进时触发本组件重建（用最新值）。
func (s *animatedBuilderState) hook() {
	w := s.cfg()
	if w == nil || w.Controller == nil {
		return
	}
	w.Controller.OnUpdate = func(v float64) { s.SetState() }
}

// Build 用 controller 的当前值调用 Builder 生成子树。
func (s *animatedBuilderState) Build(ctx BuildContext) Widget {
	w := s.cfg()
	if w == nil || w.Builder == nil {
		return &Container{}
	}
	v := 0.0
	if w.Controller != nil {
		v = w.Controller.Value()
	}
	if child := w.Builder(v); child != nil {
		return child
	}
	return &Container{}
}
