package animation

import (
	"testing"
	"time"

	"github.com/hoonfeng/goui/pkg/types"
)

// resetSched 清空全局调度器状态，避免测试间相互污染。
func resetSched() {
	schedMu.Lock()
	activeSet = map[ticker]struct{}{}
	haveLastTick = false
	frameCallback = nil
	schedMu.Unlock()
}

func almost(a, b float64) bool {
	d := a - b
	if d < 0 {
		d = -d
	}
	return d < 0.001
}

func TestControllerLinearProgress(t *testing.T) {
	resetSched()
	c := NewController(100*time.Millisecond, Linear)
	c.Start()
	if !c.IsRunning() || !HasActive() {
		t.Fatal("controller should be running and registered active")
	}

	base := time.Unix(100, 0)
	Tick(base)                            // 建立基准，无增量
	Tick(base.Add(50 * time.Millisecond)) // dt=50ms → t=0.5
	if !almost(c.Value(), 0.5) {
		t.Errorf("value=%.3f, want 0.5", c.Value())
	}

	Tick(base.Add(130 * time.Millisecond)) // elapsed=130ms>100 → done
	if c.IsRunning() {
		t.Error("controller should have finished")
	}
	if !almost(c.Value(), 1) {
		t.Errorf("final value=%.3f, want 1", c.Value())
	}
	if HasActive() {
		t.Error("no active animation should remain after done")
	}
}

func TestControllerRepeatStaysActive(t *testing.T) {
	resetSched()
	c := NewController(100*time.Millisecond, Linear)
	c.Repeat = true
	c.Start()

	base := time.Unix(200, 0)
	Tick(base)
	Tick(base.Add(150 * time.Millisecond)) // 150ms → wrap → 50ms → t≈0.5
	if !c.IsRunning() || !HasActive() {
		t.Fatal("repeat controller should keep running and active")
	}
	if !almost(c.Value(), 0.5) {
		t.Errorf("wrapped value=%.3f, want ~0.5", c.Value())
	}
}

func TestStopUnregisters(t *testing.T) {
	resetSched()
	c := NewController(time.Second, Linear)
	c.Start()
	if !HasActive() {
		t.Fatal("should be active after Start")
	}
	c.Stop()
	if c.IsRunning() || HasActive() {
		t.Error("Stop should halt and unregister the controller")
	}
}

func TestEasingEndpoints(t *testing.T) {
	for name, e := range map[string]Easing{
		"Linear": Linear, "EaseInQuad": EaseInQuad, "EaseOutQuad": EaseOutQuad,
		"EaseInOutQuad": EaseInOutQuad, "EaseInCubic": EaseInCubic,
		"EaseOutCubic": EaseOutCubic, "EaseInOutCubic": EaseInOutCubic,
		"EaseInSine": EaseInSine, "EaseOutSine": EaseOutSine, "EaseInOutSine": EaseInOutSine,
		"EaseInExpo": EaseInExpo, "EaseOutExpo": EaseOutExpo,
		"EaseInBack": EaseInBack, "EaseOutBack": EaseOutBack, "EaseInOutBack": EaseInOutBack,
		"EaseOutElastic": EaseOutElastic, "EaseOutBounce": EaseOutBounce, "EaseInBounce": EaseInBounce,
	} {
		if !almost(e(0), 0) {
			t.Errorf("%s(0)=%.3f, want 0", name, e(0))
		}
		if !almost(e(1), 1) {
			t.Errorf("%s(1)=%.3f, want 1", name, e(1))
		}
	}
}

func TestLerp(t *testing.T) {
	if got := LerpFloat(10, 20, 0.5); !almost(got, 15) {
		t.Errorf("LerpFloat=%.2f, want 15", got)
	}
	c := LerpColor(types.Color{R: 0, G: 0, B: 0, A: 255}, types.Color{R: 100, G: 200, B: 50, A: 255}, 0.5)
	if c.R != 50 || c.G != 100 || c.B != 25 {
		t.Errorf("LerpColor=%v, want R50 G100 B25", c)
	}
	p := LerpPoint(types.Point{X: 0, Y: 0}, types.Point{X: 10, Y: 20}, 0.5)
	if !almost(p.X, 5) || !almost(p.Y, 10) {
		t.Errorf("LerpPoint=%v, want (5,10)", p)
	}
}

func TestFrameCallbackFiresOnStart(t *testing.T) {
	resetSched()
	fired := 0
	SetFrameCallback(func() { fired++ })
	c := NewController(time.Second, Linear)
	c.Start()
	if fired == 0 {
		t.Error("frame callback should fire when an animation registers")
	}
}

func TestControllerStartDelay(t *testing.T) {
	resetSched()
	c := NewController(100*time.Millisecond, Linear)
	c.StartDelay = 50 * time.Millisecond
	c.Start()

	base := time.Unix(300, 0)
	Tick(base)
	Tick(base.Add(30 * time.Millisecond)) // 仍在延迟内（dt=30）
	if !almost(c.Value(), 0) {
		t.Errorf("during delay value=%.3f, want 0", c.Value())
	}
	// 用 ≤maxFrameDt(50ms) 的步进推进（避免单帧钳制干扰）：
	Tick(base.Add(70 * time.Millisecond)) // dt=40：消耗剩余 20ms 延迟 + 20ms 进度 → 0.2
	if !almost(c.Value(), 0.2) {
		t.Errorf("after delay value=%.3f, want 0.2", c.Value())
	}
	Tick(base.Add(110 * time.Millisecond)) // dt=40：再 40ms 进度 → 60ms → 0.6
	if !almost(c.Value(), 0.6) {
		t.Errorf("value=%.3f, want 0.6", c.Value())
	}
}

func TestSpringSettlesToTarget(t *testing.T) {
	resetSched()
	sp := &Spring{Stiffness: 180, Damping: 22, From: 0, To: 1}
	sp.Start()
	if !HasActive() || !sp.IsRunning() {
		t.Fatal("spring should be active after Start")
	}

	base := time.Unix(400, 0)
	Tick(base)
	for i := 1; i <= 240 && sp.IsRunning(); i++ {
		Tick(base.Add(time.Duration(i) * 16 * time.Millisecond))
	}
	if sp.IsRunning() {
		t.Errorf("spring should settle within budget, value=%.3f", sp.Value())
	}
	if !almost(sp.Value(), 1) {
		t.Errorf("settled value=%.3f, want 1", sp.Value())
	}
	if HasActive() {
		t.Error("no active animation should remain after spring settles")
	}
}
