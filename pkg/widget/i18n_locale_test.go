package widget

import (
	"testing"

	"github.com/hoonfeng/goui/internal/i18n"
)

// TestWidgetI18nLocale 验证内置组件语言包 + orI18n 兜底 + 运行时切换语言。
func TestWidgetI18nLocale(t *testing.T) {
	defer i18n.SetLocale("zh-CN") // 还原默认

	i18n.SetLocale("zh-CN")
	if got := orI18n("", "el.empty.description"); got != "暂无数据" {
		t.Errorf("zh empty=%q 期望「暂无数据」", got)
	}
	if got := orI18n("", "el.select.placeholder"); got != "请选择" {
		t.Errorf("zh select=%q 期望「请选择」", got)
	}
	if got := orI18n("自定义", "el.empty.description"); got != "自定义" {
		t.Errorf("用户显式值应优先于 i18n：%q", got)
	}

	i18n.SetLocale("en")
	if got := orI18n("", "el.empty.description"); got != "No Data" {
		t.Errorf("en empty=%q 期望 No Data", got)
	}
	if got := orI18n("", "el.popconfirm.confirmButtonText"); got != "OK" {
		t.Errorf("en popconfirm confirm=%q 期望 OK", got)
	}

	// 未知语言回退到 fallback(en)
	i18n.SetLocale("fr")
	if got := orI18n("", "el.select.placeholder"); got != "Select" {
		t.Errorf("fr 应回退 en：select=%q 期望 Select", got)
	}
}
