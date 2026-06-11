package widget

import (
	"github.com/hoonfeng/goui/pkg/canvas"
	"github.com/hoonfeng/goui/internal/layout"
	"github.com/hoonfeng/goui/pkg/paint"
	"github.com/hoonfeng/goui/pkg/types"
)

// ProgressBar 进度条（1:1 复刻 el-progress line 类型）。
// 确定模式显示 Value(0~1)；Indeterminate 显示循环滑块(外部 SetAnimOffset 驱动)。
// 支持 Status 三态(success/warning/exception) + 右侧百分比标签。
type ProgressBar struct {
	StatelessWidget
	Value         float64     // 当前进度 0.0~1.0
	Indeterminate bool        // 不确定模式
	Height        float64     // 进度条高度
	Color         types.Color // 自定义前景色(无 Status 时优先)
	BGColor       types.Color // 背景色
	Radius        float64     // 圆角(默认胶囊)
	ShowLabel     bool        // 右侧显示百分比
	Status        string      // "" / "success" / "warning" / "exception"
	LabelColor    types.Color
	LabelFont     canvas.Font
}

// DefaultProgressBar 返回默认配置(el 主色 + 胶囊圆角)。
func DefaultProgressBar() *ProgressBar {
	return &ProgressBar{
		Height:     8,
		Color:      elPrimary(),  // el #409EFF
		BGColor:    elBorderLighter(),  // #ebeef5
		Radius:     100,                                // 胶囊(Paint 内 clamp 到 h/2)
		LabelColor: elTextRegular(),
	}
}

// NewProgress 创建带百分比标签的进度条。
func NewProgress(value float64) *ProgressBar {
	p := DefaultProgressBar()
	p.Value = value
	p.ShowLabel = true
	return p
}

func (p *ProgressBar) WithStatus(s string) *ProgressBar     { p.Status = s; return p }
func (p *ProgressBar) WithHeight(h float64) *ProgressBar     { p.Height = h; return p }
func (p *ProgressBar) WithColor(c types.Color) *ProgressBar  { p.Color = c; return p }
func (p *ProgressBar) WithLabel(show bool) *ProgressBar      { p.ShowLabel = show; return p }

// fgColor 按 Status 返回前景色(Status 优先于 Color)。
func (p *ProgressBar) fgColor() types.Color {
	switch p.Status {
	case "success":
		return elSuccess()
	case "warning":
		return elWarning()
	case "exception":
		return elDanger()
	}
	if p.Color != (types.Color{}) {
		return p.Color
	}
	return elPrimary()
}

func (p *ProgressBar) CreateElement() Element {
	return &ProgressBarElement{BaseElement: BaseElement{widget: p}, bar: p}
}

type ProgressBarElement struct {
	BaseElement
	bar        *ProgressBar
	animOffset float64
}

func (e *ProgressBarElement) Build() []Element { return nil }

const progressLabelRow = 18.0

func (e *ProgressBarElement) Layout(ctx *layout.LayoutContext) layout.LayoutResult {
	p := e.bar
	h := p.Height
	if h <= 0 {
		h = 8
	}
	rowH := h
	if p.ShowLabel && progressLabelRow > rowH {
		rowH = progressLabelRow // 标签在右侧，整行高度容纳文字
	}
	e.size = ctx.Constraints.Constrain(types.Size{
		Width:  ctx.Constraints.MaxWidth,
		Height: rowH + 4,
	})
	return layout.LayoutResult{Size: e.size}
}

func (e *ProgressBarElement) Paint(cvs canvas.Canvas, offset types.Point) {
	pos := e.Offset()
	p := e.bar
	h := p.Height
	if h <= 0 {
		h = 8
	}
	radius := p.Radius
	if radius > h/2 {
		radius = h / 2 // 胶囊封顶
	}
	if radius < 0 {
		radius = 0
	}
	rowH := e.size.Height - 4
	barY := pos.Y + (rowH-h)/2

	// 右侧标签区
	labelW := 0.0
	if p.ShowLabel && !p.Indeterminate {
		labelW = 46
	}
	barW := e.size.Width - labelW

	// 背景轨道
	bgPaint := paint.DefaultPaint()
	bgPaint.Color = p.BGColor
	cvs.DrawRoundedRect(pos.X, barY, barW, h, radius, bgPaint)

	// 前景进度
	fg := p.fgColor()
	fgPaint := paint.DefaultPaint()
	fgPaint.Color = fg

	if p.Indeterminate {
		blockW := barW * 0.3
		blockX := pos.X + e.animOffset*(barW+blockW) - blockW
		drawX, drawW := blockX, blockW
		if drawX < pos.X {
			drawW -= pos.X - drawX
			drawX = pos.X
		}
		if drawX+drawW > pos.X+barW {
			drawW = pos.X + barW - drawX
		}
		if drawW > 0 {
			cvs.DrawRoundedRect(drawX, barY, drawW, h, radius, fgPaint)
		}
	} else {
		v := p.Value
		if v < 0 {
			v = 0
		}
		if v > 1 {
			v = 1
		}
		if filledW := barW * v; filledW > 0 {
			cvs.DrawRoundedRect(pos.X, barY, filledW, h, radius, fgPaint)
		}
	}

	// 右侧百分比标签
	if p.ShowLabel && !p.Indeterminate {
		font := canvas.DefaultFont()
		font.Size = 13
		label := itoa(int(p.Value*100+0.5)) + "%"
		labelColor := p.LabelColor
		if p.Status != "" {
			labelColor = fg
		}
		lr := types.Rect{X: pos.X + barW + 6, Y: pos.Y, Width: labelW - 6, Height: rowH}
		canvas.DrawTextAligned(cvs, label, lr, font, labelColor, canvas.HAlignLeft, canvas.VAlignMiddle)
	}
}

// SetAnimOffset 设置不确定模式动画偏移(外部定时器驱动)。
func (e *ProgressBarElement) SetAnimOffset(offset float64) {
	e.animOffset = offset
	if OnNeedsRepaint != nil {
		OnNeedsRepaint()
	}
}

// itoa 简单整数转字符串(progressbar 与 slider 共用)。
func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	negative := false
	if n < 0 {
		negative = true
		n = -n
	}
	digits := make([]byte, 0, 10)
	for n > 0 {
		digits = append(digits, byte('0'+n%10))
		n /= 10
	}
	if negative {
		digits = append(digits, '-')
	}
	for i, j := 0, len(digits)-1; i < j; i, j = i+1, j-1 {
		digits[i], digits[j] = digits[j], digits[i]
	}
	return string(digits)
}
