package i18n

import (
	"fmt"
	"strings"
)

// 日历本地化：月份名、星期名（供 DatePicker/Calendar 等组件）。
// 中日韩用「N月」「星期X」形式；其余语言用英文名（可后续按需扩展更多语言）。

var (
	monthNamesEn   = []string{"January", "February", "March", "April", "May", "June", "July", "August", "September", "October", "November", "December"}
	monthShortEn   = []string{"Jan", "Feb", "Mar", "Apr", "May", "Jun", "Jul", "Aug", "Sep", "Oct", "Nov", "Dec"}
	weekdayNamesEn = []string{"Sunday", "Monday", "Tuesday", "Wednesday", "Thursday", "Friday", "Saturday"}
	weekdayShortEn = []string{"Sun", "Mon", "Tue", "Wed", "Thu", "Fri", "Sat"}
)

// MonthName 返回月份(1-12)的本地化全名（越界返回 ""）。
func MonthName(m int) string {
	if m < 1 || m > 12 {
		return ""
	}
	loc := strings.ToLower(string(CurrentLocale()))
	switch {
	case strings.HasPrefix(loc, "zh"), strings.HasPrefix(loc, "ja"):
		return fmt.Sprintf("%d月", m)
	case strings.HasPrefix(loc, "ko"):
		return fmt.Sprintf("%d월", m)
	default:
		return monthNamesEn[m-1]
	}
}

// MonthShort 返回月份(1-12)的本地化缩写。
func MonthShort(m int) string {
	if m < 1 || m > 12 {
		return ""
	}
	loc := strings.ToLower(string(CurrentLocale()))
	switch {
	case strings.HasPrefix(loc, "zh"), strings.HasPrefix(loc, "ja"):
		return fmt.Sprintf("%d月", m)
	case strings.HasPrefix(loc, "ko"):
		return fmt.Sprintf("%d월", m)
	default:
		return monthShortEn[m-1]
	}
}

// WeekdayName 返回星期(0=周日 .. 6=周六)的本地化全名。
func WeekdayName(w int) string {
	w = ((w % 7) + 7) % 7
	loc := strings.ToLower(string(CurrentLocale()))
	switch {
	case strings.HasPrefix(loc, "zh"):
		return []string{"星期日", "星期一", "星期二", "星期三", "星期四", "星期五", "星期六"}[w]
	case strings.HasPrefix(loc, "ja"):
		return []string{"日曜日", "月曜日", "火曜日", "水曜日", "木曜日", "金曜日", "土曜日"}[w]
	case strings.HasPrefix(loc, "ko"):
		return []string{"일요일", "월요일", "화요일", "수요일", "목요일", "금요일", "토요일"}[w]
	default:
		return weekdayNamesEn[w]
	}
}

// WeekdayShort 返回星期(0=周日 .. 6=周六)的本地化缩写（日历表头用）。
func WeekdayShort(w int) string {
	w = ((w % 7) + 7) % 7
	loc := strings.ToLower(string(CurrentLocale()))
	switch {
	case strings.HasPrefix(loc, "zh"):
		return []string{"日", "一", "二", "三", "四", "五", "六"}[w]
	case strings.HasPrefix(loc, "ja"):
		return []string{"日", "月", "火", "水", "木", "金", "土"}[w]
	case strings.HasPrefix(loc, "ko"):
		return []string{"일", "월", "화", "수", "목", "금", "토"}[w]
	default:
		return weekdayShortEn[w]
	}
}
