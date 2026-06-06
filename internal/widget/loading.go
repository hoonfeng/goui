package widget

import (
	"math"
	"time"

	"github.com/user/goui/internal/animation"
	"github.com/user/goui/internal/canvas"
	"github.com/user/goui/internal/event"
	"github.com/user/goui/internal/layout"
	"github.com/user/goui/internal/paint"
	"github.com/user/goui/internal/types"
)

// ─── Loading 加载遮罩（L3 全屏/区域遮罩）────────────────────────
// 1:1 复刻 el-loading：半透明遮罩 + 居中旋转 spinner + 可选文字。
// 实现：「铺满 + 自绘遮罩 + 8 点环形 spinner(透明度渐变成拖尾) + 文字」。
// Angle 由动画控制器驱动旋转(静态渲染为 0)；与 Dialog/Drawer 同属 L3 模态层。

type Loading struct {
	StatelessWidget
	Text      string
	Color     types.Color // spinner/文字色(默认 el 蓝)
	MaskColor types.Color // 遮罩色(默认半透明白)
	Angle     float64     // spinner 当前旋转角(弧度，动画驱动；静态为 0)
	OnClose   func()      // 可选:点击关闭(el-loading 默认程序控制；设了则可点关，便于演示)
}

func NewLoading(text string) *Loading {
	return &Loading{Text: text}
}

func (l *Loading) WithColor(c types.Color) *Loading     { l.Color = c; return l }
func (l *Loading) WithMaskColor(c types.Color) *Loading { l.MaskColor = c; return l }

func (l *Loading) CreateElement() Element {
	return &LoadingElement{BaseElement: BaseElement{widget: l}, loading: l}
}

type LoadingElement struct {
	BaseElement
	loading *Loading
	spinner *animation.Controller // 驱动 spinner 持续旋转
}

// ensureSpinner 首次布局时启动无限旋转动画(Repeat 控制器持续推进 Angle)。
// 动画活跃期间 animation.HasActive() 为真，app 主循环持续出帧、不阻塞(见 app.go 主循环)。
func (e *LoadingElement) ensureSpinner() {
	if e.spinner != nil {
		return
	}
	e.spinner = animation.NewController(900*time.Millisecond, nil) // 900ms 转一圈，线性
	e.spinner.Repeat = true
	e.spinner.OnUpdate = func(v float64) {
		e.loading.Angle = v * 2 * math.Pi
		e.MarkNeedsPaint()
	}
	e.spinner.Start()
}

// Unmount 关闭时停掉 spinner，避免动画泄漏让主循环空转(HasActive 永真)。
func (e *LoadingElement) Unmount() {
	if e.spinner != nil {
		e.spinner.Stop()
		e.spinner = nil
	}
	e.BaseElement.Unmount()
}

func (e *LoadingElement) Layout(ctx *layout.LayoutContext) layout.LayoutResult {
	e.ensureSpinner()
	w := ctx.Constraints.MaxWidth
	h := ctx.Constraints.MaxHeight
	if w >= float64(1<<30) {
		w = 200
	}
	if h >= float64(1<<30) {
		h = 200
	}
	e.size = types.Size{Width: w, Height: h}
	return layout.LayoutResult{Size: e.size}
}

func (e *LoadingElement) Paint(cvs canvas.Canvas, offset types.Point) {
	pos := e.Offset()
	l := e.loading

	// 遮罩
	maskColor := l.MaskColor
	if maskColor == (types.Color{}) {
		maskColor = types.ColorFromRGBA(255, 255, 255, 217) // 半透明白
	}
	mp := paint.DefaultPaint()
	mp.Color = maskColor
	cvs.DrawRect(pos.X, pos.Y, e.size.Width, e.size.Height, mp)

	// spinner 颜色
	col := l.Color
	if col == (types.Color{}) {
		col = elPrimary() // el 蓝
	}

	// 居中 8 点环形 spinner
	cx := pos.X + e.size.Width/2
	cy := pos.Y + e.size.Height/2
	if l.Text != "" {
		cy -= 20 // 给下方文字留位
	}
	const spinnerR = 15.0
	const dotR = 3.0
	for i := 0; i < 8; i++ {
		ang := l.Angle + float64(i)*math.Pi/4
		dx := cx + spinnerR*math.Cos(ang)
		dy := cy + spinnerR*math.Sin(ang)
		alpha := uint8(40 + 215*float64(i)/7) // 透明度沿环渐变，形成旋转拖尾
		dp := paint.DefaultPaint()
		dp.Color = types.Color{R: col.R, G: col.G, B: col.B, A: alpha}
		cvs.DrawCircle(dx, dy, dotR, dp)
	}

	// 文字
	if l.Text != "" {
		font := canvas.DefaultFont()
		font.Size = 14
		rect := types.Rect{X: pos.X, Y: cy + spinnerR + 12, Width: e.size.Width, Height: 22}
		canvas.DrawTextAligned(cvs, l.Text, rect, font, col, canvas.HAlignCenter, canvas.VAlignMiddle)
	}
}

// HandleEvent 若设了 OnClose，点击任意处关闭(demo 友好；默认 nil 时纯程序控制)。
func (e *LoadingElement) HandleEvent(ev event.Event) bool {
	if ev.Type() == event.TypeMouseDown && e.loading.OnClose != nil {
		e.loading.OnClose()
		return true
	}
	return false
}

func (e *LoadingElement) Update(newWidget Widget) {
	if nl, ok := newWidget.(*Loading); ok {
		e.loading = nl
		e.BaseElement.widget = newWidget
		e.dirty = true
	}
}
