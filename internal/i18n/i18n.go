// Package i18n 提供 goui 的多语言（国际化）支持：语言区域、消息目录、
// 翻译查找、运行时语言切换（切换后触发界面重建），以及书写方向（RTL）判断。
//
// 典型用法：
//
//	i18n.Register("en", map[string]string{"hello": "Hello, %s!"})
//	i18n.Register("zh-CN", map[string]string{"hello": "你好，%s！"})
//	i18n.SetLocale("zh-CN")
//	label := i18n.Tf("hello", name) // → "你好，<name>！"
//
// 在组件 Build() 中调用 i18n.T/Tf/Tn 即可；SetLocale 后框架会重建界面，
// 使所有翻译刷新为新语言。
package i18n

import (
	"fmt"
	"strings"
	"sync"
)

// Locale 表示一个语言区域标识，如 "en"、"zh-CN"、"ar"、"ja"。
type Locale string

var (
	mu       sync.RWMutex
	current  Locale = "en"
	fallback Locale = "en"
	catalogs        = map[Locale]map[string]string{}
	onChange func()
)

// Register 注册/合并某个语言的翻译表（key → 翻译模板）。可多次调用累加。
func Register(loc Locale, messages map[string]string) {
	mu.Lock()
	defer mu.Unlock()
	m := catalogs[loc]
	if m == nil {
		m = make(map[string]string, len(messages))
		catalogs[loc] = m
	}
	for k, v := range messages {
		m[k] = v
	}
}

// SetFallback 设置找不到翻译时回退的语言（默认 "en"）。
func SetFallback(loc Locale) {
	mu.Lock()
	fallback = loc
	mu.Unlock()
}

// SetLocale 切换当前语言；若发生变化，触发 OnChange 回调（通常驱动界面重建）。
func SetLocale(loc Locale) {
	mu.Lock()
	changed := current != loc
	current = loc
	cb := onChange
	mu.Unlock()
	if changed && cb != nil {
		cb()
	}
}

// CurrentLocale 返回当前语言。
func CurrentLocale() Locale {
	mu.RLock()
	defer mu.RUnlock()
	return current
}

// OnChange 注册语言切换回调（由 app 连接到界面重建）。
func OnChange(fn func()) {
	mu.Lock()
	onChange = fn
	mu.Unlock()
}

func lookup(key string) string {
	mu.RLock()
	defer mu.RUnlock()
	if m, ok := catalogs[current]; ok {
		if v, ok := m[key]; ok {
			return v
		}
	}
	if current != fallback {
		if m, ok := catalogs[fallback]; ok {
			if v, ok := m[key]; ok {
				return v
			}
		}
	}
	return key // 找不到回退到 key 本身，便于开发期发现缺失
}

// T 返回 key 在当前语言下的翻译（找不到时回退语言，再找不到返回 key 本身）。
func T(key string) string { return lookup(key) }

// Tf 翻译并以 printf 风格格式化参数（模板使用 %s/%d 等）。
func Tf(key string, args ...interface{}) string {
	return fmt.Sprintf(lookup(key), args...)
}

// Tn 翻译并按命名占位符 {name} 替换（模板使用 {name}）。
func Tn(key string, vars map[string]string) string {
	s := lookup(key)
	for k, v := range vars {
		s = strings.ReplaceAll(s, "{"+k+"}", v)
	}
	return s
}

// IsRTL 返回当前语言是否从右到左书写（阿拉伯、希伯来、波斯、乌尔都等）。
func IsRTL() bool { return LocaleIsRTL(CurrentLocale()) }

// LocaleIsRTL 判断指定语言是否 RTL。
func LocaleIsRTL(loc Locale) bool {
	s := strings.ToLower(string(loc))
	for _, p := range []string{"ar", "he", "fa", "ur", "ps", "sd", "yi", "dv", "ku"} {
		if s == p || strings.HasPrefix(s, p+"-") {
			return true
		}
	}
	return false
}
