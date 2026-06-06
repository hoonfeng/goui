package widget

import (
	"strconv"

	"github.com/user/goui/internal/canvas"
	"github.com/user/goui/internal/layout"
	"github.com/user/goui/internal/paint"
	"github.com/user/goui/internal/types"
)

// ─── Steps 步骤条（1:1 复刻 el-steps horizontal）─────────────────
// N 个步骤：圆圈(序号/✓) + 标题 + 描述 + 连接线；按 Active 分 finish/process/wait 三态。
// 展示型组件(进度由外部 Active 控制)。

// Step 一个步骤。
type Step struct {
	Title       string
	Description string
}

// Steps 步骤条。
type Steps struct {
	StatelessWidget
	Steps  []Step
	Active int // 当前步骤(0-indexed)；< Active 为已完成
}

func NewSteps(steps ...Step) *Steps     { return &Steps{Steps: steps} }
func (s *Steps) WithActive(i int) *Steps { s.Active = i; return s }

func (s *Steps) CreateElement() Element {
	return &StepsElement{BaseElement: BaseElement{widget: s}, steps: s}
}

type StepsElement struct {
	BaseElement
	steps *Steps
}

func (e *StepsElement) Build() []Element { return nil }

const stepsH = 64.0

func (e *StepsElement) Layout(ctx *layout.LayoutContext) layout.LayoutResult {
	w := ctx.Constraints.MaxWidth
	if w >= float64(1<<30) {
		w = 480
	}
	e.size = ctx.Constraints.Constrain(types.Size{Width: w, Height: stepsH})
	return layout.LayoutResult{Size: e.size}
}

func (e *StepsElement) Paint(cvs canvas.Canvas, offset types.Point) {
	pos := e.Offset()
	steps := e.steps.Steps
	n := len(steps)
	if n == 0 {
		return
	}
	active := e.steps.Active

	blue := elPrimary()
	gray := elPlaceholder() // #c0c4cc
	titleDark := elTextPrimary()
	descGray := elTextSecondary()

	stepW := e.size.Width / float64(n)
	const r = 14.0
	cy := pos.Y + r + 2

	for i, step := range steps {
		cx := pos.X + float64(i)*stepW + r
		finished := i < active
		current := i == active

		// 圆圈
		if finished || current {
			cp := paint.DefaultPaint()
			cp.Color = blue
			cvs.DrawCircle(cx, cy, r, cp)
		} else {
			cp := paint.DefaultStrokePaint()
			cp.Color = gray
			cp.StrokeWidth = 1.5
			cvs.DrawCircle(cx, cy, r, cp)
		}

		// 圆内：✓(已完成) 或 序号
		iconRect := types.Rect{X: cx - r, Y: cy - r, Width: 2 * r, Height: 2 * r}
		nf := canvas.DefaultFont()
		nf.Size = 13
		nf.Weight = canvas.FontWeightBold
		if finished {
			canvas.DrawTextAligned(cvs, "✓", iconRect, nf, types.ColorWhite, canvas.HAlignCenter, canvas.VAlignMiddle)
		} else {
			numColor := types.ColorWhite
			if !current {
				numColor = gray
			}
			canvas.DrawTextAligned(cvs, strconv.Itoa(i+1), iconRect, nf, numColor, canvas.HAlignCenter, canvas.VAlignMiddle)
		}

		// 标题(圆右)
		tf := canvas.DefaultFont()
		tf.Size = 14
		tf.Weight = canvas.FontWeightBold
		tColor := titleDark
		if i > active {
			tColor = descGray
		}
		titleX := cx + r + 8
		titleW := canvas.MeasureTextGlobal(step.Title, tf).Width
		titleRect := types.Rect{X: titleX, Y: cy - 13, Width: titleW + 4, Height: 18}
		canvas.DrawTextAligned(cvs, step.Title, titleRect, tf, tColor, canvas.HAlignLeft, canvas.VAlignMiddle)

		// 连接线(从标题右侧 → 下一步圆左，避开标题文字不重叠)
		if i < n-1 {
			lp := paint.DefaultStrokePaint()
			lp.StrokeWidth = 2
			if finished {
				lp.Color = blue
			} else {
				lp.Color = elBorderLight()
			}
			lineStart := titleX + titleW + 10
			lineEnd := pos.X + float64(i+1)*stepW - 4
			if lineEnd > lineStart {
				cvs.DrawLine(lineStart, cy, lineEnd, cy, lp)
			}
		}

		// 描述(标题下)
		if step.Description != "" {
			df := canvas.DefaultFont()
			df.Size = 12
			descRect := types.Rect{X: cx + r + 8, Y: cy + 6, Width: stepW - 2*r - 16, Height: 16}
			canvas.DrawTextAligned(cvs, step.Description, descRect, df, descGray, canvas.HAlignLeft, canvas.VAlignMiddle)
		}
	}
}

func (e *StepsElement) Update(newWidget Widget) {
	if ns, ok := newWidget.(*Steps); ok {
		e.steps = ns
		e.BaseElement.widget = newWidget
		e.dirty = true
	}
}
