package i18n

import "testing"

// TestCalendar 验证月份名/星期名按语言本地化。
func TestCalendar(t *testing.T) {
	defer SetLocale("en")

	SetLocale("en")
	if MonthName(1) != "January" {
		t.Errorf("en MonthName(1)=%q 期望 January", MonthName(1))
	}
	if MonthShort(12) != "Dec" {
		t.Errorf("en MonthShort(12)=%q 期望 Dec", MonthShort(12))
	}
	if WeekdayShort(1) != "Mon" {
		t.Errorf("en WeekdayShort(1)=%q 期望 Mon", WeekdayShort(1))
	}
	if WeekdayName(0) != "Sunday" {
		t.Errorf("en WeekdayName(0)=%q 期望 Sunday", WeekdayName(0))
	}

	SetLocale("zh-CN")
	if MonthName(1) != "1月" {
		t.Errorf("zh MonthName(1)=%q 期望 1月", MonthName(1))
	}
	if WeekdayName(0) != "星期日" {
		t.Errorf("zh WeekdayName(0)=%q 期望 星期日", WeekdayName(0))
	}
	if WeekdayShort(6) != "六" {
		t.Errorf("zh WeekdayShort(6)=%q 期望 六", WeekdayShort(6))
	}

	// 越界
	if MonthName(0) != "" || MonthName(13) != "" {
		t.Error("越界月份应返回空")
	}
	// 星期取模回绕
	if WeekdayName(7) != WeekdayName(0) {
		t.Error("WeekdayName 应对 7 回绕到 0")
	}
}
