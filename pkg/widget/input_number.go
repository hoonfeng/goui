package widget

import (
	"strconv"
	"time"

	"github.com/hoonfeng/goui/pkg/animation"
	"github.com/hoonfeng/goui/pkg/canvas"
	"github.com/hoonfeng/goui/pkg/event"
	"github.com/hoonfeng/goui/internal/layout"
	"github.com/hoonfeng/goui/pkg/paint"
	"github.com/hoonfeng/goui/pkg/types"
)

// InputNumber 数字输入框（1:1 复刻 el-input-number）。
// 经典布局 [−] 数字 [+]：两侧加减按钮 ± Step，clamp 到 [Min,Max]，到边界按钮变灰。
// 支持精度、尺寸、禁用。（键盘直接输入为进阶，当前以步进按钮为主。）
type InputNumber struct {
	StatelessWidget
	Value     float64
	Min       float64
	Max       float64
	Step      float64
	Precision int // 小数位数
	OnChanged func(float64)
	Disabled  bool
	Size      string  // large / "" (default) / small
	Width     float64 // 总宽(默认 180)
}

// DefaultInputNumber 返回默认配置(步长 1，范围实际无限，宽 180)。
func DefaultInputNumber() *InputNumber {
	return &InputNumber{
		Min:   -1e15,
		Max:   1e15,
		Step:  1,
		Width: 180,
	}
}

// NewInputNumber 创建数字输入框。
func NewInputNumber(value float64, onChanged func(float64)) *InputNumber {
	n := DefaultInputNumber()
	n.Value = value
	n.OnChanged = onChanged
	return n
}

func (n *InputNumber) WithRange(min, max float64) *InputNumber { n.Min, n.Max = min, max; return n }
func (n *InputNumber) WithStep(s float64) *InputNumber         { n.Step = s; return n }
func (n *InputNumber) WithPrecision(p int) *InputNumber        { n.Precision = p; return n }
func (n *InputNumber) WithDisabled(d bool) *InputNumber        { n.Disabled = d; return n }
func (n *InputNumber) WithSize(s string) *InputNumber          { n.Size = s; return n }
func (n *InputNumber) WithWidth(w float64) *InputNumber        { n.Width = w; return n }

func (n *InputNumber) height() float64 {
	switch n.Size {
	case "large":
		return 40
	case "small":
		return 24
	default:
		return 32
	}
}

func (n *InputNumber) btnW() float64 { return n.height() }

func (n *InputNumber) formatValue() string {
	return strconv.FormatFloat(n.Value, 'f', n.Precision, 64)
}

// CreateElement 创建 InputNumberElement
func (n *InputNumber) CreateElement() Element {
	return &InputNumberElement{BaseElement: BaseElement{widget: n}, num: n}
}

// InputNumberElement 数字输入框的运行时 Element
type InputNumberElement struct {
	BaseElement
	num         *InputNumber
	repeatCtrl  *animation.Controller // 长按连续步进的计时器
	repeatDelta float64               // 连续步进的方向/步长
	repeatPhase float64               // 相位，用于检测周期回绕
}

func (e *InputNumberElement) Build() []Element { return nil }

func (e *InputNumberElement) Layout(ctx *layout.LayoutContext) layout.LayoutResult {
	n := e.num
	w := n.Width
	if w <= 0 {
		w = 180
	}
	e.size = ctx.Constraints.Constrain(types.Size{Width: w, Height: n.height()})
	return layout.LayoutResult{Size: e.size}
}

func (e *InputNumberElement) Paint(cvs canvas.Canvas, offset types.Point) {
	pos := e.Offset()
	n := e.num
	h := n.height()
	bw := n.btnW()
	W := e.size.Width
	cy := pos.Y + h/2

	border := elBorder() // #dcdfe6
	normal := elTextRegular()    // #606266
	muted := elPlaceholder()    // #c0c4cc

	// 背景框（白底 + 圆角）
	bgPaint := paint.DefaultPaint()
	bgPaint.Color = elSurface()
	cvs.DrawRoundedRect(pos.X, pos.Y, W, h, 4, bgPaint)
	// 边框
	bp := paint.DefaultStrokePaint()
	bp.Color = border
	bp.StrokeWidth = 1
	cvs.DrawRoundedRect(pos.X, pos.Y, W, h, 4, bp)
	// 按钮与数字的分隔竖线
	cvs.DrawLine(pos.X+bw, pos.Y, pos.X+bw, pos.Y+h, bp)
	cvs.DrawLine(pos.X+W-bw, pos.Y, pos.X+W-bw, pos.Y+h, bp)

	atMin := n.Value <= n.Min
	atMax := n.Value >= n.Max

	// 左 [−] 按钮
	minusColor := normal
	if atMin || n.Disabled {
		minusColor = muted
	}
	mp := paint.DefaultStrokePaint()
	mp.Color = minusColor
	mp.StrokeWidth = 1.5
	mcx := pos.X + bw/2
	cvs.DrawLine(mcx-bw*0.22, cy, mcx+bw*0.22, cy, mp)

	// 右 [+] 按钮
	plusColor := normal
	if atMax || n.Disabled {
		plusColor = muted
	}
	pp := paint.DefaultStrokePaint()
	pp.Color = plusColor
	pp.StrokeWidth = 1.5
	pcx := pos.X + W - bw/2
	cvs.DrawLine(pcx-bw*0.22, cy, pcx+bw*0.22, cy, pp) // 横
	cvs.DrawLine(pcx, cy-bw*0.22, pcx, cy+bw*0.22, pp) // 竖

	// 中间数字（居中）
	font := canvas.DefaultFont()
	font.Size = 14
	textColor := normal
	if n.Disabled {
		textColor = muted
	}
	numRect := types.Rect{X: pos.X + bw, Y: pos.Y, Width: W - 2*bw, Height: h}
	canvas.DrawTextAligned(cvs, n.formatValue(), numRect, font, textColor, canvas.HAlignCenter, canvas.VAlignMiddle)
}

// step 调整数值并 clamp 到范围。
func (e *InputNumberElement) step(delta float64) {
	n := e.num
	v := n.Value + delta
	if v < n.Min {
		v = n.Min
	}
	if v > n.Max {
		v = n.Max
	}
	if v == n.Value {
		return
	}
	n.Value = v
	if n.OnChanged != nil {
		n.OnChanged(v)
	}
	e.MarkNeedsPaint()
}

// startRepeat 启动长按连续步进：首次 MouseDown 已 step 一次，
// 此后停顿 ~400ms 再每 ~110ms 连续触发(对齐 el-input-number 的 auto-repeat)。
// 用 animation.Controller 的循环当定时器：相位每回绕一次 = 完成一个周期，触发一次 step。
func (e *InputNumberElement) startRepeat(delta float64) {
	e.stopRepeat()
	e.repeatDelta = delta
	e.repeatPhase = 0
	c := animation.NewController(110*time.Millisecond, nil) // 线性循环当定时器
	c.Repeat = true
	c.StartDelay = 400 * time.Millisecond
	c.OnUpdate = func(v float64) {
		if v < e.repeatPhase { // 相位回绕=完成一个周期
			e.step(e.repeatDelta)
		}
		e.repeatPhase = v
	}
	c.Start()
	e.repeatCtrl = c
}

// stopRepeat 停止连续步进(松开/移出时)。
func (e *InputNumberElement) stopRepeat() {
	if e.repeatCtrl != nil {
		e.repeatCtrl.Stop()
		e.repeatCtrl = nil
	}
}

func (e *InputNumberElement) HandleEvent(ev event.Event) bool {
	if e.num.Disabled {
		return false
	}
	switch ev.Type() {
	case event.TypeMouseDown:
		if me, ok := ev.(*event.MouseEvent); ok {
			localX := me.X - e.Offset().X
			bw := e.num.btnW()
			switch {
			case localX < bw:
				e.step(-e.num.Step)
				e.startRepeat(-e.num.Step)
				return true
			case localX > e.size.Width-bw:
				e.step(e.num.Step)
				e.startRepeat(e.num.Step)
				return true
			}
		}
	case event.TypeMouseUp, event.TypeMouseLeave:
		e.stopRepeat()
		return true
	}
	return false
}

func (e *InputNumberElement) Update(newWidget Widget) {
	if nn, ok := newWidget.(*InputNumber); ok {
		e.num = nn
		e.BaseElement.widget = newWidget
		e.dirty = true
	}
}
