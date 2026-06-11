package widget

import (
	"testing"
	"time"

	"github.com/hoonfeng/goui/pkg/animation"
	"github.com/hoonfeng/goui/pkg/types"
)

// TestAnimatedContainerImplicitTween 验证隐式动画：配置(宽度/颜色)变化后，
// AnimatedContainer 从旧值平滑过渡到新值，而非瞬间跳变。
func TestAnimatedContainerImplicitTween(t *testing.T) {
	animation.ResetScheduler()

	col := types.ColorFromRGB(0, 0, 0)
	ac := &AnimatedContainer{
		Width: 100, Height: 50, Color: &col,
		Duration: 100 * time.Millisecond, Curve: animation.Linear,
	}
	el := CreateElementFor(ac).(*StatefulElement)
	el.Mount(nil, 0)
	el.Build()

	st := el.state.(*animatedContainerState)
	if st.curW != 100 {
		t.Fatalf("initial curW=%.1f, want 100", st.curW)
	}

	// 父级以新目标重建：宽度 100→200，颜色黑→红
	col2 := types.ColorFromRGB(255, 0, 0)
	el.Update(&AnimatedContainer{
		Width: 200, Height: 50, Color: &col2,
		Duration: 100 * time.Millisecond, Curve: animation.Linear,
	})
	if st.ctrl == nil || !st.ctrl.IsRunning() {
		t.Fatal("update with a new target should start an animation")
	}

	base := time.Unix(500, 0)
	animation.Tick(base)
	animation.Tick(base.Add(50 * time.Millisecond)) // t=0.5

	if st.curW < 140 || st.curW > 160 {
		t.Errorf("mid-tween curW=%.1f, want ~150", st.curW)
	}
	if st.curColor.R < 100 || st.curColor.R > 160 {
		t.Errorf("mid-tween color.R=%d, want ~128", st.curColor.R)
	}

	// 推进到结束：应精确落到目标值并停止
	animation.Tick(base.Add(120 * time.Millisecond))
	if st.curW != 200 {
		t.Errorf("final curW=%.1f, want 200", st.curW)
	}
	if st.curColor.R != 255 {
		t.Errorf("final color.R=%d, want 255", st.curColor.R)
	}
	if st.ctrl.IsRunning() {
		t.Error("animation should be finished")
	}
}

// TestAnimatedContainerNoTweenWhenUnchanged 目标未变化时不应启动动画。
func TestAnimatedContainerNoTweenWhenUnchanged(t *testing.T) {
	animation.ResetScheduler()

	ac := &AnimatedContainer{Width: 80, Height: 30, Duration: 100 * time.Millisecond}
	el := CreateElementFor(ac).(*StatefulElement)
	el.Mount(nil, 0)
	el.Build()

	el.Update(&AnimatedContainer{Width: 80, Height: 30, Duration: 100 * time.Millisecond})
	st := el.state.(*animatedContainerState)
	if st.ctrl != nil && st.ctrl.IsRunning() {
		t.Error("unchanged target should not start an animation")
	}
	if animation.HasActive() {
		t.Error("no active animation expected for unchanged target")
	}
}

// ── 诊断：完整重建链路是否把状态变化传播到深层 AnimatedContainer ──

type testAnimRoot struct {
	StatefulWidget
	st *testAnimRootState
}

func (r *testAnimRoot) CreateState() State {
	r.st = &testAnimRootState{}
	return r.st
}

type testAnimRootState struct {
	BaseState
	expanded bool
}

func (s *testAnimRootState) Build(ctx BuildContext) Widget {
	w := 100.0
	if s.expanded {
		w = 200.0
	}
	return NewColumn([]Widget{
		&AnimatedContainer{Width: w, Height: 50, Duration: 100 * time.Millisecond, Curve: animation.Linear},
	})
}

func buildTreeRec(el Element) {
	for _, c := range el.Build() {
		buildTreeRec(c)
	}
}

func findACState(el Element) *animatedContainerState {
	if se, ok := el.(*StatefulElement); ok {
		if acs, ok := se.state.(*animatedContainerState); ok {
			return acs
		}
	}
	for _, c := range el.Children() {
		if r := findACState(c); r != nil {
			return r
		}
	}
	return nil
}

// TestRebuildPropagatesToNestedAnimated 验证 Rebuild + buildTree 能把状态变化
// 传播到嵌套的 AnimatedContainer（触发其 Update→DidUpdateWidget→动画）。
func TestRebuildPropagatesToNestedAnimated(t *testing.T) {
	animation.ResetScheduler()
	root := &testAnimRoot{}
	rootEl := CreateElementFor(root).(*StatefulElement)
	rootEl.Mount(nil, 0)
	buildTreeRec(rootEl)

	ac := findACState(rootEl)
	if ac == nil {
		t.Fatal("AnimatedContainer state not found in tree")
	}
	if ac.curW != 100 {
		t.Fatalf("initial curW=%.0f, want 100", ac.curW)
	}

	// 模拟点击：改状态 + Rebuild（SetState 所做）+ buildTree（relayout 所做）
	root.st.expanded = true
	rootEl.Rebuild()
	buildTreeRec(rootEl)

	if ac.ctrl == nil || !ac.ctrl.IsRunning() {
		t.Fatal("nested AnimatedContainer animation did not start after rebuild+buildTree")
	}
}

// TestSetStateToAnimationEndToEnd 端到端模拟运行时：SetState 请求 relayout，
// relayout(buildTree) 把新目标传到深层 AnimatedContainer 并启动动画，Tick 推进插值。
func TestSetStateToAnimationEndToEnd(t *testing.T) {
	animation.ResetScheduler()

	// 模拟 app 把 OnNeedsLayout 接到 Pipeline.MarkNeedsLayout
	relayouts := 0
	OnNeedsLayout = func() { relayouts++ }
	defer func() { OnNeedsLayout = nil }()

	root := &testAnimRoot{}
	rootEl := CreateElementFor(root).(*StatefulElement)
	rootEl.Mount(nil, 0)
	buildTreeRec(rootEl)
	ac := findACState(rootEl)
	if ac == nil {
		t.Fatal("AnimatedContainer not found")
	}

	// 模拟点击：改状态并 SetState（应请求 relayout）
	root.st.expanded = true
	root.st.SetState()
	if relayouts == 0 {
		t.Fatal("SetState should request a relayout (OnNeedsLayout)")
	}

	// 模拟 app 在收到 relayout 请求后执行 buildTree
	buildTreeRec(rootEl)
	if ac.ctrl == nil || !ac.ctrl.IsRunning() {
		t.Fatal("animation should start after SetState + relayout")
	}

	// 模拟动画帧推进到中途
	base := time.Unix(1000, 0)
	animation.Tick(base)
	animation.Tick(base.Add(50 * time.Millisecond)) // 半程
	if ac.curW <= 100 || ac.curW >= 200 {
		t.Errorf("mid-animation curW=%.1f, want strictly between 100 and 200", ac.curW)
	}
}
