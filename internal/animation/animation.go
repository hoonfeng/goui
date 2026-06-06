// Package animation 提供 goui 的动画引擎：缓动曲线、数值/颜色插值、
// 动画控制器（Controller）以及驱动逐帧推进的全局调度器（Scheduler）。
//
// 设计要点：
//   - Controller 驱动一个 0→1 的归一化进度值，支持时长、缓动曲线、循环、往返；
//   - 全局 Scheduler 维护"活跃动画"集合，宿主主循环每帧调用 Tick(now) 推进，
//     并通过 HasActive() 决定是否需要持续渲染（而非阻塞等待消息）；
//   - 动画注册时通过 frameCallback 通知宿主"需要开始连续出帧"。
//
// 典型用法：
//
//	c := animation.NewController(300*time.Millisecond, animation.EaseInOutCubic)
//	c.OnUpdate = func(v float64) { width = animation.LerpFloat(100, 200, v); rebuild() }
//	c.Start()
package animation

import (
	"math"
	"sync"
	"time"

	"github.com/user/goui/internal/types"
)

// ─────────────────────────────────────────────
// 缓动曲线
// ─────────────────────────────────────────────

// Easing 是缓动函数：输入归一化时间 t∈[0,1]，返回缓动后的进度∈[0,1]。
type Easing func(t float64) float64

func Linear(t float64) float64 { return t }

func EaseInQuad(t float64) float64  { return t * t }
func EaseOutQuad(t float64) float64 { return t * (2 - t) }
func EaseInOutQuad(t float64) float64 {
	if t < 0.5 {
		return 2 * t * t
	}
	return -1 + (4-2*t)*t
}

func EaseInCubic(t float64) float64  { return t * t * t }
func EaseOutCubic(t float64) float64 { u := t - 1; return u*u*u + 1 }
func EaseInOutCubic(t float64) float64 {
	if t < 0.5 {
		return 4 * t * t * t
	}
	u := 2*t - 2
	return 0.5*u*u*u + 1
}

// ── 正弦 ──
func EaseInSine(t float64) float64    { return 1 - math.Cos(t*math.Pi/2) }
func EaseOutSine(t float64) float64   { return math.Sin(t * math.Pi / 2) }
func EaseInOutSine(t float64) float64 { return -(math.Cos(math.Pi*t) - 1) / 2 }

// ── 指数 ──
func EaseInExpo(t float64) float64 {
	if t == 0 {
		return 0
	}
	return math.Pow(2, 10*t-10)
}
func EaseOutExpo(t float64) float64 {
	if t == 1 {
		return 1
	}
	return 1 - math.Pow(2, -10*t)
}

// ── 回弹（Back：越过终点再回弹，适合"弹出"效果）──
func EaseInBack(t float64) float64 {
	const c1 = 1.70158
	const c3 = c1 + 1
	return c3*t*t*t - c1*t*t
}
func EaseOutBack(t float64) float64 {
	const c1 = 1.70158
	const c3 = c1 + 1
	u := t - 1
	return 1 + c3*u*u*u + c1*u*u
}
func EaseInOutBack(t float64) float64 {
	const c1 = 1.70158
	const c2 = c1 * 1.525
	if t < 0.5 {
		return (math.Pow(2*t, 2) * ((c2+1)*2*t - c2)) / 2
	}
	return (math.Pow(2*t-2, 2)*((c2+1)*(2*t-2)+c2) + 2) / 2
}

// ── 弹性（Elastic：带振荡的弹簧感）──
func EaseOutElastic(t float64) float64 {
	if t == 0 || t == 1 {
		return t
	}
	const c4 = (2 * math.Pi) / 3
	return math.Pow(2, -10*t)*math.Sin((t*10-0.75)*c4) + 1
}

// ── 弹跳（Bounce：落地回弹）──
func EaseOutBounce(t float64) float64 {
	const n1 = 7.5625
	const d1 = 2.75
	switch {
	case t < 1/d1:
		return n1 * t * t
	case t < 2/d1:
		t -= 1.5 / d1
		return n1*t*t + 0.75
	case t < 2.5/d1:
		t -= 2.25 / d1
		return n1*t*t + 0.9375
	default:
		t -= 2.625 / d1
		return n1*t*t + 0.984375
	}
}
func EaseInBounce(t float64) float64 { return 1 - EaseOutBounce(1-t) }

// applyCurve 安全地应用缓动曲线（nil 视为线性），并把输入夹到 [0,1]。
func applyCurve(c Easing, t float64) float64 {
	if t < 0 {
		t = 0
	} else if t > 1 {
		t = 1
	}
	if c == nil {
		return t
	}
	return c(t)
}

// ─────────────────────────────────────────────
// 插值（Tween）
// ─────────────────────────────────────────────

// LerpFloat 在 a、b 之间按 t∈[0,1] 线性插值。
func LerpFloat(a, b, t float64) float64 { return a + (b-a)*t }

func lerpU8(a, b uint8, t float64) uint8 {
	v := float64(a) + (float64(b)-float64(a))*t
	if v < 0 {
		v = 0
	} else if v > 255 {
		v = 255
	}
	return uint8(v + 0.5)
}

// LerpColor 在两个颜色之间按 t 插值（逐通道，含 alpha）。
func LerpColor(a, b types.Color, t float64) types.Color {
	return types.Color{
		R: lerpU8(a.R, b.R, t),
		G: lerpU8(a.G, b.G, t),
		B: lerpU8(a.B, b.B, t),
		A: lerpU8(a.A, b.A, t),
	}
}

// LerpPoint 在两个点之间按 t 插值。
func LerpPoint(a, b types.Point, t float64) types.Point {
	return types.Point{X: LerpFloat(a.X, b.X, t), Y: LerpFloat(a.Y, b.Y, t)}
}

// LerpRect 在两个矩形之间按 t 插值。
func LerpRect(a, b types.Rect, t float64) types.Rect {
	return types.Rect{
		X:      LerpFloat(a.X, b.X, t),
		Y:      LerpFloat(a.Y, b.Y, t),
		Width:  LerpFloat(a.Width, b.Width, t),
		Height: LerpFloat(a.Height, b.Height, t),
	}
}

// ─────────────────────────────────────────────
// 动画控制器
// ─────────────────────────────────────────────

// Controller 驱动一个 0→1 的归一化动画进度值。
type Controller struct {
	Duration    time.Duration // 单程时长
	Curve       Easing        // 缓动曲线（nil=线性）
	Repeat      bool          // 到达终点后从头循环
	AutoReverse bool          // 到达终点后反向播放（往返/yoyo，隐含循环）
	StartDelay  time.Duration // 启动后到真正开始推进之间的延迟
	OnUpdate    func(value float64)
	OnDone      func()

	value     float64
	elapsed   time.Duration
	delayLeft time.Duration
	forward   bool
	running   bool
}

// NewController 创建一个动画控制器。
func NewController(d time.Duration, curve Easing) *Controller {
	return &Controller{Duration: d, Curve: curve, forward: true}
}

// Start 启动（或重启）动画，并注册到全局调度器。
func (c *Controller) Start() {
	c.elapsed = 0
	c.delayLeft = c.StartDelay
	c.forward = true
	c.running = true
	c.value = applyCurve(c.Curve, 0)
	register(c)
}

// Stop 停止动画并从调度器注销（不触发 OnDone）。
func (c *Controller) Stop() {
	if !c.running {
		return
	}
	c.running = false
	unregister(c)
}

// Value 返回当前缓动后的进度值（0→1）。
func (c *Controller) Value() float64 { return c.value }

// IsRunning 返回动画是否在运行。
func (c *Controller) IsRunning() bool { return c.running }

// update 按时间增量推进动画，由 Scheduler.Tick 调用。
func (c *Controller) update(dt time.Duration) {
	if !c.running {
		return
	}
	// 先消耗起始延迟，延迟期间保持在起点不推进
	if c.delayLeft > 0 {
		if dt < c.delayLeft {
			c.delayLeft -= dt
			return
		}
		dt -= c.delayLeft
		c.delayLeft = 0
	}
	c.elapsed += dt

	done := false
	if c.Duration > 0 {
		// 用减法处理跨越终点（一帧 dt 很大时可能跨多个周期），保留余数。
		for c.elapsed >= c.Duration {
			switch {
			case c.AutoReverse:
				c.elapsed -= c.Duration
				c.forward = !c.forward
			case c.Repeat:
				c.elapsed -= c.Duration
			default:
				c.elapsed = c.Duration
				done = true
			}
			if done {
				break
			}
		}
	} else {
		done = true // 零时长：立即完成
	}

	prog := 1.0
	if c.Duration > 0 {
		prog = float64(c.elapsed) / float64(c.Duration)
		if prog > 1 {
			prog = 1
		}
	}
	raw := prog
	if !c.forward {
		raw = 1 - prog
	}
	c.value = applyCurve(c.Curve, raw)

	if c.OnUpdate != nil {
		c.OnUpdate(c.value)
	}
	if done {
		c.running = false
		unregister(c)
		if c.OnDone != nil {
			c.OnDone()
		}
	}
}

// advance 实现 ticker 接口：按时间增量推进动画。
func (c *Controller) advance(dt time.Duration) { c.update(dt) }

// isActive 实现 ticker 接口：返回动画是否在运行。
func (c *Controller) isActive() bool { return c.running }

// ─────────────────────────────────────────────
// 全局调度器
// ─────────────────────────────────────────────

// ticker 是可被调度器逐帧推进的动画单元（Controller、Spring 等均实现它）。
type ticker interface {
	advance(dt time.Duration)
	isActive() bool
}

var (
	schedMu       sync.Mutex
	activeSet     = map[ticker]struct{}{}
	lastTick      time.Time
	haveLastTick  bool
	frameCallback func()
)

// register 把动画单元加入活跃集合，并通知宿主需要开始连续出帧。
func register(t ticker) {
	schedMu.Lock()
	activeSet[t] = struct{}{}
	cb := frameCallback
	schedMu.Unlock()
	if cb != nil {
		cb()
	}
}

func unregister(t ticker) {
	schedMu.Lock()
	delete(activeSet, t)
	schedMu.Unlock()
}

// HasActive 返回当前是否存在活跃动画。宿主主循环据此决定是否持续渲染。
func HasActive() bool {
	schedMu.Lock()
	defer schedMu.Unlock()
	return len(activeSet) > 0
}

// SetFrameCallback 注册"需要连续出帧"的回调（通常触发一次重绘标记）。
// 由宿主（app）在初始化时设置。
func SetFrameCallback(fn func()) {
	schedMu.Lock()
	frameCallback = fn
	schedMu.Unlock()
}

// ResetScheduler 停止所有动画并重置时间基准（主要用于测试或重新初始化）。
func ResetScheduler() {
	schedMu.Lock()
	activeSet = map[ticker]struct{}{}
	haveLastTick = false
	schedMu.Unlock()
}

// Tick 按当前时间推进所有活跃动画。宿主主循环每帧调用一次。
// 首次调用仅记录基准时间，不产生时间增量。
func Tick(now time.Time) {
	schedMu.Lock()
	if !haveLastTick {
		lastTick = now
		haveLastTick = true
		schedMu.Unlock()
		return
	}
	dt := now.Sub(lastTick)
	lastTick = now
	if dt < 0 {
		dt = 0
	}
	// 拷贝活跃集合，避免回调中修改集合引发并发问题
	list := make([]ticker, 0, len(activeSet))
	for t := range activeSet {
		list = append(list, t)
	}
	schedMu.Unlock()

	for _, t := range list {
		t.advance(dt)
	}
}

// ─────────────────────────────────────────────
// 物理弹簧
// ─────────────────────────────────────────────

// Spring 是基于弹簧物理（刚度/阻尼/质量）的动画：从 From 运动到 To，
// 不依赖固定时长，由物理收敛决定何时结束。适合自然回弹、跟手抛掷等效果。
//
//	sp := &animation.Spring{Stiffness: 180, Damping: 12, From: 0, To: 1}
//	sp.OnUpdate = func(v float64) { x = animation.LerpFloat(a, b, v); rebuild() }
//	sp.Start()
type Spring struct {
	Stiffness float64 // 刚度（越大越快越硬），典型 100~300
	Damping   float64 // 阻尼（越大越快停止、越小回弹越多），典型 8~30
	Mass      float64 // 质量（<=0 视为 1）
	From, To  float64 // 起点、终点
	Velocity  float64 // 初速度（可用于跟手抛掷）
	OnUpdate  func(value float64)
	OnDone    func()

	pos, vel float64
	elapsed  time.Duration
	running  bool
}

// Start 启动弹簧并注册到调度器。
func (s *Spring) Start() {
	s.pos = s.From
	s.vel = s.Velocity
	s.elapsed = 0
	s.running = true
	if s.OnUpdate != nil {
		s.OnUpdate(s.pos)
	}
	register(s)
}

// Stop 停止弹簧并注销（不触发 OnDone）。
func (s *Spring) Stop() {
	if !s.running {
		return
	}
	s.running = false
	unregister(s)
}

// Value 返回当前位置。
func (s *Spring) Value() float64 { return s.pos }

// IsRunning 返回是否在运行。
func (s *Spring) IsRunning() bool { return s.running }

func (s *Spring) isActive() bool { return s.running }

// advance 用半隐式欧拉积分推进弹簧物理（ticker 接口）。
func (s *Spring) advance(dt time.Duration) {
	if !s.running {
		return
	}
	mass := s.Mass
	if mass <= 0 {
		mass = 1
	}
	stiffness := s.Stiffness
	if stiffness <= 0 {
		stiffness = 180
	}
	damping := s.Damping
	if damping < 0 {
		damping = 0
	}

	s.elapsed += dt
	dtSec := dt.Seconds()
	if dtSec > 0.064 {
		dtSec = 0.064 // 限制单帧最大步长，避免卡顿后数值爆炸
	}
	const maxStep = 1.0 / 240.0
	steps := int(dtSec/maxStep) + 1
	h := dtSec / float64(steps)
	for i := 0; i < steps; i++ {
		force := -stiffness*(s.pos-s.To) - damping*s.vel
		s.vel += (force / mass) * h
		s.pos += s.vel * h
	}

	// 收敛检测（相对运动范围）+ 最大时长保护
	scale := math.Abs(s.To - s.From)
	if scale < 1 {
		scale = 1
	}
	settled := math.Abs(s.pos-s.To) < 0.001*scale && math.Abs(s.vel) < 0.01*scale
	if settled || s.elapsed > 10*time.Second {
		s.pos = s.To
		s.vel = 0
		s.running = false
		unregister(s)
		if s.OnUpdate != nil {
			s.OnUpdate(s.pos)
		}
		if s.OnDone != nil {
			s.OnDone()
		}
		return
	}
	if s.OnUpdate != nil {
		s.OnUpdate(s.pos)
	}
}
