package widget

import (
	"github.com/hoonfeng/goui/pkg/canvas"
	"github.com/hoonfeng/goui/pkg/event"
	"github.com/hoonfeng/goui/internal/layout"
	"github.com/hoonfeng/goui/pkg/paint"
	"github.com/hoonfeng/goui/pkg/types"
)

// RadioButton 单选按钮控件。
// 通常与 RadioGroup 配合使用，也可独立使用（由外部控制选中状态）。
type RadioButton struct {
	StatelessWidget
	pseudoColors // CSS 伪类：交互态边框色覆盖（:hover）
	GroupName   string      // 分组名称
	Value       interface{} // 关联值
	Selected    bool        // 是否选中
	OnChanged   func()      // 选中回调
	Label       string      // 标签文本
	ActiveColor types.Color // 选中颜色
	LabelColor  types.Color // 标签颜色
	Font        canvas.Font // 标签字体
	RadioSize   float64     // 单选按钮直径
}

// DefaultRadioButton 返回默认配置的单选按钮
func DefaultRadioButton() *RadioButton {
	return &RadioButton{
		ActiveColor: types.ColorFromRGB(66, 133, 244),
		LabelColor:  elTextPrimary(),
		RadioSize:   18,
		Font:        canvas.DefaultFont(),
	}
}

// CreateElement 创建 RadioButtonElement
func (r *RadioButton) CreateElement() Element {
	return &RadioButtonElement{
		BaseElement: BaseElement{widget: r},
		radio:       r,
	}
}

// RadioButtonElement 单选按钮的运行时 Element
type RadioButtonElement struct {
	BaseElement
	radio   *RadioButton
	hovered bool
	pressed bool
}

// Build 返回空（无子控件）
func (e *RadioButtonElement) Build() []Element { return nil }

// Layout 测量尺寸
func (e *RadioButtonElement) Layout(ctx *layout.LayoutContext) layout.LayoutResult {
	r := e.radio
	radioSize := r.RadioSize
	if radioSize <= 0 {
		radioSize = 18
	}

	labelW := 0.0
	if r.Label != "" {
		font := r.Font
		if font.Size <= 0 {
			font = canvas.DefaultFont()
		}
		labelW = float64(len(r.Label)) * font.Size * 0.6
	}

	totalW := radioSize + 6 + labelW
	totalH := radioSize + 4

	e.size = ctx.Constraints.Constrain(types.Size{
		Width:  totalW,
		Height: totalH,
	})
	return layout.LayoutResult{Size: e.size}
}

// Paint 绘制单选按钮
func (e *RadioButtonElement) Paint(cvs canvas.Canvas, offset types.Point) {
	pos := e.Offset()
	r := e.radio

	radioSize := r.RadioSize
	if radioSize <= 0 {
		radioSize = 18
	}

	// 垂直居中
	centerY := pos.Y + e.size.Height/2

	// 绘制外圆
	outerPaint := paint.DefaultStrokePaint()
	outerPaint.StrokeWidth = 2

	color := r.ActiveColor
	if r.Selected {
		outerPaint.Color = color
	} else if e.hovered || e.pressed {
		outerPaint.Color = r.hoverBorderOr(types.ColorFromRGB(100, 100, 100))
	} else {
		outerPaint.Color = r.borderOr(types.ColorFromRGB(180, 180, 180))
	}

	cx := pos.X + radioSize/2
	cy := centerY
	outerR := radioSize / 2

	cvs.DrawCircle(cx, cy, outerR, outerPaint)

	// 如果选中，绘制内圆（填充）
	if r.Selected {
		innerPaint := paint.DefaultPaint()
		innerPaint.Color = color
		innerR := outerR * 0.5
		cvs.DrawCircle(cx, cy, innerR, innerPaint)
	}

	// 绘制标签
	if r.Label != "" {
		font := r.Font
		if font.Size <= 0 {
			font = canvas.DefaultFont()
		}
		textPaint := paint.DefaultPaint()
		textPaint.Color = r.LabelColor
		textX := pos.X + radioSize + 6
		textY := centerY + font.Size/3
		cvs.DrawText(r.Label, textX, textY, font, textPaint)
	}
}

// HandleEvent 处理事件
func (e *RadioButtonElement) HandleEvent(ev event.Event) bool {
	switch ev.Type() {
	case event.TypeMouseEnter:
		e.hovered = true
		if OnNeedsRepaint != nil {
			OnNeedsRepaint()
		}
		return true
	case event.TypeMouseLeave:
		e.hovered = false
		if OnNeedsRepaint != nil {
			OnNeedsRepaint()
		}
		return true
	case event.TypeMouseDown:
		e.pressed = true
		if OnNeedsRepaint != nil {
			OnNeedsRepaint()
		}
		return true
	case event.TypeMouseUp:
		r := e.radio
		if !r.Selected {
			r.Selected = true
			if OnNeedsRepaint != nil {
				OnNeedsRepaint()
			}
			if r.OnChanged != nil {
				r.OnChanged()
			}
			// 通知同一组的其他 RadioButton 取消选中
			e.notifyGroup()
		}
		e.pressed = false
		return true
	case event.TypeKeyDown:
		keyEv, ok := ev.(*event.KeyEvent)
		if ok && (keyEv.Key == "Enter" || keyEv.Key == "Space") {
			e.pressed = true
			if OnNeedsRepaint != nil {
				OnNeedsRepaint()
			}
			return true
		}
		return false
	case event.TypeKeyUp:
		keyEv, ok := ev.(*event.KeyEvent)
		if ok && (keyEv.Key == "Enter" || keyEv.Key == "Space") {
			r := e.radio
			if !r.Selected {
				r.Selected = true
				if OnNeedsRepaint != nil {
					OnNeedsRepaint()
				}
				if r.OnChanged != nil {
					r.OnChanged()
				}
				// 通知同一组的其他 RadioButton 取消选中
				e.notifyGroup()
			}
			e.pressed = false
			return true
		}
		return false
	}
	return false
}

// notifyGroup 通知同一组的其他 RadioButton 取消选中
func (e *RadioButtonElement) notifyGroup() {
	parent := e.Parent()
	if parent == nil {
		return
	}
	// 遍历兄弟节点，查找同一 Group 的 RadioButton
	for _, child := range parent.Children() {
		if child == e {
			continue
		}
		if radioEl, ok := child.(*RadioButtonElement); ok {
			if radioEl.radio.GroupName == e.radio.GroupName && radioEl.radio.Selected {
				radioEl.radio.Selected = false
				if OnNeedsRepaint != nil {
					OnNeedsRepaint()
				}
				if radioEl.radio.OnChanged != nil {
					radioEl.radio.OnChanged()
				}
			}
		}
	}
}

// Update 更新 RadioButton 配置
func (e *RadioButtonElement) Update(newWidget Widget) {
	if newR, ok := newWidget.(*RadioButton); ok {
		e.radio = newR
		e.BaseElement.widget = newWidget
		e.dirty = true
	}
}
