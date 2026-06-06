package widget

import (
	"time"

	"github.com/user/goui/internal/animation"
	"github.com/user/goui/internal/canvas"
	"github.com/user/goui/internal/event"
	"github.com/user/goui/internal/layout"
	"github.com/user/goui/internal/paint"
	"github.com/user/goui/internal/types"
)

// Switch 开关控件（1:1 复刻 el-switch）。
// 开/关两态，点击切换，滑块平滑滑动；支持轨道内文字、右侧标签、尺寸、禁用。
type Switch struct {
	StatelessWidget
	Value         bool        // 当前开关状态
	OnChanged     func(bool)  // 状态变化回调
	ActiveColor   types.Color // 开启时轨道色
	InactiveColor types.Color // 关闭时轨道色
	ThumbColor    types.Color // 滑块色
	ActiveText    string      // 开启时轨道内文字（el inline-prompt）
	InactiveText  string      // 关闭时轨道内文字
	Label         string      // 右侧标签
	LabelColor    types.Color // 标签色
	Font          canvas.Font // 标签字体
	Disabled      bool        // 禁用
	Size          string      // "large" / "" (default) / "small"
}

// DefaultSwitch 返回默认配置的开关（读取 Theme.Switch token）。
func DefaultSwitch() *Switch {
	t := CurrentTheme()
	return &Switch{
		ActiveColor:   t.Switch.ActiveColor,
		InactiveColor: t.Switch.InactiveColor,
		ThumbColor:    t.Switch.ThumbColor,
		LabelColor:    elTextRegular(),
	}
}

// NewSwitch 创建开关。
func NewSwitch(value bool, onChanged func(bool)) *Switch {
	s := DefaultSwitch()
	s.Value = value
	s.OnChanged = onChanged
	return s
}

// 链式 setter（Web 风格）。
func (s *Switch) WithLabel(label string) *Switch          { s.Label = label; return s }
func (s *Switch) WithActiveColor(c types.Color) *Switch   { s.ActiveColor = c; return s }
func (s *Switch) WithInactiveColor(c types.Color) *Switch { s.InactiveColor = c; return s }
func (s *Switch) WithTexts(active, inactive string) *Switch {
	s.ActiveText, s.InactiveText = active, inactive
	return s
}
func (s *Switch) WithDisabled(d bool) *Switch  { s.Disabled = d; return s }
func (s *Switch) WithSize(size string) *Switch { s.Size = size; return s }

// dims 按尺寸返回轨道宽/高。
func (s *Switch) dims() (trackW, trackH float64) {
	switch s.Size {
	case "large":
		return 50, 26
	case "small":
		return 36, 18
	default:
		return 44, 24
	}
}

func (s *Switch) labelFont() canvas.Font {
	if s.Font.Size > 0 {
		return s.Font
	}
	f := canvas.DefaultFont()
	f.Size = 14
	return f
}

// CreateElement 创建 SwitchElement
func (s *Switch) CreateElement() Element {
	return &SwitchElement{
		BaseElement: BaseElement{widget: s},
		sw:          s,
	}
}

// SwitchElement 开关的运行时 Element
type SwitchElement struct {
	BaseElement
	sw            *Switch
	hovered       bool
	pressed       bool
	thumbProgress float64               // 滑块位置 0(关)→1(开)
	thumbInit     bool                  // thumbProgress 是否已按初值初始化
	thumbCtrl     *animation.Controller // 驱动滑块平滑滑动
}

// Build 返回空（无子控件）
func (e *SwitchElement) Build() []Element { return nil }

// animateThumb 把滑块进度平滑过渡到 target(0/1)。
func (e *SwitchElement) animateThumb(target float64) {
	from := e.thumbProgress
	if e.thumbCtrl != nil {
		e.thumbCtrl.Stop()
	}
	e.thumbCtrl = animation.NewController(220*time.Millisecond, animation.EaseOutCubic)
	e.thumbCtrl.OnUpdate = func(v float64) {
		e.thumbProgress = animation.LerpFloat(from, target, v)
		e.MarkNeedsPaint()
	}
	e.thumbCtrl.Start()
}

// Layout 测量开关尺寸
func (e *SwitchElement) Layout(ctx *layout.LayoutContext) layout.LayoutResult {
	s := e.sw
	trackW, trackH := s.dims()

	labelW := 0.0
	if s.Label != "" {
		labelW = 6 + canvas.MeasureTextGlobal(s.Label, s.labelFont()).Width
	}

	e.size = ctx.Constraints.Constrain(types.Size{
		Width:  trackW + labelW,
		Height: trackH + 4,
	})
	return layout.LayoutResult{Size: e.size}
}

// Paint 绘制开关
func (e *SwitchElement) Paint(cvs canvas.Canvas, offset types.Point) {
	pos := e.Offset()
	s := e.sw
	trackW, trackH := s.dims()
	thumbR := (trackH - 4) / 2

	// 首帧按 Value 初始化滑块位置（避免从 0 滑入）
	if !e.thumbInit {
		if s.Value {
			e.thumbProgress = 1
		}
		e.thumbInit = true
	}

	trackY := pos.Y + (e.size.Height-trackH)/2

	// 轨道色：按滑块进度在 inactive↔active 间插值
	bgColor := animation.LerpColor(s.InactiveColor, s.ActiveColor, e.thumbProgress)
	switch {
	case s.Disabled:
		bgColor = lighten(bgColor, 0.5) // 禁用变浅
	case e.pressed:
		bgColor = darken(bgColor, 0.9)
	}

	// 轨道
	bgPaint := paint.DefaultPaint()
	bgPaint.Color = bgColor
	cvs.DrawRoundedRect(pos.X, trackY, trackW, trackH, trackH/2, bgPaint)

	// 滑块位置（按进度插值）
	thumbLeft := pos.X + thumbR + 2
	thumbRight := pos.X + trackW - thumbR - 2
	thumbCX := thumbLeft + e.thumbProgress*(thumbRight-thumbLeft)
	thumbCY := trackY + trackH/2

	// 轨道内文字（el inline-prompt）：显示在滑块对侧空白处
	if s.ActiveText != "" || s.InactiveText != "" {
		txt := s.InactiveText
		if e.thumbProgress > 0.5 {
			txt = s.ActiveText
		}
		if txt != "" {
			tf := canvas.DefaultFont()
			tf.Size = trackH * 0.46
			var tr types.Rect
			if e.thumbProgress > 0.5 {
				tr = types.Rect{X: pos.X, Y: trackY, Width: thumbRight - thumbR - pos.X, Height: trackH}
			} else {
				tr = types.Rect{X: thumbLeft + thumbR, Y: trackY, Width: pos.X + trackW - (thumbLeft + thumbR), Height: trackH}
			}
			canvas.DrawTextAligned(cvs, txt, tr, tf, types.ColorWhite, canvas.HAlignCenter, canvas.VAlignMiddle)
		}
	}

	// 滑块（圆形）
	thumbColor := s.ThumbColor
	if s.Disabled {
		thumbColor = lighten(thumbColor, 0.2)
	}
	thumbPaint := paint.DefaultPaint()
	thumbPaint.Color = thumbColor
	cvs.DrawCircle(thumbCX, thumbCY, thumbR, thumbPaint)

	// 右侧标签（统一对齐原语）
	if s.Label != "" {
		labelColor := s.LabelColor
		if s.Disabled {
			labelColor = lighten(labelColor, 0.4)
		}
		rect := types.Rect{X: pos.X + trackW + 6, Y: pos.Y, Width: e.size.Width - trackW - 6, Height: e.size.Height}
		canvas.DrawTextAligned(cvs, s.Label, rect, s.labelFont(), labelColor, canvas.HAlignLeft, canvas.VAlignMiddle)
	}
}

// toggle 切换状态并触发滑动动画 + 回调。
func (e *SwitchElement) toggle() {
	s := e.sw
	s.Value = !s.Value
	target := 0.0
	if s.Value {
		target = 1
	}
	e.animateThumb(target)
	if s.OnChanged != nil {
		s.OnChanged(s.Value)
	}
}

// HandleEvent 处理事件
func (e *SwitchElement) HandleEvent(ev event.Event) bool {
	if e.sw.Disabled {
		return false
	}
	switch ev.Type() {
	case event.TypeMouseEnter:
		e.hovered = true
		e.MarkNeedsPaint()
		return true
	case event.TypeMouseLeave:
		e.hovered = false
		e.pressed = false
		e.MarkNeedsPaint()
		return true
	case event.TypeMouseDown:
		e.pressed = true
		e.MarkNeedsPaint()
		return true
	case event.TypeMouseUp:
		e.pressed = false
		e.toggle()
		return true
	case event.TypeKeyDown:
		if keyEv, ok := ev.(*event.KeyEvent); ok && (keyEv.Key == "Enter" || keyEv.Key == "Space") {
			e.pressed = true
			e.MarkNeedsPaint()
			return true
		}
		return false
	case event.TypeKeyUp:
		if keyEv, ok := ev.(*event.KeyEvent); ok && (keyEv.Key == "Enter" || keyEv.Key == "Space") {
			e.pressed = false
			e.toggle()
			return true
		}
		return false
	}
	return false
}

// Update 更新 Switch 配置
func (e *SwitchElement) Update(newWidget Widget) {
	if newSw, ok := newWidget.(*Switch); ok {
		e.sw = newSw
		e.BaseElement.widget = newWidget
		e.dirty = true
	}
}

// lighten 把颜色按 amt(0~1) 向白色靠拢（用于禁用态变浅）。
func lighten(c types.Color, amt float64) types.Color {
	return types.Color{
		R: uint8(float64(c.R) + (255-float64(c.R))*amt),
		G: uint8(float64(c.G) + (255-float64(c.G))*amt),
		B: uint8(float64(c.B) + (255-float64(c.B))*amt),
		A: c.A,
	}
}

// darken 把颜色各分量乘以 factor(<1 变深，用于按下态)。
func darken(c types.Color, factor float64) types.Color {
	return types.Color{
		R: uint8(float64(c.R) * factor),
		G: uint8(float64(c.G) * factor),
		B: uint8(float64(c.B) * factor),
		A: c.A,
	}
}
