package widget

import (
	"math"

	"github.com/user/goui/internal/canvas"
	"github.com/user/goui/internal/event"
	"github.com/user/goui/internal/layout"
	"github.com/user/goui/internal/paint"
	"github.com/user/goui/internal/types"
)

// Rate 评分控件（1:1 复刻 el-rate）。
// N 颗五角星，hover 预览、点选设值；支持半星、尺寸、只读、自定义颜色。
type Rate struct {
	StatelessWidget
	Value       float64       // 当前分值
	Max         int           // 星星总数(默认 5)
	OnChanged   func(float64) // 分值变化回调
	Disabled    bool          // 只读
	AllowHalf   bool          // 允许半星
	ActiveColor types.Color   // 选中色(el #F7BA2A)
	VoidColor   types.Color   // 未选中色(el #C6D1DE)
	Size        string        // large / "" (default) / small
}

// DefaultRate 返回默认配置的评分(5 星，el 金黄/浅灰)。
func DefaultRate() *Rate {
	return &Rate{
		Max:         5,
		ActiveColor: types.ColorFromRGB(247, 186, 42),  // el #F7BA2A
		VoidColor:   types.ColorFromRGB(198, 209, 222), // el #C6D1DE
	}
}

// NewRate 创建评分。
func NewRate(value float64, onChanged func(float64)) *Rate {
	r := DefaultRate()
	r.Value = value
	r.OnChanged = onChanged
	return r
}

func (r *Rate) WithMax(m int) *Rate                 { r.Max = m; return r }
func (r *Rate) WithDisabled(d bool) *Rate           { r.Disabled = d; return r }
func (r *Rate) WithAllowHalf(a bool) *Rate          { r.AllowHalf = a; return r }
func (r *Rate) WithActiveColor(c types.Color) *Rate { r.ActiveColor = c; return r }
func (r *Rate) WithSize(s string) *Rate             { r.Size = s; return r }

func (r *Rate) starSize() float64 {
	switch r.Size {
	case "large":
		return 28
	case "small":
		return 16
	default:
		return 20
	}
}

func (r *Rate) maxStars() int {
	if r.Max <= 0 {
		return 5
	}
	return r.Max
}

// CreateElement 创建 RateElement
func (r *Rate) CreateElement() Element {
	return &RateElement{BaseElement: BaseElement{widget: r}, rate: r, hoverValue: -1}
}

// RateElement 评分的运行时 Element
type RateElement struct {
	BaseElement
	rate       *Rate
	hoverValue float64 // 鼠标悬停预览值；-1 表示无 hover
}

// Build 返回空（无子控件）
func (e *RateElement) Build() []Element { return nil }

func (e *RateElement) Layout(ctx *layout.LayoutContext) layout.LayoutResult {
	r := e.rate
	ss := r.starSize()
	gap := ss * 0.25
	n := r.maxStars()
	w := float64(n)*ss + float64(n-1)*gap
	e.size = ctx.Constraints.Constrain(types.Size{Width: w, Height: ss})
	return layout.LayoutResult{Size: e.size}
}

func (e *RateElement) Paint(cvs canvas.Canvas, offset types.Point) {
	pos := e.Offset()
	r := e.rate
	ss := r.starSize()
	gap := ss * 0.25
	n := r.maxStars()
	outer := ss / 2
	inner := outer * 0.42

	// 显示值：hover 优先，否则当前值
	disp := r.Value
	if e.hoverValue >= 0 {
		disp = e.hoverValue
	}

	cy := pos.Y + ss/2
	for i := 0; i < n; i++ {
		cx := pos.X + float64(i)*(ss+gap) + ss/2
		fill := disp - float64(i) // 这颗星的填充量(>=1 满，0.5 半，<=0 空)
		switch {
		case fill >= 1:
			drawStar(cvs, cx, cy, outer, inner, r.ActiveColor)
		case fill >= 0.5 && r.AllowHalf:
			drawStar(cvs, cx, cy, outer, inner, r.VoidColor)
			cvs.Save()
			cvs.ClipRect(cx-outer, cy-outer, outer, 2*outer) // 裁到左半
			drawStar(cvs, cx, cy, outer, inner, r.ActiveColor)
			cvs.Restore()
		default:
			drawStar(cvs, cx, cy, outer, inner, r.VoidColor)
		}
	}
}

// drawStar 在 (cx,cy) 画一颗实心五角星。
func drawStar(cvs canvas.Canvas, cx, cy, outer, inner float64, color types.Color) {
	sp := paint.DefaultPaint()
	sp.Color = color
	cvs.DrawPath(starPath(cx, cy, outer, inner), sp)
}

// starPath 构造五角星路径：10 个顶点(5 外角 + 5 内角)，从正上方顶点开始。
func starPath(cx, cy, outer, inner float64) *canvas.Path {
	p := canvas.NewPath()
	for i := 0; i < 10; i++ {
		rr := outer
		if i%2 == 1 {
			rr = inner
		}
		ang := -math.Pi/2 + float64(i)*math.Pi/5 // 顶点朝上，每 36°
		x := cx + rr*math.Cos(ang)
		y := cy + rr*math.Sin(ang)
		if i == 0 {
			p.MoveTo(x, y)
		} else {
			p.LineTo(x, y)
		}
	}
	p.Close()
	return p
}

// valueAt 根据局部 x 算出对应分值（含半星）。
func (e *RateElement) valueAt(localX float64) float64 {
	r := e.rate
	ss := r.starSize()
	gap := ss * 0.25
	n := r.maxStars()
	unit := ss + gap
	idx := int(localX / unit)
	if idx < 0 {
		idx = 0
	}
	if idx >= n {
		idx = n - 1
	}
	within := localX - float64(idx)*unit
	if r.AllowHalf && within < ss/2 {
		return float64(idx) + 0.5
	}
	return float64(idx + 1)
}

func (e *RateElement) HandleEvent(ev event.Event) bool {
	if e.rate.Disabled {
		return false
	}
	switch ev.Type() {
	case event.TypeMouseMove, event.TypeMouseEnter:
		if me, ok := ev.(*event.MouseEvent); ok {
			e.hoverValue = e.valueAt(me.X - e.Offset().X)
			e.MarkNeedsPaint()
			return true
		}
	case event.TypeMouseLeave:
		e.hoverValue = -1
		e.MarkNeedsPaint()
		return true
	case event.TypeMouseDown:
		if me, ok := ev.(*event.MouseEvent); ok {
			v := e.valueAt(me.X - e.Offset().X)
			e.rate.Value = v
			if e.rate.OnChanged != nil {
				e.rate.OnChanged(v)
			}
			e.MarkNeedsPaint()
			return true
		}
	}
	return false
}

func (e *RateElement) Update(newWidget Widget) {
	if nr, ok := newWidget.(*Rate); ok {
		e.rate = nr
		e.BaseElement.widget = newWidget
		e.dirty = true
	}
}
