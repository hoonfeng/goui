package i18n

import (
	"strings"
	"testing"
	"time"
)

func TestFormatNumberGrouping(t *testing.T) {
	reset()
	SetLocale("en")
	if got := FormatNumber(1234567); !strings.Contains(got, ",") {
		t.Errorf("en FormatNumber(1234567)=%q, expected thousands grouping", got)
	}
}

func TestPluralForms(t *testing.T) {
	reset()
	forms := map[string]string{"one": "{n} item", "other": "{n} items"}

	SetLocale("en")
	if got := Plural(1, forms); got != "1 item" {
		t.Errorf("en plural(1)=%q, want '1 item'", got)
	}
	if got := Plural(3, forms); got != "3 items" {
		t.Errorf("en plural(3)=%q, want '3 items'", got)
	}

	SetLocale("zh-CN") // 中文无单复数变化 → 始终 other
	if got := Plural(1, forms); got != "1 items" {
		t.Errorf("zh plural(1)=%q, want '1 items' (no plural)", got)
	}
}

func TestFormatDate(t *testing.T) {
	reset()
	dt := time.Date(2026, 3, 5, 14, 30, 0, 0, time.UTC)
	cases := []struct {
		loc  Locale
		want string
	}{
		{"zh-CN", "2026年3月5日"},
		{"ja-JP", "2026年3月5日"},
		{"ko-KR", "2026년 3월 5일"},
		{"en", "3/5/2026"},   // 美式 M/D/Y
		{"en-US", "3/5/2026"},
		{"en-GB", "5/3/2026"}, // 英式 D/M/Y
		{"fr-FR", "5/3/2026"},
		{"de-DE", "5/3/2026"},
		{"sv-SE", "2026-03-05"}, // 默认 ISO 8601
	}
	for _, c := range cases {
		SetLocale(c.loc)
		if got := FormatDate(dt); got != c.want {
			t.Errorf("FormatDate[%s]=%q, want %q", c.loc, got, c.want)
		}
	}
}

func TestFormatTime(t *testing.T) {
	reset()
	dt := time.Date(2026, 3, 5, 14, 30, 0, 0, time.UTC)
	SetLocale("en-US")
	if got := FormatTime(dt); got != "2:30 PM" {
		t.Errorf("FormatTime[en-US]=%q, want %q", got, "2:30 PM")
	}
	SetLocale("zh-CN")
	if got := FormatTime(dt); got != "14:30" {
		t.Errorf("FormatTime[zh-CN]=%q, want %q", got, "14:30")
	}
}
