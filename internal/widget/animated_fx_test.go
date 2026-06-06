package widget

import (
	"testing"
	"time"

	"github.com/user/goui/internal/animation"
	"github.com/user/goui/internal/canvas"
	"github.com/user/goui/internal/layout"
	"github.com/user/goui/internal/types"
)

func TestAnimatedOpacityTween(t *testing.T) {
	animation.ResetScheduler()
	ao := &AnimatedOpacity{Opacity: 1, Duration: 100 * time.Millisecond, Curve: animation.Linear, Child: &Container{Width: 10, Height: 10}}
	el := CreateElementFor(ao).(*StatefulElement)
	el.Mount(nil, 0)
	el.Build()

	el.Update(&AnimatedOpacity{Opacity: 0, Duration: 100 * time.Millisecond, Curve: animation.Linear, Child: &Container{Width: 10, Height: 10}})
	st := el.state.(*animatedOpacityState)
	if st.ctrl == nil || !st.ctrl.IsRunning() {
		t.Fatal("AnimatedOpacity should animate on opacity change")
	}
	base := time.Unix(3000, 0)
	animation.Tick(base)
	animation.Tick(base.Add(50 * time.Millisecond))
	if st.cur < 0.4 || st.cur > 0.6 {
		t.Errorf("opacity mid=%.2f, want ~0.5", st.cur)
	}
}

func TestAnimatedScaleTween(t *testing.T) {
	animation.ResetScheduler()
	as := &AnimatedScale{Scale: 1, Duration: 100 * time.Millisecond, Curve: animation.Linear, Child: &Container{Width: 10, Height: 10}}
	el := CreateElementFor(as).(*StatefulElement)
	el.Mount(nil, 0)
	el.Build()

	el.Update(&AnimatedScale{Scale: 2, Duration: 100 * time.Millisecond, Curve: animation.Linear, Child: &Container{Width: 10, Height: 10}})
	st := el.state.(*animatedScaleState)
	if st.ctrl == nil || !st.ctrl.IsRunning() {
		t.Fatal("AnimatedScale should animate on scale change")
	}
	base := time.Unix(4000, 0)
	animation.Tick(base)
	animation.Tick(base.Add(50 * time.Millisecond))
	if st.cur < 1.4 || st.cur > 1.6 {
		t.Errorf("scale mid=%.2f, want ~1.5", st.cur)
	}
}

// TestOpacityAndTransformPaintNoPanic 透明度/变换组件渲染不应 panic（含全透明跳过、缩放变换）。
func TestOpacityAndTransformPaintNoPanic(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("paint panicked: %v", r)
		}
	}()
	c := canvas.NewSkiaCanvas(60, 60)

	for _, w := range []Widget{
		&Opacity{Opacity: 0, Child: &Container{Width: 20, Height: 20}},   // 全透明跳过
		&Opacity{Opacity: 0.5, Child: &Container{Width: 20, Height: 20}}, // 半透明组合成
		&Transform{Scale: 1.5, Rotation: 30, Child: &Container{Width: 20, Height: 20}},
	} {
		el := CreateElementFor(w)
		el.Mount(nil, 0)
		el.Build()
		el.Layout(&layout.LayoutContext{Constraints: layout.Tight(60, 60)})
		el.SetPosition(types.Point{X: 5, Y: 5})
		el.Paint(c, types.Point{})
	}
	if err := c.Flush(); err != nil {
		t.Fatal(err)
	}
}
