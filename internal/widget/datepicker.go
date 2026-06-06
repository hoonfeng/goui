package widget

import (
	"fmt"
	"time"

	"github.com/user/goui/internal/canvas"
	"github.com/user/goui/internal/event"
	"github.com/user/goui/internal/i18n"
	"github.com/user/goui/internal/layout"
	"github.com/user/goui/internal/paint"
	"github.com/user/goui/internal/types"
)

// ─── DatePicker 日期选择器（1:1 复刻 el-date-picker 单选核心）──────
// 触发器(输入框样式 + 日历图标) + 点击弹出日历浮层(复用 Overlay 地基)。
// 日历面板：« ‹ 年月 › » 导航 + 星期行 + 6×7 日期网格(今日/选中/hover 高亮)。
// 当前为单日期选择；范围选择/时间/快捷选项为进阶。

// DatePicker 日期选择器配置。
type DatePicker struct {
	StatelessWidget
	pseudoColors // CSS 伪类：交互态边框色覆盖（:hover/:focus）
	Value       string // "2006-01-02" 或空
	Placeholder string
	OnChange    func(string)
	Width       float64
	Disabled    bool
}

// NewDatePicker 创建日期选择器。
func NewDatePicker() *DatePicker {
	return &DatePicker{Width: 220}
}

func (d *DatePicker) WithValue(v string) *DatePicker          { d.Value = v; return d }
func (d *DatePicker) WithPlaceholder(p string) *DatePicker    { d.Placeholder = p; return d }
func (d *DatePicker) WithOnChange(fn func(string)) *DatePicker { d.OnChange = fn; return d }
func (d *DatePicker) WithWidth(w float64) *DatePicker         { d.Width = w; return d }
func (d *DatePicker) WithDisabled(b bool) *DatePicker         { d.Disabled = b; return d }

func (d *DatePicker) CreateElement() Element {
	return &DatePickerElement{BaseElement: BaseElement{widget: d}, dp: d}
}

// DatePickerElement 触发器运行时 Element。
type DatePickerElement struct {
	BaseElement
	dp        *DatePicker
	overlayID int
	open      bool
	hovered   bool
}

func (e *DatePickerElement) Build() []Element { return nil }

func (e *DatePickerElement) Layout(ctx *layout.LayoutContext) layout.LayoutResult {
	w := e.dp.Width
	if w <= 0 {
		w = 220
	}
	e.size = ctx.Constraints.Constrain(types.Size{Width: w, Height: 32})
	return layout.LayoutResult{Size: e.size}
}

func (e *DatePickerElement) Paint(cvs canvas.Canvas, offset types.Point) {
	pos := e.Offset()
	dp := e.dp
	W, H := e.size.Width, e.size.Height

	border := dp.borderOr(elBorder())
	if (e.hovered || e.open) && !dp.Disabled {
		if e.open {
			border = dp.focusBorderOr(elPrimary())
		} else {
			border = dp.hoverBorderOr(elPrimary())
		}
	}
	bg := paint.DefaultPaint()
	bg.Color = elSurface()
	if dp.Disabled {
		bg.Color = elFill()
	}
	cvs.DrawRoundedRect(pos.X, pos.Y, W, H, 4, bg)
	bp := paint.DefaultStrokePaint()
	bp.Color = border
	bp.StrokeWidth = 1
	cvs.DrawRoundedRect(pos.X+0.5, pos.Y+0.5, W-1, H-1, 4, bp)

	// 日历图标（左侧）：空心框 + 顶分隔线 + 两个挂钩
	ic := paint.DefaultStrokePaint()
	ic.Color = elPlaceholder()
	ic.StrokeWidth = 1.2
	ix := pos.X + 10
	iy := pos.Y + H/2 - 6
	cvs.DrawRoundedRect(ix, iy, 12, 12, 2, ic)
	cvs.DrawLine(ix, iy+3.5, ix+12, iy+3.5, ic)
	cvs.DrawLine(ix+3.5, iy-2, ix+3.5, iy+1, ic)
	cvs.DrawLine(ix+8.5, iy-2, ix+8.5, iy+1, ic)

	// 文本：值或 placeholder
	label := dp.Value
	textColor := elTextRegular()
	if label == "" {
		label = orI18n(dp.Placeholder, "el.datepicker.selectDate")
		textColor = elPlaceholder()
	}
	if dp.Disabled {
		textColor = elPlaceholder()
	}
	font := canvas.DefaultFont()
	font.Size = 14
	tr := types.Rect{X: pos.X + 30, Y: pos.Y, Width: W - 30 - 12, Height: H}
	canvas.DrawTextAligned(cvs, label, tr, font, textColor, canvas.HAlignLeft, canvas.VAlignMiddle)
}

func (e *DatePickerElement) HandleEvent(ev event.Event) bool {
	if e.dp.Disabled {
		return false
	}
	switch ev.Type() {
	case event.TypeMouseEnter:
		e.hovered = true
		e.MarkNeedsPaint()
		return true
	case event.TypeMouseLeave:
		e.hovered = false
		e.MarkNeedsPaint()
		return true
	case event.TypeMouseDown:
		if e.open {
			e.hide()
		} else {
			e.show()
		}
		e.MarkNeedsPaint()
		return true
	}
	return false
}

func (e *DatePickerElement) show() {
	if e.open {
		return
	}
	off := e.Offset()
	panel := &datePanel{
		selected: e.dp.Value,
		onSelect: func(v string) {
			e.dp.Value = v
			if e.dp.OnChange != nil {
				e.dp.OnChange(v)
			}
			e.hide()
			e.MarkNeedsPaint()
		},
	}
	entry := &OverlayEntry{
		AnchorRect:     types.Rect{X: off.X, Y: off.Y, Width: e.size.Width, Height: e.size.Height},
		Content:        panel,
		Placement:      PlacementBottomStart,
		Gap:            6,
		Transition:     "fade",
		OnClickOutside: func() { e.hide(); e.MarkNeedsPaint() },
	}
	e.overlayID = ShowOverlay(entry)
	e.open = true
}

func (e *DatePickerElement) hide() {
	if !e.open {
		return
	}
	HideOverlay(e.overlayID)
	e.open = false
}

func (e *DatePickerElement) Update(newWidget Widget) {
	if nd, ok := newWidget.(*DatePicker); ok {
		e.dp = nd
		e.BaseElement.widget = newWidget
		e.dirty = true
	}
}

// ─── datePanel 日历浮层 ─────────────────────────────────────────

type datePanel struct {
	StatelessWidget
	selected string
	onSelect func(string)
}

func (p *datePanel) CreateElement() Element {
	return &datePanelElement{BaseElement: BaseElement{widget: p}, panel: p, hoverDay: -1}
}

const (
	dpCell    = 36.0
	dpPad     = 12.0
	dpHeaderH = 40.0
	dpWeekH   = 32.0
	dpFooterH = 38.0
)

func dpWeekHeads() []string { // 星期表头（按当前语言，0=周日）
	w := make([]string, 7)
	for i := range w {
		w[i] = i18n.WeekdayShort(i)
	}
	return w
}

type datePanelElement struct {
	BaseElement
	panel     *datePanel
	viewYear  int
	viewMonth time.Month
	inited    bool
	hoverDay  int
}

func (e *datePanelElement) Build() []Element { return nil }

// ensureView 首次依据选中值(或今天)初始化显示的年月。
func (e *datePanelElement) ensureView() {
	if e.inited {
		return
	}
	e.inited = true
	if t, ok := parseDate(e.panel.selected); ok {
		e.viewYear, e.viewMonth = t.Year(), t.Month()
	} else {
		now := time.Now()
		e.viewYear, e.viewMonth = now.Year(), now.Month()
	}
}

func (e *datePanelElement) Layout(ctx *layout.LayoutContext) layout.LayoutResult {
	e.ensureView()
	w := dpPad*2 + dpCell*7
	h := dpPad*2 + dpHeaderH + dpWeekH + dpCell*6 + dpFooterH
	e.size = ctx.Constraints.Constrain(types.Size{Width: w, Height: h})
	return layout.LayoutResult{Size: e.size}
}

func (e *datePanelElement) Paint(cvs canvas.Canvas, offset types.Point) {
	e.ensureView()
	pos := e.Offset()
	W, H := e.size.Width, e.size.Height

	// 阴影 + 白底 + 边框
	shadow := paint.DefaultPaint()
	shadow.Color = types.ColorFromRGBA(0, 0, 0, 22)
	cvs.DrawRoundedRect(pos.X, pos.Y+2, W, H, 6, shadow)
	bg := paint.DefaultPaint()
	bg.Color = elSurface()
	cvs.DrawRoundedRect(pos.X, pos.Y, W, H, 6, bg)
	bp := paint.DefaultStrokePaint()
	bp.Color = elBorderLight()
	bp.StrokeWidth = 1
	cvs.DrawRoundedRect(pos.X+0.5, pos.Y+0.5, W-1, H-1, 6, bp)

	// ─ Header：« ‹ 年月 › » ─
	hy := pos.Y + dpPad
	acy := hy + dpHeaderH/2
	arrow := paint.DefaultStrokePaint()
	arrow.Color = elTextRegular()
	arrow.StrokeWidth = 1.4
	// « 上一年
	ax := pos.X + dpPad + 4
	cvs.DrawLine(ax+4, acy-4, ax, acy, arrow)
	cvs.DrawLine(ax, acy, ax+4, acy+4, arrow)
	cvs.DrawLine(ax+9, acy-4, ax+5, acy, arrow)
	cvs.DrawLine(ax+5, acy, ax+9, acy+4, arrow)
	// ‹ 上一月
	bx := pos.X + dpPad + 30
	cvs.DrawLine(bx+4, acy-4, bx, acy, arrow)
	cvs.DrawLine(bx, acy, bx+4, acy+4, arrow)
	// 年月标题
	tf := canvas.DefaultFont()
	tf.Size = 15
	tf.Weight = canvas.FontWeightBold
	title := fmt.Sprintf("%d 年 %d 月", e.viewYear, int(e.viewMonth))
	tr := types.Rect{X: pos.X + 56, Y: hy, Width: W - 112, Height: dpHeaderH}
	canvas.DrawTextAligned(cvs, title, tr, tf, elTextPrimary(), canvas.HAlignCenter, canvas.VAlignMiddle)
	// › 下一月
	dx := pos.X + W - dpPad - 34
	cvs.DrawLine(dx, acy-4, dx+4, acy, arrow)
	cvs.DrawLine(dx+4, acy, dx, acy+4, arrow)
	// » 下一年
	ex := pos.X + W - dpPad - 13
	cvs.DrawLine(ex, acy-4, ex+4, acy, arrow)
	cvs.DrawLine(ex+4, acy, ex, acy+4, arrow)
	cvs.DrawLine(ex+5, acy-4, ex+9, acy, arrow)
	cvs.DrawLine(ex+9, acy, ex+5, acy+4, arrow)

	// 分隔线
	sep := paint.DefaultStrokePaint()
	sep.Color = elBorderLighter()
	sep.StrokeWidth = 1
	cvs.DrawLine(pos.X+dpPad, hy+dpHeaderH, pos.X+W-dpPad, hy+dpHeaderH, sep)

	// ─ 星期行 ─
	wf := canvas.DefaultFont()
	wf.Size = 13
	wy := hy + dpHeaderH
	gridX := pos.X + dpPad
	for i, wh := range dpWeekHeads() {
		wr := types.Rect{X: gridX + float64(i)*dpCell, Y: wy, Width: dpCell, Height: dpWeekH}
		canvas.DrawTextAligned(cvs, wh, wr, wf, elTextRegular(), canvas.HAlignCenter, canvas.VAlignMiddle)
	}

	// ─ 日期网格 ─
	gridY := wy + dpWeekH
	off := firstWeekday(e.viewYear, e.viewMonth)
	dim := daysInMonth(e.viewYear, e.viewMonth)
	df := canvas.DefaultFont()
	df.Size = 14

	var selY, selM, selD int
	hasSel := false
	if t, ok := parseDate(e.panel.selected); ok {
		selY, selM, selD, hasSel = t.Year(), int(t.Month()), t.Day(), true
	}
	now := time.Now()
	todY, todM, todD := now.Year(), int(now.Month()), now.Day()

	for day := 1; day <= dim; day++ {
		cell := off + day - 1
		row := cell / 7
		col := cell % 7
		cx := gridX + float64(col)*dpCell + dpCell/2
		cy := gridY + float64(row)*dpCell + dpCell/2

		isSel := hasSel && selY == e.viewYear && selM == int(e.viewMonth) && selD == day
		isToday := todY == e.viewYear && todM == int(e.viewMonth) && todD == day
		isHover := e.hoverDay == day

		if isSel {
			fill := paint.DefaultPaint()
			fill.Color = elPrimary()
			cvs.DrawCircle(cx, cy, 15, fill)
		} else if isHover {
			hb := paint.DefaultPaint()
			hb.Color = types.ColorFromRGB(236, 245, 255)
			cvs.DrawCircle(cx, cy, 15, hb)
		}
		color := elTextRegular()
		if isSel {
			color = types.ColorFromRGB(255, 255, 255)
		} else if isToday {
			color = elPrimary()
		}
		dr := types.Rect{X: cx - dpCell/2, Y: cy - dpCell/2, Width: dpCell, Height: dpCell}
		canvas.DrawTextAligned(cvs, fmt.Sprintf("%d", day), dr, df, color, canvas.HAlignCenter, canvas.VAlignMiddle)
	}

	// ─ 底部「今天」快捷 ─
	footerY := gridY + dpCell*6
	cvs.DrawLine(pos.X+dpPad, footerY, pos.X+W-dpPad, footerY, sep)
	tf2 := canvas.DefaultFont()
	tf2.Size = 14
	fr := types.Rect{X: pos.X, Y: footerY, Width: W, Height: dpFooterH}
	canvas.DrawTextAligned(cvs, orI18n("", "el.datepicker.today"), fr, tf2, elPrimary(), canvas.HAlignCenter, canvas.VAlignMiddle)
}

func (e *datePanelElement) HandleEvent(ev event.Event) bool {
	me, ok := ev.(*event.MouseEvent)
	if !ok {
		return false
	}
	pos := e.Offset()
	lx := me.X - pos.X
	ly := me.Y - pos.Y
	switch ev.Type() {
	case event.TypeMouseMove, event.TypeMouseEnter:
		day := e.dayAt(lx, ly)
		if day != e.hoverDay {
			e.hoverDay = day
			e.MarkNeedsPaint()
		}
		return true
	case event.TypeMouseLeave:
		e.hoverDay = -1
		e.MarkNeedsPaint()
		return true
	case event.TypeMouseDown:
		// Header 箭头区
		if ly >= dpPad && ly < dpPad+dpHeaderH {
			W := e.size.Width
			switch {
			case lx < dpPad+24:
				e.shiftYear(-1)
			case lx < dpPad+48:
				e.shiftMonth(-1)
			case lx > W-dpPad-24:
				e.shiftYear(1)
			case lx > W-dpPad-48:
				e.shiftMonth(1)
			default:
				return true
			}
			e.MarkNeedsPaint()
			return true
		}
		// 底部「今天」快捷
		if ly >= dpPad+dpHeaderH+dpWeekH+dpCell*6 {
			if e.panel.onSelect != nil {
				now := time.Now()
				e.panel.onSelect(fmt.Sprintf("%04d-%02d-%02d", now.Year(), int(now.Month()), now.Day()))
			}
			return true
		}
		// 日期格
		day := e.dayAt(lx, ly)
		if day > 0 && e.panel.onSelect != nil {
			e.panel.onSelect(fmt.Sprintf("%04d-%02d-%02d", e.viewYear, int(e.viewMonth), day))
		}
		return true
	}
	return false
}

// dayAt 返回局部坐标命中的当月日（1..dim），落空返回 -1。
func (e *datePanelElement) dayAt(lx, ly float64) int {
	gridX := dpPad
	gridY := dpPad + dpHeaderH + dpWeekH
	if lx < gridX || ly < gridY {
		return -1
	}
	col := int((lx - gridX) / dpCell)
	row := int((ly - gridY) / dpCell)
	if col < 0 || col > 6 || row < 0 || row > 5 {
		return -1
	}
	day := row*7 + col - firstWeekday(e.viewYear, e.viewMonth) + 1
	if day < 1 || day > daysInMonth(e.viewYear, e.viewMonth) {
		return -1
	}
	return day
}

func (e *datePanelElement) shiftMonth(delta int) {
	m := int(e.viewMonth) + delta
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

func (e *datePanelElement) shiftYear(delta int) { e.viewYear += delta }

func (e *datePanelElement) Update(newWidget Widget) {
	if np, ok := newWidget.(*datePanel); ok {
		e.panel = np
		e.BaseElement.widget = newWidget
		e.dirty = true
	}
}

// ─── 日期工具 ──────────────────────────────────────────────────

func parseDate(s string) (time.Time, bool) {
	t, err := time.Parse("2006-01-02", s)
	if err != nil {
		return time.Time{}, false
	}
	return t, true
}

// daysInMonth 返回该年月的天数（下月第 0 天 = 本月最后一天）。
func daysInMonth(y int, m time.Month) int {
	return time.Date(y, m+1, 0, 0, 0, 0, 0, time.Local).Day()
}

// firstWeekday 返回该月 1 号是星期几（0=周日 … 6=周六）。
func firstWeekday(y int, m time.Month) int {
	return int(time.Date(y, m, 1, 0, 0, 0, 0, time.Local).Weekday())
}

// DatePanelPreview 返回独立日历面板 widget，仅供 demo/测试预览面板外观
// （正常使用时面板由 DatePickerElement 点击弹出、挂在 Overlay 顶层）。
func DatePanelPreview(selected string) Widget {
	return &datePanel{selected: selected}
}
