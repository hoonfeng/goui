package widget

import (
	"math"
	"reflect"
	"time"

	"github.com/hoonfeng/goui/pkg/animation"
	"github.com/hoonfeng/goui/pkg/canvas"
	"github.com/hoonfeng/goui/internal/layout"
	"github.com/hoonfeng/goui/pkg/types"
)

// Transform 对子组件应用 2D 变换（缩放/旋转/平移），变换围绕子组件中心。
// Rotation 单位为角度（度，与 Canvas.Rotate 一致）。Scale 为 0 时视为 1（不缩放）。
type Transform struct {
	Scale    float64 // 缩放倍数（0 视为 1）
	Rotation float64 // 旋转角度（度），围绕中心
	OffsetX  float64 // 平移 X
	OffsetY  float64 // 平移 Y
	Child    Widget
}

// CreateElement 创建 transformElement。
func (t *Transform) CreateElement() Element {
	return &transformElement{
		BaseElement: BaseElement{widget: t},
		transform:   t,
	}
}

type transformElement struct {
	BaseElement
	transform *Transform
	child     Element
}

func (e *transformElement) Update(newWidget Widget) {
	e.widget = newWidget
	e.dirty = true
	if t, ok := newWidget.(*Transform); ok {
		e.transform = t
	}
}

func (e *transformElement) Build() []Element {
	if e.transform.Child == nil {
		if e.child != nil {
			e.child.Unmount()
			e.child = nil
		}
		e.children = nil
		return nil
	}
	if e.child != nil && e.child.WidgetType() == reflect.TypeOf(e.transform.Child) {
		e.child.Update(e.transform.Child)
	} else {
		if e.child != nil {
			e.child.Unmount()
		}
		e.child = CreateElementFor(e.transform.Child)
		e.child.Mount(e, 0)
	}
	e.children = []Element{e.child}
	return e.children
}

func (e *transformElement) Layout(ctx *layout.LayoutContext) layout.LayoutResult {
	if e.child != nil {
		result := e.child.Layout(ctx)
		e.size = result.Size
		return result
	}
	e.size = ctx.Constraints.Constrain(types.Size{})
	return layout.LayoutResult{Size: e.size}
}

func (e *transformElement) Paint(cvs canvas.Canvas, offset types.Point) {
	if e.child == nil {
		return
	}
	t := e.transform
	scale := t.Scale
	if scale == 0 {
		scale = 1
	}
	// 无变换：直接绘制，避免不必要的矩阵开销
	if scale == 1 && t.Rotation == 0 && t.OffsetX == 0 && t.OffsetY == 0 {
		e.child.Paint(cvs, offset)
		return
	}
	pos := e.Offset()
	cx := pos.X + e.size.Width/2
	cy := pos.Y + e.size.Height/2

	cvs.Save()
	// 围绕中心变换：平移到中心 → 旋转/缩放 → 平移回原点
	cvs.Translate(cx+t.OffsetX, cy+t.OffsetY)
	if t.Rotation != 0 {
		cvs.Rotate(t.Rotation)
	}
	if scale != 1 {
		cvs.Scale(scale, scale)
	}
	cvs.Translate(-cx, -cy)
	e.child.Paint(cvs, offset)
	cvs.Restore()
}

// HitTestTransformer 表示元素对其子树施加了坐标变换；命中测试时需把父坐标系
// 中的点逆变换到子坐标系，使变换后的子元素也能被正确命中。Transform 实现它。
type HitTestTransformer interface {
	TransformHitPoint(x, y float64) (cx, cy float64)
}

// TransformHitPoint 把父坐标系中的点逆变换到子坐标系（与 Paint 的正变换互逆），
// 使旋转/缩放/平移后的子元素命中判定与视觉一致。
func (e *transformElement) TransformHitPoint(x, y float64) (float64, float64) {
	t := e.transform
	scale := t.Scale
	if scale == 0 {
		scale = 1
	}
	if scale == 1 && t.Rotation == 0 && t.OffsetX == 0 && t.OffsetY == 0 {
		return x, y
	}
	pos := e.Offset()
	cx := pos.X + e.size.Width/2
	cy := pos.Y + e.size.Height/2
	// 正变换：T(cx+offX, cy+offY)·R(θ)·S(s)·T(-cx,-cy)
	// 逆变换：减 (cx+offX, cy+offY) → 逆旋转 → 逆缩放 → 加 (cx, cy)
	px := x - (cx + t.OffsetX)
	py := y - (cy + t.OffsetY)
	if t.Rotation != 0 {
		rad := -t.Rotation * math.Pi / 180 // 逆旋转（度→弧度）
		cos := math.Cos(rad)
		sin := math.Sin(rad)
		px, py = px*cos-py*sin, px*sin+py*cos
	}
	if scale != 1 {
		px /= scale
		py /= scale
	}
	return px + cx, py + cy
}

// ─────────────────────────────────────────────
// AnimatedScale — 隐式缩放动画
// ─────────────────────────────────────────────

// AnimatedScale 在 Scale 变化时自动以动画过渡（围绕中心缩放）。
type AnimatedScale struct {
	StatefulWidget
	Scale    float64
	Duration time.Duration
	Curve    animation.Easing
	Child    Widget
}

func (a *AnimatedScale) CreateState() State { return &animatedScaleState{} }

type animatedScaleState struct {
	BaseState
	ctrl *animation.Controller
	cur  float64
	from float64
}

func (s *animatedScaleState) cfg() *AnimatedScale {
	if w, ok := s.Widget().(*AnimatedScale); ok {
		return w
	}
	return &AnimatedScale{Scale: 1}
}

func (s *animatedScaleState) InitState() { s.cur = s.cfg().Scale }

func (s *animatedScaleState) DidUpdateWidget(old Widget) {
	oldS, _ := old.(*AnimatedScale)
	w := s.cfg()
	if oldS != nil && oldS.Scale == w.Scale {
		return
	}
	s.from = s.cur
	dur := w.Duration
	if dur <= 0 {
		dur = 250 * time.Millisecond
	}
	curve := w.Curve
	if curve == nil {
		curve = animation.EaseOutBack
	}
	if s.ctrl != nil {
		s.ctrl.Stop()
	}
	s.ctrl = animation.NewController(dur, curve)
	s.ctrl.OnUpdate = func(v float64) {
		s.cur = animation.LerpFloat(s.from, w.Scale, v)
		s.SetState()
	}
	s.ctrl.Start()
}

func (s *animatedScaleState) Build(ctx BuildContext) Widget {
	return &Transform{Scale: s.cur, Child: s.cfg().Child}
}

// ─────────────────────────────────────────────
// AnimatedRotation — 隐式旋转动画
// ─────────────────────────────────────────────

// AnimatedRotation 在 Rotation（度）变化时自动以动画过渡（围绕中心旋转）。
type AnimatedRotation struct {
	StatefulWidget
	Rotation float64
	Duration time.Duration
	Curve    animation.Easing
	Child    Widget
}

func (a *AnimatedRotation) CreateState() State { return &animatedRotationState{} }

type animatedRotationState struct {
	BaseState
	ctrl *animation.Controller
	cur  float64
	from float64
}

func (s *animatedRotationState) cfg() *AnimatedRotation {
	if w, ok := s.Widget().(*AnimatedRotation); ok {
		return w
	}
	return &AnimatedRotation{}
}

func (s *animatedRotationState) InitState() { s.cur = s.cfg().Rotation }

func (s *animatedRotationState) DidUpdateWidget(old Widget) {
	oldR, _ := old.(*AnimatedRotation)
	w := s.cfg()
	if oldR != nil && oldR.Rotation == w.Rotation {
		return
	}
	s.from = s.cur
	dur := w.Duration
	if dur <= 0 {
		dur = 300 * time.Millisecond
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
		s.cur = animation.LerpFloat(s.from, w.Rotation, v)
		s.SetState()
	}
	s.ctrl.Start()
}

func (s *animatedRotationState) Build(ctx BuildContext) Widget {
	return &Transform{Rotation: s.cur, Child: s.cfg().Child}
}
