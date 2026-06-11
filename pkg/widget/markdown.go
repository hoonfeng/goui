package widget

import (
	"regexp"
	"strings"

	"github.com/hoonfeng/goui/pkg/types"
)

// ─── Markdown 渲染 ───────────────────────────────────────────────
// parseMarkdown 块级解析(标题/无序·有序列表/引用/代码块/分隔线/段落)→ widget 树。
// 行内 **bold**/*italic*/`code` 标记当前简化为"去标记纯文本"(完整行内混排样式为进阶)。
// MarkdownView 纯渲染展示；Markdown(markdown_editor 思路)= Textarea 编辑 + 实时预览。

var (
	mdBold    = regexp.MustCompile(`\*\*([^*]+)\*\*`)
	mdItalic  = regexp.MustCompile(`\*([^*]+)\*`)
	mdCodeInl = regexp.MustCompile("`([^`]+)`")
	mdOrdered = regexp.MustCompile(`^(\d+)\.\s+(.*)`)
)

// mdInline 去除行内标记(粗体/斜体/行内代码) → 纯文本。
func mdInline(s string) string {
	s = mdBold.ReplaceAllString(s, "$1")
	s = mdItalic.ReplaceAllString(s, "$1")
	s = mdCodeInl.ReplaceAllString(s, "$1")
	return s
}

// parseMarkdown 把 markdown 文本渲染为块级 widget 树。
func parseMarkdown(md string) Widget {
	blocks := []interface{}{Style{FlexDirection: "column", Gap: 8}}

	inCode := false
	var codeLines []string
	flushCode := func() {
		if len(codeLines) > 0 {
			blocks = append(blocks, mdCodeBlock(codeLines))
			codeLines = nil
		}
	}
	for _, raw := range strings.Split(md, "\n") {
		trimmed := strings.TrimSpace(raw)
		if strings.HasPrefix(trimmed, "```") {
			if inCode {
				flushCode()
			}
			inCode = !inCode
			continue
		}
		if inCode {
			codeLines = append(codeLines, raw)
			continue
		}
		if trimmed == "" {
			continue
		}
		switch {
		case trimmed == "---" || trimmed == "***":
			blocks = append(blocks, mdHr())
		case strings.HasPrefix(trimmed, "### "):
			blocks = append(blocks, H3(mdInline(trimmed[4:])))
		case strings.HasPrefix(trimmed, "## "):
			blocks = append(blocks, H2(mdInline(trimmed[3:])))
		case strings.HasPrefix(trimmed, "# "):
			blocks = append(blocks, H1(mdInline(trimmed[2:])))
		case strings.HasPrefix(trimmed, "> "):
			blocks = append(blocks, mdQuote(mdInline(trimmed[2:])))
		case strings.HasPrefix(trimmed, "- "), strings.HasPrefix(trimmed, "* "):
			blocks = append(blocks, mdListItem("•", mdInline(trimmed[2:])))
		default:
			if m := mdOrdered.FindStringSubmatch(trimmed); m != nil {
				blocks = append(blocks, mdListItem(m[1]+".", mdInline(m[2])))
			} else {
				blocks = append(blocks, P(mdInline(trimmed)))
			}
		}
	}
	flushCode()
	return Div(blocks...)
}

func mdHr() Widget {
	return Div(Style{Height: 1, BackgroundColor: types.ColorRef(220, 223, 230)})
}

func mdQuote(text string) Widget {
	return Div(
		Style{FlexDirection: "row", Gap: 8, Padding: types.EdgeInsetsLTRB(0, 2, 0, 2)},
		Div(Style{Width: 3, BackgroundColor: types.ColorRef(220, 223, 230)}),
		NewText(text, elTextSecondary()),
	)
}

func mdListItem(bullet, text string) Widget {
	return Div(
		Style{FlexDirection: "row", Gap: 8},
		NewText(bullet, elTextRegular()),
		NewText(text, elTextRegular()),
	)
}

func mdCodeBlock(lines []string) Widget {
	kids := []interface{}{Style{FlexDirection: "column", Gap: 2, Padding: types.EdgeInsets(10),
		BackgroundColor: types.ColorRef(245, 247, 250), BorderRadius: 4}}
	for _, line := range lines {
		kids = append(kids, NewText(line, elTextRegular()))
	}
	return Div(kids...)
}

// ─── MarkdownView 纯渲染(展示用) ──────────────────────────────────
type MarkdownView struct {
	StatelessWidget
	Source string
}

// NewMarkdownView 创建只读 Markdown 渲染视图。
func NewMarkdownView(md string) *MarkdownView { return &MarkdownView{Source: md} }

func (m *MarkdownView) Build(ctx BuildContext) Widget { return parseMarkdown(m.Source) }
