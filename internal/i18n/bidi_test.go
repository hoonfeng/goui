package i18n

import "testing"

// TestBidiReorder 验证双向文本重排（纯 LTR 原样、RTL 处理、reverseRunes）。
func TestBidiReorder(t *testing.T) {
	if got := BidiReorder("hello 123"); got != "hello 123" {
		t.Errorf("纯 LTR 应原样，得 %q", got)
	}
	heb := "אבג" // אבג（希伯来，RTL）
	// 纯 RTL：视觉顺序应为字符逆序
	if got := BidiReorder(heb); got != "גבא" {
		t.Errorf("纯 RTL 应逆序，得 %q", got)
	}
	// 混排不崩、非空
	if got := BidiReorder("a " + heb + " b"); got == "" {
		t.Error("混排不应空")
	}
	if reverseRunes("abc") != "cba" {
		t.Error("reverseRunes 错")
	}
}
