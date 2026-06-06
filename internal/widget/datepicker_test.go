package widget

import (
	"testing"
	"time"

	"github.com/user/goui/internal/event"
	"github.com/user/goui/internal/layout"
)

func layoutPanel(sel string) *datePanelElement {
	el := (&datePanel{selected: sel}).CreateElement().(*datePanelElement)
	el.Layout(&layout.LayoutContext{Constraints: layout.BoxConstraints{MaxWidth: 1000, MaxHeight: 1000}})
	return el
}

// TestDaysInMonth 含闰年二月。
func TestDaysInMonth(t *testing.T) {
	cases := []struct {
		y    int
		m    time.Month
		want int
	}{
		{2024, time.February, 29}, // 闰年
		{2023, time.February, 28},
		{2024, time.June, 30},
		{2024, time.January, 31},
	}
	for _, c := range cases {
		if got := daysInMonth(c.y, c.m); got != c.want {
			t.Errorf("daysInMonth(%d,%d)=%d want %d", c.y, c.m, got, c.want)
		}
	}
}

// TestFirstWeekday 0=周日…6=周六。
func TestFirstWeekday(t *testing.T) {
	if got := firstWeekday(2024, time.June); got != 6 { // 2024-06-01 周六
		t.Errorf("firstWeekday(2024,6)=%d want 6", got)
	}
	if got := firstWeekday(2024, time.January); got != 1 { // 2024-01-01 周一
		t.Errorf("firstWeekday(2024,1)=%d want 1", got)
	}
}

func TestParseDate(t *testing.T) {
	tm, ok := parseDate("2024-06-15")
	if !ok || tm.Year() != 2024 || tm.Month() != time.June || tm.Day() != 15 {
		t.Errorf("parseDate(2024-06-15)=%v ok=%v", tm, ok)
	}
	if _, ok := parseDate(""); ok {
		t.Error("空串应解析失败")
	}
	if _, ok := parseDate("not-a-date"); ok {
		t.Error("非法串应解析失败")
	}
}

// TestDatePanelShiftMonth 跨年进退。
func TestDatePanelShiftMonth(t *testing.T) {
	el := (&datePanel{selected: "2024-12-15"}).CreateElement().(*datePanelElement)
	el.ensureView()
	el.shiftMonth(1) // 2024-12 → 2025-01
	if el.viewYear != 2025 || el.viewMonth != time.January {
		t.Errorf("shiftMonth 跨年=%d-%d want 2025-1", el.viewYear, el.viewMonth)
	}
	el.shiftMonth(-1) // → 2024-12
	if el.viewYear != 2024 || el.viewMonth != time.December {
		t.Errorf("shiftMonth 回退=%d-%d want 2024-12", el.viewYear, el.viewMonth)
	}
}

// TestDatePanelDayAt 网格坐标命中真实日。
func TestDatePanelDayAt(t *testing.T) {
	el := layoutPanel("2024-06-15")
	gy := dpPad + dpHeaderH + dpWeekH
	// 6/1 在 row0 col6（周六）
	if got := el.dayAt(dpPad+6*dpCell+dpCell/2, gy+dpCell/2); got != 1 {
		t.Errorf("dayAt(col6,row0)=%d want 1(6月1号周六)", got)
	}
	// 15 号：cell=off(6)+14=20 → row2 col6
	if got := el.dayAt(dpPad+6*dpCell+dpCell/2, gy+2*dpCell+dpCell/2); got != 15 {
		t.Errorf("dayAt(15号格)=%d want 15", got)
	}
	// 表头区域应落空
	if got := el.dayAt(dpPad+dpCell/2, dpPad+dpHeaderH/2); got != -1 {
		t.Errorf("dayAt(表头)=%d want -1", got)
	}
}

// TestDatePanelHover MouseMove 命中日期格高亮。
func TestDatePanelHover(t *testing.T) {
	el := layoutPanel("2024-06-15")
	off := el.Offset()
	gy := dpPad + dpHeaderH + dpWeekH
	el.HandleEvent(event.NewMouseEvent(event.TypeMouseMove,
		off.X+dpPad+6*dpCell+dpCell/2, off.Y+gy+dpCell/2, event.ButtonNone, event.ModNone))
	if el.hoverDay != 1 {
		t.Errorf("hoverDay=%d want 1", el.hoverDay)
	}
}
