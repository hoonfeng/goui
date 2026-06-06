package i18n

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
)

// 从 JSON 配置加载语言包——「按需加载，不改代码」。以后要加日语/韩语等，
// 写一份 JSON（或放一个目录）丢进来即可，无需重新编译。结构为 {语言: {key: 翻译}}：
//
//	{
//	  "ja": {"el.empty.description": "データなし", "el.select.placeholder": "選択してください"},
//	  "ko": {"el.empty.description": "데이터 없음"}
//	}
//
// 多次加载累加/覆盖（同 Register 的合并语义），可只覆盖个别 key。

// LoadLocaleJSON 从 JSON 字节加载语言包（结构 {语言: {key: 翻译}}）。
func LoadLocaleJSON(data []byte) error {
	var m map[string]map[string]string
	if err := json.Unmarshal(data, &m); err != nil {
		return err
	}
	for loc, msgs := range m {
		Register(Locale(loc), msgs)
	}
	return nil
}

// LoadLocaleFile 从 JSON 文件加载语言包；若 path 是目录，则加载其中所有 *.json。
func LoadLocaleFile(path string) error {
	info, err := os.Stat(path)
	if err != nil {
		return err
	}
	if !info.IsDir() {
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		return LoadLocaleJSON(data)
	}
	entries, err := os.ReadDir(path)
	if err != nil {
		return err
	}
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".json") {
			continue
		}
		if data, err := os.ReadFile(filepath.Join(path, e.Name())); err == nil {
			LoadLocaleJSON(data) // 单个文件出错不阻断其余
		}
	}
	return nil
}

// ExportLocaleJSON 把某语言当前已注册的全部翻译导出为 JSON（便于生成模板、再翻译、回灌）。
func ExportLocaleJSON(loc Locale) ([]byte, error) {
	mu.RLock()
	m := catalogs[loc]
	out := make(map[string]string, len(m))
	for k, v := range m {
		out[k] = v
	}
	mu.RUnlock()
	return json.MarshalIndent(map[string]map[string]string{string(loc): out}, "", "  ")
}
