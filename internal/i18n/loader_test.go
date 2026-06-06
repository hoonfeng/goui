package i18n

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

// TestLoadLocaleJSON 验证从 JSON 配置加载语言包（多语言一次加载）。
func TestLoadLocaleJSON(t *testing.T) {
	defer SetLocale("en")
	data := []byte(`{"ja":{"el.empty.description":"データなし","el.select.placeholder":"選択"},"ko":{"el.empty.description":"데이터 없음"}}`)
	if err := LoadLocaleJSON(data); err != nil {
		t.Fatalf("LoadLocaleJSON: %v", err)
	}
	SetLocale("ja")
	if got := T("el.empty.description"); got != "データなし" {
		t.Errorf("ja empty=%q 期望 データなし", got)
	}
	if got := T("el.select.placeholder"); got != "選択" {
		t.Errorf("ja select=%q 期望 選択", got)
	}
	SetLocale("ko")
	if got := T("el.empty.description"); got != "데이터 없음" {
		t.Errorf("ko empty=%q 期望 데이터 없음", got)
	}
	if LoadLocaleJSON([]byte("{bad json")) == nil {
		t.Error("坏 JSON 应报错")
	}
}

// TestLoadLocaleFile 验证从文件/目录加载语言包。
func TestLoadLocaleFile(t *testing.T) {
	defer SetLocale("en")
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "th.json"), []byte(`{"th":{"el.empty.description":"ไม่มีข้อมูล"}}`), 0644)
	if err := LoadLocaleFile(dir); err != nil {
		t.Fatalf("LoadLocaleFile: %v", err)
	}
	SetLocale("th")
	if got := T("el.empty.description"); got != "ไม่มีข้อมูล" {
		t.Errorf("th empty=%q", got)
	}
}

// TestExportLocaleJSON 验证导出语言包（生成翻译模板，往返一致）。
func TestExportLocaleJSON(t *testing.T) {
	Register("xx", map[string]string{"a": "b"})
	data, err := ExportLocaleJSON("xx")
	if err != nil {
		t.Fatalf("ExportLocaleJSON: %v", err)
	}
	var m map[string]map[string]string
	if err := json.Unmarshal(data, &m); err != nil {
		t.Fatalf("导出非法 JSON: %v", err)
	}
	if m["xx"]["a"] != "b" {
		t.Errorf("export 往返失败: %s", data)
	}
}
