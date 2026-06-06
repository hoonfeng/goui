package widget

import (
	"fmt"
	"time"

	"github.com/user/goui/internal/canvas"
	"github.com/user/goui/internal/event"
	"github.com/user/goui/internal/layout"
	"github.com/user/goui/internal/paint"
	"github.com/user/goui/internal/types"
)

// ─── Calendar 日历（1:1 复刻 el-calendar 月视图）──────────────────
// 独立月历(文档流、非弹出)：‹ 年月 › 导航 + 星期行 + 6×7 大格日期。
// 复用 datepicker.go 的 daysInMonth/firstWeekday/parseDate/dpWeekHeads 工具。

type Calendar struct {
	StatelessWidget
	Value    string // 选中日期 "2006-01-02"
	OnSelect func(string)
}

func NewCalendar() *Calendar                                { return &Calendar{} }
func (c *Calendar) WithValue(v string) *Calendar           { c.Value = v; return c }
func (c *Calendar) WithOnSelect(fn func(string)) *Calendar { c.OnSelect = fn; return c }

func (c *Calendar) CreateElement() Element {
	return &CalendarElement{BaseElement: BaseElement{widget: c}, cal: c, hoverDay: -1}
}

const (
	calCellW  = 60.0
	calCellH  = 48.0
	calHeader = 48.0
	calWeekH  = 34.0
	calPad    = 8.0
)

type CalendarElement struct {
	BaseElement
	cal       *Calendar
	viewYear  int
	viewMonth time.Month
	inited    bool
	hoverDay  int
	selected  string // 运行时选中(点击设)；为空时用 cal.Value
}

func (e *CalendarElement) Build() []Element { return nil }

func (e *CalendarElement) ensureView() {
	if e.inited {
		return
	}
	e.inited = true
	if t, ok := parseDate(e.cal.Value); ok {
		e.viewYear, e.viewMonth = t.Year(), t.Month()
	} else {
		now := time.Now()
		e.viewYear, e.viewMonth = now.Year(), now.Month()
	}
}

func (e *CalendarElement) curSelected() string {
	if e.selected != "" {
		return e.selected
	}
	return e.cal.Value
}

func (e *CalendarElement) Layout(ctx *layout.LayoutContext) layout.LayoutResult {
	e.ensureView()
	w := calPad*2 + calCellW*7
	h := calPad*2 + calHeader + calWeekH + calCellH*6
	e.size = ctx.Constraints.Constrain(types.Size{Width: w, Height: h})
	return layout.LayoutResult{Size: e.size}
}

func (e *CalendarElement) Paint(cvs canvas.Canvas, offset types.Point) {
	e.ensureView()
	pos := e.Offset()
	W, H := e.size.Width, e.size.Height

	bg := paint.DefaultPaint()
	bg.Color = elSurface()
	cvs.DrawRoundedRect(pos.X, pos.Y, W, H, 6, bg)
	bp := paint.DefaultStrokePaint()
	bp.Color = elBorderLight()
	bp.StrokeWidth = 1
	cvs.DrawRoundedRect(pos.X+0.5, pos.Y+0.5, W-1, H-1, 6, bp)

	// Header：‹ 年月 ›
	hy := pos.Y + calPad
	acy := hy + calHeader/2
	arrow := paint.DefaultStrokePaint()
	arrow.Color = elTextRegular()
	arrow.StrokeWidth = 1.6
	lx := pos.X + calPad + 14
	cvs.DrawLine(lx+4, acy-5, lx-2, acy, arrow)
	cvs.DrawLine(lx-2, acy, lx+4, acy+5, arrow)
	rx := pos.X + W - calPad - 16
	cvs.DrawLine(rx, acy-5, rx+6, acy, arrow)
	cvs.DrawLine(rx+6, acy, rx, acy+5, arrow)
	tf := canvas.DefaultFont()
	tf.Size = 16
	tf.Weight = canvas.FontWeightBold
	title := fmt.Sprintf("%d 年 %d 月", e.viewYear, int(e.viewMonth))
	canvas.DrawTextAligned(cvs, title, types.Rect{X: pos.X + 60, Y: hy, Width: W - 120, Height: calHeader},
		tf, elTextPrimary(), canvas.HAlignCenter, canvas.VAlignMiddle)

	// 星期行
	wf := canvas.DefaultFont()
	wf.Size = 13
	wy := hy + calHeader
	gridX := pos.X + calPad
	sep := paint.DefaultStrokePaint()
	sep.Color = elBorderLighter()
	sep.StrokeWidth = 1
	cvs.DrawLine(gridX, wy, pos.X+W-calPad, wy, sep)
	for i, wh := range dpWeekHeads() {
		canvas.DrawTextAligned(cvs, wh, types.Rect{X: gridX + float64(i)*calCellW, Y: wy, Width: calCellW, Height: calWeekH},
			wf, elTextRegular(), canvas.HAlignCenter, canvas.VAlignMiddle)
	}

	// 网格
	gridY := wy + calWeekH
	off := firstWeekday(e.viewYear, e.viewMonth)
	dim := daysInMonth(e.viewYear, e.viewMonth)
	df := canvas.DefaultFont()
	df.Size = 14
	var selY, selM, selD int
	hasSel := false
	if t, ok := parseDate(e.curSelected()); ok {
		selY, selM, selD, hasSel = t.Year(), int(t.Month()), t.Day(), true
	}
	now := time.Now()
	todY, todM, todD := now.Year(), int(now.Month()), now.Day()

	for day := 1; day <= dim; day++ {
		cell := off + day - 1
		col := cell % 7
		row := cell / 7
		cx := gridX + float64(col)*calCellW
		cy := gridY + float64(row)*calCellH
		isSel := hasSel && selY == e.viewYear && selM == int(e.viewMonth) && selD == day
		isToday := todY == e.viewYear && todM == int(e.viewMonth) && todD == day

		if isSel {
			f := paint.DefaultPaint()
			f.Color = elPrimary()
			cvs.DrawRoundedRect(cx+4, cy+4, calCellW-8, calCellH-8, 4, f)
		} else if e.hoverDay == day {
			f := paint.DefaultPaint()
			f.Color = types.ColorFromRGB(236, 245, 255)
			cvs.DrawRoundedRect(cx+4, cy+4, calCellW-8, calCellH-8, 4, f)
		}
		color := elTextPrimary()
		if isSel {
			color = types.ColorFromRGB(255, 255, 255)
		} else if isToday {
			color = elPrimary()
		}
		canvas.DrawTextAligned(cvs, fmt.Sprintf("%d", day), types.Rect{X: cx, Y: cy, Width: calCellW, Height: calCellH},
			df, color, canvas.HAlignCenter, canvas.VAlignMiddle)
	}
}

func (e *CalendarElement) cellDayAt(lx, ly float64) int {
	gridX := calPad
	gridY := calPad + calHeader + calWeekH
	if lx < gridX || ly < gridY {
		return -1
	}
	col := int((lx - gridX) / calCellW)
	row := int((ly - gridY) / calCellH)
	if col < 0 || col > 6 || row < 0 || row > 5 {
		return -1
	}
	day := row*7 + col - firstWeekday(e.viewYear, e.viewMonth) + 1
	if day < 1 || day > daysInMonth(e.viewYear, e.viewMonth) {
		return -1
	}
	return day
}

func (e *CalendarElement) shiftMonth(d int) {
	m := int(e.viewMonth) + d
	for m < 1 {
		m += 12
		e.viewYear--
	}
	for m > 12 {
		m -= 12
		e.viewYear++
	}
	e.viewMonth = time.Month(m)
}

func (e *CalendarElement) HandleEvent(ev event.Event) bool {
	me, ok := ev.(*event.MouseEvent)
	if !ok {
		return false
	}
	pos := e.Offset()
	lx, ly := me.X-pos.X, me.Y-pos.Y
	switch ev.Type() {
	case event.TypeMouseMove, event.TypeMouseEnter:
		d := e.cellDayAt(lx, ly)
		if d != e.hoverDay {
			e.hoverDay = d
			e.MarkNeedsPaint()
		}
		return true
	case event.TypeMouseLeave:
		e.hoverDay = -1
		e.MarkNeedsPaint()
		return true
	case event.TypeMouseDown:
		// header 箭头切月（尺寸不变，纯重绘）
		if ly >= calPad && ly < calPad+calHeader {
			if lx < calPad+44 {
				e.shiftMonth(-1)
			} else if lx > e.size.Width-calPad-44 {
				e.shiftMonth(1)
			}
			e.MarkNeedsPaint()
			return true
		}
		if d := e.cellDayAt(lx, ly); d > 0 {
			e.selected = fmt.Sprintf("%04d-%02d-%02d", e.viewYear, int(e.viewMonth), d)
			if e.cal.OnSelect != nil {
				e.cal.OnSelect(e.selected)
			}
			e.MarkNeedsPaint()
		}
		return true
	}
	return false
}

func (e *CalendarElement) Update(newWidget Widget) {
	if nc, ok := newWidget.(*Calendar); ok {
		e.cal = nc
		e.BaseElement.widget = newWidget
		e.dirty = true
	}
}
