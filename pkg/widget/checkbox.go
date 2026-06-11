package widget

import (
	"github.com/hoonfeng/goui/pkg/canvas"
	"github.com/hoonfeng/goui/pkg/event"
	"github.com/hoonfeng/goui/internal/layout"
	"github.com/hoonfeng/goui/pkg/paint"
	"github.com/hoonfeng/goui/pkg/types"
)

// ─── Web 风格便捷构造函数 ──────────────────────────

// NewCheckbox 创建复选框控件。
// 用法: widget.NewCheckbox("记住我", true, fn)
// 标签和回调可选，传空字符串或 nil 表示不需要。
func NewCheckbox(label string, checked bool, onChanged func(bool)) *Checkbox {
	t := CurrentTheme()
	return &Checkbox{
		Label:       label,
		Checked:     checked,
		OnChanged:   onChanged,
		ActiveColor: t.Checkbox.ActiveColor,
		LabelColor:  t.Checkbox.LabelColor,
		BoxSize:     t.Checkbox.BoxSize,
		Font:        canvas.DefaultFont(),
	}
}

// OnChange 设置变化回调（链式调用）
func (c *Checkbox) OnChange(fn func(bool)) *Checkbox {
	c.OnChanged = fn
	return c
}

// SetChecked 设置选中状态（链式调用）
func (c *Checkbox) SetChecked(checked bool) *Checkbox {
	c.Checked = checked
	return c
}

// SetLabel 设置标签文字（链式调用）
func (c *Checkbox) SetLabel(label string) *Checkbox {
	c.Label = label
	return c
}

// SetColor 设置选中颜色（链式调用）
func (c *Checkbox) SetColor(color types.Color) *Checkbox {
	c.ActiveColor = color
	return c
}

// Checkbox 复选框控件
type Checkbox struct {
	StatelessWidget
	pseudoColors // CSS 伪类：交互态边框色覆盖（:hover）
	Checked     bool
	Label       string
	OnChanged   func(checked bool)
	ActiveColor types.Color
	Font        canvas.Font
	LabelColor  types.Color
	BoxSize     float64 // 方框尺寸
}

// DefaultCheckbox 返回默认配置的 Checkbox
// 配色从全局主题读取，用户可通过 SetTheme() 统一切换。
func DefaultCheckbox() *Checkbox {
	t := CurrentTheme()
	return &Checkbox{
		ActiveColor: t.Checkbox.ActiveColor,
		LabelColor:  t.Checkbox.LabelColor,
		BoxSize:     t.Checkbox.BoxSize,
		Font:        canvas.DefaultFont(),
	}
}

// CreateElement 创建 CheckboxElement
func (c *Checkbox) CreateElement() Element {
	return &CheckboxElement{
		BaseElement: BaseElement{widget: c},
		checkbox:    c,
	}
}

// CheckboxElement 复选框运行时 Element
type CheckboxElement struct {
	BaseElement
	checkbox *Checkbox
	hovered  bool
	pressed  bool
}

// Build 返回 nil（没有子控件）
func (e *CheckboxElement) Build() []Element { return nil }

// Layout 测量尺寸
func (e *CheckboxElement) Layout(ctx *layout.LayoutContext) layout.LayoutResult {
	c := e.checkbox
	font := c.Font
	if font.Size <= 0 {
		font = canvas.DefaultFont()
	}

	boxSize := c.BoxSize
	if boxSize <= 0 {
		boxSize = 18
	}

	// 如果有标签，使用全局文本度量精确测量标签宽度
	labelWidth := 0.0
	if c.Label != "" {
		metrics := canvas.MeasureTextGlobal(c.Label, c.Font)
		labelWidth = metrics.Width
	}

	totalWidth := boxSize + 6 + labelWidth // 方框 + 间距 + 标签
	height := boxSize + 4                  // 稍微高一点便于点击

	e.size = ctx.Constraints.Constrain(types.Size{
		Width:  totalWidth,
		Height: height,
	})
	return layout.LayoutResult{Size: e.size}
}

// Paint 绘制复选框
// 使用主题系统的配色，提供清晰的选中/未选中状态视觉反馈。
func (e *CheckboxElement) Paint(cvs canvas.Canvas, offset types.Point) {
	pos := e.Offset()
	c := e.checkbox

	boxSize := c.BoxSize
	if boxSize <= 0 {
		boxSize = 18
	}

	// 方框位置（垂直居中）
	boxX := pos.X
	boxY := pos.Y + (e.size.Height-boxSize)/2

	// ── 获取主题配色 ──
	t := CurrentTheme()
	activeColor := c.ActiveColor
	if activeColor == (types.Color{}) || (activeColor.R == 0 && activeColor.G == 0 && activeColor.B == 0 && activeColor.A == 0) {
		activeColor = t.Checkbox.ActiveColor
	}

	// ── 绘制方框 ──
	bgPaint := paint.DefaultPaint()
	borderPaint := paint.DefaultStrokePaint()
	borderPaint.StrokeWidth = 1.5

	if c.Checked {
		// 选中状态：实心填充 + 白色勾
		bgPaint.Color = activeColor
		cvs.DrawRoundedRect(boxX, boxY, boxSize, boxSize, t.Checkbox.BorderRadius, bgPaint)

		// 选中态边框（与填充同色，视觉上融为一体的同时给勾选标记做底）
		borderPaint.Color = activeColor
		cvs.DrawRoundedRect(boxX, boxY, boxSize, boxSize, t.Checkbox.BorderRadius, borderPaint)

		// ── 绘制勾选标记（✓）─ 更粗更清晰 ──
		linePaint := paint.DefaultStrokePaint()
		linePaint.Color = t.Checkbox.CheckColor
		linePaint.StrokeWidth = 2.5

		// ✓ 路径：从勾的底部到顶部（更标准的比例）
		margin := boxSize * 0.2
		x1 := boxX + margin
		y1 := boxY + boxSize*0.55
		x2 := boxX + boxSize*0.4
		y2 := boxY + boxSize*0.75
		x3 := boxX + boxSize*0.85
		y3 := boxY + boxSize*0.25

		cvs.DrawLine(x1, y1, x2, y2, linePaint)
		cvs.DrawLine(x2, y2, x3, y3, linePaint)
	} else {
		// 未选中状态：白色背景 + 灰色边框
		bgPaint.Color = t.Checkbox.InactiveBgColor
		cvs.DrawRoundedRect(boxX, boxY, boxSize, boxSize, t.Checkbox.BorderRadius, bgPaint)

		// 按下时加深边框 + 浅色背景
		if e.pressed {
			borderPaint.Color = types.Color{
				R: uint8(float64(t.Checkbox.HoverBorderColor.R) * 0.7),
				G: uint8(float64(t.Checkbox.HoverBorderColor.G) * 0.7),
				B: uint8(float64(t.Checkbox.HoverBorderColor.B) * 0.7),
				A: t.Checkbox.HoverBorderColor.A,
			}
			pressBg := paint.DefaultPaint()
			pressBg.Color = types.Color{
				R: uint8(float64(t.Checkbox.HoverBgColor.R) * 0.8),
				G: uint8(float64(t.Checkbox.HoverBgColor.G) * 0.8),
				B: uint8(float64(t.Checkbox.HoverBgColor.B) * 0.8),
				A: t.Checkbox.HoverBgColor.A,
			}
			cvs.DrawRoundedRect(boxX, boxY, boxSize, boxSize, t.Checkbox.BorderRadius, pressBg)
		} else if e.hovered {
			borderPaint.Color = c.hoverBorderOr(t.Checkbox.HoverBorderColor)
		} else {
			borderPaint.Color = c.borderOr(t.Checkbox.BorderColor)
		}
		cvs.DrawRoundedRect(boxX, boxY, boxSize, boxSize, t.Checkbox.BorderRadius, borderPaint)
	}

	// ── 绘制标签 ──
	if c.Label != "" {
		font := c.Font
		if font.Size <= 0 {
			font = canvas.DefaultFont()
		}
		textPaint := paint.DefaultPaint()
		textPaint.Color = c.LabelColor
		if textPaint.Color.A == 0 {
			textPaint.Color = t.Checkbox.LabelColor
			if textPaint.Color.A == 0 {
				textPaint.Color = elTextPrimary()
			}
		}

		// 使用实际 ascent 正确定位基线，使标签与方框文字对齐
		textX := boxX + boxSize + 8
		labelRect := types.Rect{X: textX, Y: pos.Y, Width: e.size.Width, Height: e.size.Height}
		canvas.DrawTextAligned(cvs, c.Label, labelRect, font, textPaint.Color, canvas.HAlignLeft, canvas.VAlignMiddle)
	}
}

// HandleEvent 处理事件
func (e *CheckboxElement) HandleEvent(ev event.Event) bool {
	switch ev.Type() {
	case event.TypeMouseEnter:
		e.hovered = true
		if OnNeedsRepaint != nil {
			OnNeedsRepaint()
		}
		return true
	case event.TypeMouseLeave:
		e.hovered = false
		e.pressed = false
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
		// 切换选中状态（无论 pressed 状态如何，保证向后兼容）
		c := e.checkbox
		c.Checked = !c.Checked
		if c.OnChanged != nil {
			c.OnChanged(c.Checked)
		}
		e.pressed = false
		if OnNeedsRepaint != nil {
			OnNeedsRepaint()
		}
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
			c := e.checkbox
			c.Checked = !c.Checked
			if c.OnChanged != nil {
				c.OnChanged(c.Checked)
			}
			e.pressed = false
			if OnNeedsRepaint != nil {
				OnNeedsRepaint()
			}
			return true
		}
		return false
	}
	return false
}

// Update 更新 Checkbox 配置
func (e *CheckboxElement) Update(newWidget Widget) {
	if newCheck, ok := newWidget.(*Checkbox); ok {
		e.checkbox = newCheck
		e.BaseElement.widget = newWidget
		e.dirty = true
	}
}
