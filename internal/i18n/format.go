package i18n

import (
	"fmt"
	"strings"
	"time"

	"golang.org/x/text/language"
	"golang.org/x/text/message"
	"golang.org/x/text/number"
)

// langTag 把当前 Locale 解析为 language.Tag（解析失败回退英语）。
func langTag() language.Tag {
	t, err := language.Parse(string(CurrentLocale()))
	if err != nil {
		return language.English
	}
	return t
}

// FormatNumber 按当前语言格式化数字（千分位分组等本地化规则）。
func FormatNumber(n interface{}) string {
	return message.NewPrinter(langTag()).Sprint(number.Decimal(n))
}

// FormatPercent 按当前语言格式化百分比（传入 0~1 的比例）。
func FormatPercent(f float64) string {
	return message.NewPrinter(langTag()).Sprint(number.Percent(f))
}

// Plural 根据数量选择复数形式并替换其中的 {n}（用本地化数字）。
// forms 至少包含 "other"；英语等区分单复数的语言可另给 "one"。
// 无复数变化的语言（中文、日文、韩文等）始终使用 "other"。
func Plural(count int, forms map[string]string) string {
	key := "other"
	if count == 1 && !isNoPluralLocale(CurrentLocale()) {
		key = "one"
	}
	s, ok := forms[key]
	if !ok {
		s = forms["other"]
	}
	return strings.ReplaceAll(s, "{n}", FormatNumber(count))
}

func isNoPluralLocale(loc Locale) bool {
	s := strings.ToLower(string(loc))
	for _, p := range []string{"zh", "ja", "ko", "th", "vi", "id", "ms"} {
		if s == p || strings.HasPrefix(s, p+"-") {
			return true
		}
	}
	return false
}

// FormatDate 按当前语言区域以数字形式格式化日期（年月日顺序因地区而异）。
func FormatDate(t time.Time) string {
	y, m, d := t.Year(), int(t.Month()), t.Day()
	loc := strings.ToLower(string(CurrentLocale()))
	switch {
	case strings.HasPrefix(loc, "zh") || strings.HasPrefix(loc, "ja"):
		return fmt.Sprintf("%d年%d月%d日", y, m, d)
	case strings.HasPrefix(loc, "ko"):
		return fmt.Sprintf("%d년 %d월 %d일", y, m, d)
	case loc == "en" || strings.HasPrefix(loc, "en-us"):
		return fmt.Sprintf("%d/%d/%d", m, d, y) // 美式 M/D/Y
	case isDMYLocale(loc):
		return fmt.Sprintf("%d/%d/%d", d, m, y) // 多数地区 D/M/Y
	default:
		return fmt.Sprintf("%04d-%02d-%02d", y, m, d) // ISO 8601
	}
}

// FormatTime 按地区惯例格式化时间（美式 12 小时制，其余 24 小时制）。
func FormatTime(t time.Time) string {
	loc := strings.ToLower(string(CurrentLocale()))
	if loc == "en" || strings.HasPrefix(loc, "en-us") {
		return t.Format("3:04 PM")
	}
	return t.Format("15:04")
}

func isDMYLocale(loc string) bool {
	for _, p := range []string{"en-gb", "fr", "de", "es", "it", "pt", "ru", "ar", "he", "nl", "pl", "tr"} {
		if loc == p || strings.HasPrefix(loc, p+"-") {
			return true
		}
	}
	return false
}
