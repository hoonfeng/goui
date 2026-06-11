package widget

import (
	"github.com/user/goui/internal/canvas"
	"github.com/user/goui/internal/event"
	"github.com/user/goui/internal/layout"
	"github.com/user/goui/internal/paint"
	"github.com/user/goui/internal/types"
)

// Slider 滑块选择器（1:1 复刻 el-slider）：拖拽选值；白底 + 主色边框的滑块、hover 放大、支持禁用。
type Slider struct {
	StatelessWidget
	Min           float64
	Max           float64
	Step          float64
	Value         float64
	OnChanged     func(float64)
	ActiveColor   types.Color
	InactiveColor types.Color
	ThumbColor    types.Color
	ThumbRadius   float64
	TrackHeight   float64
	ShowLabel     bool
	LabelColor    types.Color
	Font          canvas.Font
	Disabled      bool
}

// DefaultSlider 返回默认配置（el 主色 + 白底边框滑块）。
func DefaultSlider() *Slider {
	return &Slider{
		Min:           0,
		Max:           100,
		Step:          1,
		Value:         0,
		ActiveColor:   elPrimary(),  // el #409EFF
		InactiveColor: elBorderLight(), // #e4e7ed
		ThumbColor:    elPrimary(),
		ThumbRadius:   9,
		TrackHeight:   6,
		LabelColor:    elTextRegular(),
	}
}

// NewSlider 创建滑块(默认 0~100)。
func NewSlider(value float64, onChanged func(float64)) *Slider {
	s := DefaultSlider()
	s.Value = value
	s.OnChanged = onChanged
	return s
}

func (s *Slider) WithRange(min, max float64) *Slider { s.Min, s.Max = min, max; return s }
func (s *Slider) WithStep(st float64) *Slider        { s.Step = st; return s }
func (s *Slider) WithLabel(show bool) *Slider        { s.ShowLabel = show; return s }
func (s *Slider) WithDisabled(d bool) *Slider        { s.Disabled = d; return s }
func (s *Slider) WithColor(c types.Color) *Slider    { s.ActiveColor, s.ThumbColor = c, c; return s }

// CreateElement 创建 SliderElement
func (s *Slider) CreateElement() Element {
	return &SliderElement{BaseElement: BaseElement{widget: s}, slider: s}
}

// SliderElement 滑块的运行时 Element
type SliderElement struct {
	BaseElement
	slider    *Slider
	hovered   bool
	dragging  bool
	thumbSize float64
}

func (e *SliderElement) Build() []Element { return nil }

// Layout 测量滑块尺寸（标签占上方一行高度，与旧版一致）。
func (e *SliderElement) Layout(ctx *layout.LayoutContext) layout.LayoutResult {
	s := e.slider
	thumbR := s.ThumbRadius
	if thumbR <= 0 {
		thumbR = 9
	}
	e.thumbSize = thumbR * 2

	labelH := 0.0
	if s.ShowLabel {
		font := s.Font
		if font.Size <= 0 {
			font = canvas.DefaultFont()
		}
		labelH = font.Size * 1.4
	}
	e.size = ctx.Constraints.Constrain(types.Size{
		Width:  ctx.Constraints.MaxWidth,
		Height: e.thumbSize + labelH + 8,
	})
	return layout.LayoutResult{Size: e.size}
}

func (e *SliderElement) Paint(cvs canvas.Canvas, offset types.Point) {
	pos := e.Offset()
	s := e.slider
	thumbR := s.ThumbRadius
	if thumbR <= 0 {
		thumbR = 9
	}
	trackH := s.TrackHeight
	if trackH <= 0 {
		trackH = 6
	}
	padding := thumbR + 2
	trackW := e.size.Width - padding*2
	if trackW < 1 {
		trackW = 1
	}
	ratio := (s.Value - s.Min) / (s.Max - s.Min)
	if ratio < 0 {
		ratio = 0
	}
	if ratio > 1 {
		ratio = 1
	}
	centerY := pos.Y + e.size.Height/2
	trackY := centerY - trackH/2
	thumbCX := pos.X + padding + ratio*trackW
	thumbCY := centerY

	active := s.ActiveColor
	inactive := s.InactiveColor
	if s.Disabled {
		active = lighten(active, 0.45)
	}

	// 未激活轨道(右)
	ip := paint.DefaultPaint()
	ip.Color = inactive
	cvs.DrawRoundedRect(pos.X+padding+ratio*trackW, trackY, trackW*(1-ratio), trackH, trackH/2, ip)
	// 激活轨道(左)
	ap := paint.DefaultPaint()
	ap.Color = active
	cvs.DrawRoundedRect(pos.X+padding, trackY, ratio*trackW, trackH, trackH/2, ap)

	// 滑块(el 风格：阴影 + 白底 + 主色边框；hover/拖拽放大)
	tr := thumbR
	if (e.hovered || e.dragging) && !s.Disabled {
		tr = thumbR + 1.5
	}
	shadow := paint.DefaultPaint()
	shadow.Color = types.ColorFromRGBA(0, 0, 0, 28)
	cvs.DrawCircle(thumbCX, thumbCY+1, tr, shadow)
	white := paint.DefaultPaint()
	white.Color = elSurface()
	cvs.DrawCircle(thumbCX, thumbCY, tr, white)
	border := paint.DefaultStrokePaint()
	border.Color = active
	border.StrokeWidth = 2
	cvs.DrawCircle(thumbCX, thumbCY, tr, border)

	// 标签(滑块上方居中)
	if s.ShowLabel {
		font := s.Font
		if font.Size <= 0 {
			font = canvas.DefaultFont()
			font.Size = 13
		}
		label := formatFloat(s.Value, 0)
		lr := types.Rect{X: thumbCX - 24, Y: pos.Y, Width: 48, Height: font.Size * 1.4}
		canvas.DrawTextAligned(cvs, label, lr, font, s.LabelColor, canvas.HAlignCenter, canvas.VAlignMiddle)
	}
}

func (e *SliderElement) HandleEvent(ev event.Event) bool {
	if e.slider.Disabled {
		return false
	}
	switch ev.Type() {
	case event.TypeMouseEnter:
		e.hovered = true
		e.MarkNeedsPaint()
		return true
	case event.TypeMouseLeave:
		// 仅清悬停；拖拽中鼠标移出滑块不应停拖（有鼠标捕获，按住不放就继续跟手），由 Up/DragEnd 才停。
		e.hovered = false
		e.MarkNeedsPaint()
		return true
	// 按下即开始拖：MouseDown 先吸到点击处；超过 app 拖拽阈值(5px)后事件转为 Drag* —— 两路都收，才能连续跟手。
	case event.TypeMouseDown, event.TypeDragStart:
		e.dragging = true
		e.updateValueFromEvent(ev)
		e.MarkNeedsPaint()
		return true
	case event.TypeMouseMove, event.TypeDragMove:
		if e.dragging {
			e.updateValueFromEvent(ev)
			return true
		}
		return false
	case event.TypeMouseUp, event.TypeDragEnd:
		if e.dragging {
			e.dragging = false
			e.MarkNeedsPaint()
			return true
		}
		return false
	}
	return false
}

// updateValueFromEvent 根据鼠标位置更新值(含步长吸附)。兼容 MouseEvent 与 DragEvent（拖拽中是后者）。
func (e *SliderElement) updateValueFromEvent(ev event.Event) {
	mx, ok := eventMouseX(ev)
	if !ok {
		return
	}
	s := e.slider
	pos := e.Offset()
	thumbR := s.ThumbRadius
	if thumbR <= 0 {
		thumbR = 9
	}
	padding := thumbR + 2
	trackW := e.size.Width - padding*2
	if trackW < 1 {
		trackW = 1
	}
	ratio := (mx - (pos.X + padding)) / trackW
	if ratio < 0 {
		ratio = 0
	}
	if ratio > 1 {
		ratio = 1
	}
	value := s.Min + ratio*(s.Max-s.Min)
	if s.Step > 0 {
		steps := (value - s.Min) / s.Step
		value = s.Min + float64(int(steps+0.5))*s.Step
		if value > s.Max {
			value = s.Max
		}
		if value < s.Min {
			value = s.Min
		}
	}
	if value != s.Value {
		s.Value = value
		if s.OnChanged != nil {
			s.OnChanged(value)
		}
		e.MarkNeedsPaint()
	}
}

// eventMouseX 从 MouseEvent / DragEvent 取鼠标 X：超过 app 拖拽阈值后，MouseMove 会被改派为 DragMove。
func eventMouseX(ev event.Event) (float64, bool) {
	switch m := ev.(type) {
	case *event.MouseEvent:
		return m.X, true
	case *event.DragEvent:
		return m.X, true
	}
	return 0, false
}

// formatFloat 简单浮点格式化(当前仅整数)。
func formatFloat(v float64, decimals int) string {
	n := int(v + 0.5)
	return itoa(n)
}

// Update 更新 Slider 配置
func (e *SliderElement) Update(newWidget Widget) {
	if newSl, ok := newWidget.(*Slider); ok {
		e.slider = newSl
		e.BaseElement.widget = newWidget
		e.dirty = true
	}
}
