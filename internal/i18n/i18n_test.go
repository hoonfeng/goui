package i18n

import "testing"

func reset() {
	mu.Lock()
	current = "en"
	fallback = "en"
	catalogs = map[Locale]map[string]string{}
	onChange = nil
	mu.Unlock()
}

func TestTranslateAndFallback(t *testing.T) {
	reset()
	Register("en", map[string]string{"hello": "Hello", "bye": "Bye"})
	Register("zh-CN", map[string]string{"hello": "你好"})

	if got := T("hello"); got != "Hello" {
		t.Errorf("en hello = %q", got)
	}
	SetLocale("zh-CN")
	if got := T("hello"); got != "你好" {
		t.Errorf("zh hello = %q", got)
	}
	if got := T("bye"); got != "Bye" { // zh 缺失 → 回退 en
		t.Errorf("zh bye fallback = %q", got)
	}
	if got := T("missing"); got != "missing" { // 全缺 → 返回 key
		t.Errorf("missing = %q", got)
	}
}

func TestTfAndTn(t *testing.T) {
	reset()
	Register("en", map[string]string{"greet": "Hi, %s!", "named": "{a} + {b}"})
	if got := Tf("greet", "Bob"); got != "Hi, Bob!" {
		t.Errorf("Tf = %q", got)
	}
	if got := Tn("named", map[string]string{"a": "1", "b": "2"}); got != "1 + 2" {
		t.Errorf("Tn = %q", got)
	}
}

func TestSetLocaleTriggersOnChange(t *testing.T) {
	reset()
	fired := 0
	OnChange(func() { fired++ })
	SetLocale("zh-CN")
	if fired != 1 {
		t.Errorf("OnChange fired %d, want 1", fired)
	}
	SetLocale("zh-CN") // 同语言不触发
	if fired != 1 {
		t.Errorf("OnChange fired %d on same locale, want 1", fired)
	}
}

func TestRTLDetection(t *testing.T) {
	if !LocaleIsRTL("ar") || !LocaleIsRTL("he-IL") || !LocaleIsRTL("fa") {
		t.Error("RTL locales should be detected")
	}
	if LocaleIsRTL("en") || LocaleIsRTL("zh-CN") {
		t.Error("LTR locales should not be RTL")
	}
}
