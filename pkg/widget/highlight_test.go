package widget

import "testing"

// HighlightCode：行数正确、整行文本无损重建（含缩进）、关键字着色生效、跨行状态传递。
func TestHighlightCode(t *testing.T) {
	rows := HighlightCode("func main() { // hi\n\tx := 42", "go")
	if len(rows) != 2 {
		t.Fatalf("应 2 行，得 %d", len(rows))
	}
	kwColor := ceTokenColor(tkKeyword)
	var line0 string
	sawKeyword := false
	for _, sp := range rows[0] {
		line0 += sp.Text
		if sp.Text == "func" && sp.Color == kwColor {
			sawKeyword = true
		}
	}
	if line0 != "func main() { // hi" {
		t.Errorf("第一行重建文本不符: %q", line0)
	}
	if !sawKeyword {
		t.Error("应有 func 关键字着色 span")
	}
	var line1 string
	for _, sp := range rows[1] {
		line1 += sp.Text
	}
	if line1 != "\tx := 42" {
		t.Errorf("第二行重建文本（应保留前导制表符）不符: %q", line1)
	}
}

// 跨行块注释：第一行 /* 起，第二行仍在注释内。
func TestHighlightCodeBlockComment(t *testing.T) {
	rows := HighlightCode("a /* open\nstill */ b", "go")
	commentColor := ceTokenColor(tkComment)
	// 第二行开头应是注释色
	if len(rows) != 2 || len(rows[1]) == 0 || rows[1][0].Color != commentColor {
		t.Errorf("跨行块注释未正确传递：%+v", rows)
	}
}
