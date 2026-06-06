package widget

import (
	"strings"

	"github.com/user/goui/internal/types"
)

// 公开的只读代码高亮 API：复用编辑器的词法分析内核（codehighlight.go 的 tokenizeLine）+ 主题着色
// （codeeditor_render.go 的 ceTokenColor），供外部按行渲染高亮代码（聊天/文档代码块等只读展示，
// 无行号/光标/编辑）。颜色随 SetTheme(Theme.CodeEditor) 变化。

// HighlightSpan 一段着色文本。
type HighlightSpan struct {
	Text  string
	Color types.Color
}

// HighlightCode 把多行代码按语言做词法高亮，返回逐行的着色 span。
// 跨行状态（块注释 /* */、Go 原始字符串、Markdown ``` 围栏）按行正确传递。
// langName 同 CodeEditor 语言名（"go"/"js"/"python"/"rust"/"markdown"…，经内部解析；未知→go 兜底）。
func HighlightCode(code, langName string) [][]HighlightSpan {
	lang := ceLangFor(langName)
	lines := strings.Split(code, "\n")
	out := make([][]HighlightSpan, 0, len(lines))
	st := stNormal
	for _, line := range lines {
		runes := []rune(line)
		toks, ns := tokenizeLine(runes, lang, st)
		st = ns
		out = append(out, spansFromTokens(runes, toks))
	}
	return out
}

// spansFromTokens 按 token 区间把整行切成着色 span，token 间的空隙补普通文本色 → 重建完整行。
func spansFromTokens(runes []rune, toks []ceToken) []HighlightSpan {
	var spans []HighlightSpan
	col := 0
	for _, tk := range toks {
		if tk.start < col || tk.end > len(runes) || tk.end < tk.start {
			continue // 防御：忽略越界/乱序 token
		}
		if tk.start > col {
			spans = append(spans, HighlightSpan{Text: string(runes[col:tk.start]), Color: ceTokenColor(tkText)})
		}
		if tk.end > tk.start {
			spans = append(spans, HighlightSpan{Text: string(runes[tk.start:tk.end]), Color: ceTokenColor(tk.kind)})
		}
		col = tk.end
	}
	if col < len(runes) {
		spans = append(spans, HighlightSpan{Text: string(runes[col:]), Color: ceTokenColor(tkText)})
	}
	return spans
}
