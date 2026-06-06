package i18n

import (
	"strings"

	"golang.org/x/text/unicode/bidi"
)

// BidiReorder 把含双向文字（阿拉伯/希伯来 与 拉丁/数字 混排）的一段文本，
// 按 Unicode 双向算法重排为「视觉顺序」字符串，供从左到右逐字绘制时正确显示。
// 纯 LTR 文本走快速路径原样返回。
//
// 适用于只读文本展示（如标签 / 提示）；可编辑文本的完整 BiDi（光标 logical↔visual
// 映射、行内重排与测量同步）是更大的工程，此处只提供段落级视觉重排能力。
func BidiReorder(text string) string {
	if !hasRTL(text) {
		return text
	}
	var p bidi.Paragraph
	if _, err := p.SetString(text); err != nil {
		return text
	}
	ord, err := p.Order()
	if err != nil {
		return text
	}
	var sb strings.Builder
	for i := 0; i < ord.NumRuns(); i++ {
		r := ord.Run(i)
		s := r.String()
		if r.Direction() == bidi.RightToLeft {
			s = reverseRunes(s) // RTL run 字符逆序，使其在 LTR 绘制下读序正确
		}
		sb.WriteString(s)
	}
	return sb.String()
}

func reverseRunes(s string) string {
	rs := []rune(s)
	for i, j := 0, len(rs)-1; i < j; i, j = i+1, j-1 {
		rs[i], rs[j] = rs[j], rs[i]
	}
	return string(rs)
}

// hasRTL 是否含 RTL 文字（希伯来 0590–05FF / 阿拉伯 0600–08FF 及表现形式区）。
func hasRTL(s string) bool {
	for _, r := range s {
		if (r >= 0x0590 && r <= 0x08FF) || (r >= 0xFB1D && r <= 0xFDFF) || (r >= 0xFE70 && r <= 0xFEFF) {
			return true
		}
	}
	return false
}
