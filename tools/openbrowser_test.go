package tools

import (
	"strings"
	"testing"
)

func TestOpenBrowser(t *testing.T) {
	// 测试 URL 为空的情况
	_, err := OpenBrowser("")
	if err == nil {
		t.Error("OpenBrowser should fail on empty URL")
	}

	// 测试有效 URL（不验证实际打开，只验证函数返回逻辑）
	result, err := OpenBrowser("https://example.com")
	if err != nil {
		// 在某些无 GUI 环境（如 CI）可能无法打开浏览器，这是预期行为
		t.Logf("OpenBrowser returned error (expected in headless env): %v", err)
	} else {
		if !strings.Contains(result, "已打开浏览器") {
			t.Errorf("OpenBrowser unexpected response: %s", result)
		}
	}
}
